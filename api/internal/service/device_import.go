package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	"openfms/api/internal/model"
)

// DeviceImportService 设备导入服务
type DeviceImportService struct {
	db            *gorm.DB
	deviceService *DeviceService
	tasks         map[string]*model.DeviceImportResult
	tasksMu       sync.RWMutex
}

// NewDeviceImportService 创建设备导入服务
func NewDeviceImportService(db *gorm.DB, deviceService *DeviceService) *DeviceImportService {
	return &DeviceImportService{
		db:            db,
		deviceService: deviceService,
		tasks:         make(map[string]*model.DeviceImportResult),
	}
}

// GenerateImportTemplate 生成导入模板Excel文件
func (s *DeviceImportService) GenerateImportTemplate() (*bytes.Buffer, error) {
	f := excelize.NewFile()
	defer f.Close()

	// 设置工作表名称
	sheetName := "设备导入模板"
	f.SetSheetName("Sheet1", sheetName)

	// 获取模板列定义
	columns := model.GetDeviceImportTemplateColumns()

	// 写入表头
	for i, col := range columns {
		cell := fmt.Sprintf("%c1", 'A'+i)
		header := col.Name
		if col.Required {
			header += "*"
		}
		f.SetCellValue(sheetName, cell, header)
	}

	// 写入示例数据
	for i, col := range columns {
		cell := fmt.Sprintf("%c2", 'A'+i)
		f.SetCellValue(sheetName, cell, col.Example)
	}

	// 设置列宽
	for i := range columns {
		col := string('A' + i)
		f.SetColWidth(sheetName, col, col, 20)
	}

	// 添加说明工作表
	f.NewSheet("填写说明")
	f.SetCellValue("填写说明", "A1", "字段说明")
	f.SetCellValue("填写说明", "A2", "字段名")
	f.SetCellValue("填写说明", "B2", "必填")
	f.SetCellValue("填写说明", "C2", "说明")
	f.SetCellValue("填写说明", "D2", "示例")

	for i, col := range columns {
		row := i + 3
		f.SetCellValue("填写说明", fmt.Sprintf("A%d", row), col.Name)
		required := "否"
		if col.Required {
			required = "是"
		}
		f.SetCellValue("填写说明", fmt.Sprintf("B%d", row), required)
		f.SetCellValue("填写说明", fmt.Sprintf("C%d", row), col.Description)
		f.SetCellValue("填写说明", fmt.Sprintf("D%d", row), col.Example)
	}

	// 设置说明工作表列宽
	f.SetColWidth("填写说明", "A", "A", 15)
	f.SetColWidth("填写说明", "B", "B", 10)
	f.SetColWidth("填写说明", "C", "C", 50)
	f.SetColWidth("填写说明", "D", "D", 20)

	// 写入缓冲区
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}

	return &buf, nil
}

// ParseExcel 解析Excel文件
func (s *DeviceImportService) ParseExcel(reader io.Reader) ([]model.DeviceImportRow, error) {
	f, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, fmt.Errorf("无法读取Excel文件: %w", err)
	}
	defer f.Close()

	// 获取第一个工作表
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("Excel文件没有工作表")
	}

	// 优先使用"设备导入模板"工作表
	sheetName := sheets[0]
	for _, name := range sheets {
		if name == "设备导入模板" {
			sheetName = name
			break
		}
	}

	// 读取所有行
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("无法读取工作表: %w", err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("Excel文件数据不足，至少需要包含表头和一行数据")
	}

	// 解析列索引
	headerMap := make(map[string]int)
	for i, cell := range rows[0] {
		// 去除必填标记 *
		cell = removeRequiredMark(cell)
		headerMap[cell] = i
	}

	// 验证必需列
	requiredColumns := []string{"设备号", "设备名称", "协议类型"}
	for _, col := range requiredColumns {
		if _, ok := headerMap[col]; !ok {
			return nil, fmt.Errorf("缺少必需列: %s", col)
		}
	}

	// 解析数据行
	var devices []model.DeviceImportRow
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) == 0 || isEmptyRow(row) {
			continue
		}

		device := model.DeviceImportRow{
			RowNum: i + 1,
		}

		if idx, ok := headerMap["设备号"]; ok && idx < len(row) {
			device.DeviceID = row[idx]
		}
		if idx, ok := headerMap["设备名称"]; ok && idx < len(row) {
			device.Name = row[idx]
		}
		if idx, ok := headerMap["协议类型"]; ok && idx < len(row) {
			device.Protocol = row[idx]
		}
		if idx, ok := headerMap["SIM卡号"]; ok && idx < len(row) {
			device.SIMCard = row[idx]
		}
		if idx, ok := headerMap["绑定车牌"]; ok && idx < len(row) {
			device.VehiclePlate = row[idx]
		}
		if idx, ok := headerMap["状态"]; ok && idx < len(row) {
			device.Status = row[idx]
		}
		if idx, ok := headerMap["备注"]; ok && idx < len(row) {
			device.Remark = row[idx]
		}

		devices = append(devices, device)
	}

	return devices, nil
}

// ValidateRows 验证导入数据
func (s *DeviceImportService) ValidateRows(rows []model.DeviceImportRow) []model.DeviceImportRow {
	// 用于检查设备号重复
	deviceIDMap := make(map[string]int)

	for i := range rows {
		row := &rows[i]
		var errors []string

		// 验证设备号
		if row.DeviceID == "" {
			errors = append(errors, "设备号不能为空")
		} else {
			// 检查设备号格式（只允许字母、数字、下划线、横线）
			if !isValidDeviceID(row.DeviceID) {
				errors = append(errors, "设备号格式不正确，只允许字母、数字、下划线、横线")
			}
			// 检查重复
			if prevRow, ok := deviceIDMap[row.DeviceID]; ok {
				errors = append(errors, fmt.Sprintf("设备号与第%d行重复", prevRow))
			} else {
				deviceIDMap[row.DeviceID] = row.RowNum
			}
			// 检查数据库中是否已存在
			var count int64
			s.db.Model(&model.Device{}).Where("device_id = ?", row.DeviceID).Count(&count)
			if count > 0 {
				errors = append(errors, "设备号已存在")
			}
		}

		// 验证设备名称
		if row.Name == "" {
			errors = append(errors, "设备名称不能为空")
		} else if len(row.Name) > 100 {
			errors = append(errors, "设备名称长度不能超过100个字符")
		}

		// 验证协议类型
		if row.Protocol == "" {
			errors = append(errors, "协议类型不能为空")
		} else if !model.ValidProtocols[row.Protocol] {
			errors = append(errors, "协议类型无效，支持: JT808, GT06, Wialon")
		}

		// 验证车牌号（如果提供）
		if row.VehiclePlate != "" {
			var count int64
			s.db.Model(&model.Vehicle{}).Where("plate_number = ?", row.VehiclePlate).Count(&count)
			if count == 0 {
				errors = append(errors, fmt.Sprintf("车牌号 '%s' 不存在", row.VehiclePlate))
			}
		}

		if len(errors) > 0 {
			row.Error = joinErrors(errors)
		}
	}

	return rows
}

// ImportDevices 导入设备（异步）
func (s *DeviceImportService) ImportDevices(ctx context.Context, taskID string, rows []model.DeviceImportRow, createdBy uint) {
	// 初始化任务状态
	result := &model.DeviceImportResult{
		TaskID:       taskID,
		Status:       "processing",
		TotalCount:   len(rows),
		SuccessCount: 0,
		ErrorCount:   0,
		Progress:     0,
	}

	s.tasksMu.Lock()
	s.tasks[taskID] = result
	s.tasksMu.Unlock()

	// 在后台处理导入
	go s.processImport(ctx, taskID, rows, createdBy)
}

// processImport 处理导入任务
func (s *DeviceImportService) processImport(ctx context.Context, taskID string, rows []model.DeviceImportRow, createdBy uint) {
	total := len(rows)
	successCount := 0
	errorCount := 0
	var importErrors []model.DeviceImportError

	// 批量处理，每100条提交一次
	batchSize := 100

	for i := 0; i < total; i += batchSize {
		end := i + batchSize
		if end > total {
			end = total
		}

		batch := rows[i:end]

		// 使用事务处理每一批
		err := s.db.Transaction(func(tx *gorm.DB) error {
			for _, row := range batch {
				// 如果有验证错误，跳过
				if row.Error != "" {
					errorCount++
					importErrors = append(importErrors, model.DeviceImportError{
						RowNum: row.RowNum,
						Error:  row.Error,
					})
					continue
				}

				// 创建设备
				device := &model.Device{
					DeviceID: row.DeviceID,
					Name:     row.Name,
					Protocol: row.Protocol,
					Status:   model.ParseStatus(row.Status),
				}

				// 如果有车牌号，查找车辆ID
				if row.VehiclePlate != "" {
					var vehicle model.Vehicle
					if err := tx.Where("plate_number = ?", row.VehiclePlate).First(&vehicle).Error; err == nil {
						device.VehicleID = &vehicle.ID
					}
				}

				if err := tx.Create(device).Error; err != nil {
					errorCount++
					importErrors = append(importErrors, model.DeviceImportError{
						RowNum: row.RowNum,
						Error:  fmt.Sprintf("数据库错误: %v", err),
					})
					continue
				}

				successCount++
			}
			return nil
		})

		if err != nil {
			// 事务错误，记录到错误中
			for _, row := range batch {
				if row.Error == "" {
					errorCount++
					importErrors = append(importErrors, model.DeviceImportError{
						RowNum: row.RowNum,
						Error:  fmt.Sprintf("批量处理错误: %v", err),
					})
				}
			}
		}

		// 更新进度
		progress := int(float64(end) / float64(total) * 100)
		s.tasksMu.Lock()
		if task, ok := s.tasks[taskID]; ok {
			task.Progress = progress
			task.SuccessCount = successCount
			task.ErrorCount = errorCount
		}
		s.tasksMu.Unlock()

		// 小延迟，避免数据库压力过大
		time.Sleep(10 * time.Millisecond)
	}

	// 完成任务
	s.tasksMu.Lock()
	if task, ok := s.tasks[taskID]; ok {
		task.Status = "completed"
		task.SuccessCount = successCount
		task.ErrorCount = errorCount
		if len(importErrors) > 0 {
			// 只保留前50个错误
			if len(importErrors) > 50 {
				task.Errors = importErrors[:50]
			} else {
				task.Errors = importErrors
			}
		}
	}
	s.tasksMu.Unlock()

	// 保存任务到数据库
	taskRecord := &model.DeviceImportTask{
		TaskID:       taskID,
		Status:       "completed",
		TotalCount:   total,
		SuccessCount: successCount,
		ErrorCount:   errorCount,
		CreatedBy:    createdBy,
	}
	if len(importErrors) > 0 {
		errorsMap := make(model.JSONMap)
		for i, err := range importErrors {
			if i >= 50 {
				break
			}
			key := fmt.Sprintf("row_%d", err.RowNum)
			errorsMap[key] = err.Error
		}
		taskRecord.Errors = errorsMap
	}
	now := time.Now()
	taskRecord.CompletedAt = &now
	s.db.Create(taskRecord)
}

// GetImportResult 获取导入结果
func (s *DeviceImportService) GetImportResult(taskID string) (*model.DeviceImportResult, bool) {
	s.tasksMu.RLock()
	defer s.tasksMu.RUnlock()

	result, ok := s.tasks[taskID]
	return result, ok
}

// PreviewImport 预览导入数据（不实际导入）
func (s *DeviceImportService) PreviewImport(rows []model.DeviceImportRow) map[string]interface{} {
	validated := s.ValidateRows(rows)

	validCount := 0
	invalidCount := 0
	var previewData []map[string]interface{}

	for _, row := range validated {
		item := map[string]interface{}{
			"row_num":       row.RowNum,
			"device_id":     row.DeviceID,
			"name":          row.Name,
			"protocol":      row.Protocol,
			"sim_card":      row.SIMCard,
			"vehicle_plate": row.VehiclePlate,
			"status":        row.Status,
			"remark":        row.Remark,
			"valid":         row.Error == "",
		}
		if row.Error != "" {
			item["error"] = row.Error
			invalidCount++
		} else {
			validCount++
		}
		previewData = append(previewData, item)
	}

	// 只返回前20条预览
	if len(previewData) > 20 {
		previewData = previewData[:20]
	}

	return map[string]interface{}{
		"total":       len(rows),
		"valid_count": validCount,
		"invalid_count": invalidCount,
		"preview":     previewData,
	}
}

// Helper functions

func removeRequiredMark(s string) string {
	if len(s) > 0 && s[len(s)-1] == '*' {
		return s[:len(s)-1]
	}
	return s
}

func isEmptyRow(row []string) bool {
	for _, cell := range row {
		if cell != "" {
			return false
		}
	}
	return true
}

func isValidDeviceID(id string) bool {
	for _, c := range id {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			return false
		}
	}
	return len(id) > 0 && len(id) <= 32
}

func joinErrors(errors []string) string {
	result := ""
	for i, err := range errors {
		if i > 0 {
			result += "; "
		}
		result += err
	}
	return result
}

// GenerateErrorReport 生成错误报告Excel
func (s *DeviceImportService) GenerateErrorReport(rows []model.DeviceImportRow) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "导入错误报告"
	f.SetSheetName("Sheet1", sheetName)

	// 写入表头
	headers := []string{"行号", "设备号", "设备名称", "协议类型", "SIM卡号", "绑定车牌", "状态", "备注", "错误信息"}
	for i, header := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetName, cell, header)
	}

	// 写入错误数据
	rowNum := 2
	for _, row := range rows {
		if row.Error != "" {
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowNum), row.RowNum)
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowNum), row.DeviceID)
			f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowNum), row.Name)
			f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowNum), row.Protocol)
			f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowNum), row.SIMCard)
			f.SetCellValue(sheetName, fmt.Sprintf("F%d", rowNum), row.VehiclePlate)
			f.SetCellValue(sheetName, fmt.Sprintf("G%d", rowNum), row.Status)
			f.SetCellValue(sheetName, fmt.Sprintf("H%d", rowNum), row.Remark)
			f.SetCellValue(sheetName, fmt.Sprintf("I%d", rowNum), row.Error)
			rowNum++
		}
	}

	// 设置列宽
	f.SetColWidth(sheetName, "A", "A", 10)
	f.SetColWidth(sheetName, "B", "H", 20)
	f.SetColWidth(sheetName, "I", "I", 50)

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}

	return &buf, nil
}

// CleanupOldTasks 清理旧任务（防止内存泄漏）
func (s *DeviceImportService) CleanupOldTasks(maxAge time.Duration) {
	s.tasksMu.Lock()
	defer s.tasksMu.Unlock()

	// 注意：这里简化处理，实际应该记录任务创建时间
	// 这里只保留最近的100个任务
	if len(s.tasks) > 100 {
		newTasks := make(map[string]*model.DeviceImportResult)
		count := 0
		for k, v := range s.tasks {
			if count >= 100 {
				break
			}
			newTasks[k] = v
			count++
		}
		s.tasks = newTasks
	}
}

// parseInt 安全地解析整数
func parseInt(s string) int {
	if s == "" {
		return 0
	}
	i, _ := strconv.Atoi(s)
	return i
}
