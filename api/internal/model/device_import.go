package model

import (
	"time"
)

// DeviceImportTask 设备导入任务
type DeviceImportTask struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	TaskID       string    `json:"task_id" gorm:"uniqueIndex;size:64"`
	Status       string    `json:"status" gorm:"size:20"` // pending, processing, completed, failed
	TotalCount   int       `json:"total_count"`
	SuccessCount int       `json:"success_count"`
	ErrorCount   int       `json:"error_count"`
	Errors       JSONMap   `json:"errors" gorm:"type:jsonb"`
	FileURL      string    `json:"file_url" gorm:"size:255"`
	CreatedBy    uint      `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	CompletedAt  *time.Time `json:"completed_at"`
}

// DeviceImportRow Excel导入的单个设备记录
type DeviceImportRow struct {
	RowNum      int    `json:"row_num"`
	DeviceID    string `json:"device_id"`
	Name        string `json:"name"`
	Protocol    string `json:"protocol"`
	SIMCard     string `json:"sim_card"`
	VehiclePlate string `json:"vehicle_plate"`
	Status      string `json:"status"`
	Remark      string `json:"remark"`
	Error       string `json:"error,omitempty"`
}

// DeviceImportResult 导入结果
type DeviceImportResult struct {
	TaskID       string             `json:"task_id"`
	Status       string             `json:"status"`
	TotalCount   int                `json:"total_count"`
	SuccessCount int                `json:"success_count"`
	ErrorCount   int                `json:"error_count"`
	Errors       []DeviceImportError `json:"errors,omitempty"`
	Progress     int                `json:"progress"` // 0-100
}

// DeviceImportError 导入错误详情
type DeviceImportError struct {
	RowNum int    `json:"row_num"`
	Field  string `json:"field,omitempty"`
	Value  string `json:"value,omitempty"`
	Error  string `json:"error"`
}

// DeviceImportTemplateColumn 导入模板列定义
type DeviceImportTemplateColumn struct {
	Name        string `json:"name"`
	Key         string `json:"key"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
	Example     string `json:"example"`
}

// GetDeviceImportTemplateColumns 获取导入模板列定义
func GetDeviceImportTemplateColumns() []DeviceImportTemplateColumn {
	return []DeviceImportTemplateColumn{
		{
			Name:        "设备号",
			Key:         "device_id",
			Required:    true,
			Description: "设备唯一标识，通常是SIM卡号或设备IMEI",
			Example:     "13912345678",
		},
		{
			Name:        "设备名称",
			Key:         "name",
			Required:    true,
			Description: "设备的显示名称",
			Example:     "车辆01",
		},
		{
			Name:        "协议类型",
			Key:         "protocol",
			Required:    true,
			Description: "设备通信协议: JT808, GT06, Wialon",
			Example:     "JT808",
		},
		{
			Name:        "SIM卡号",
			Key:         "sim_card",
			Required:    false,
			Description: "SIM卡号码（可选，如与设备号不同）",
			Example:     "13912345678",
		},
		{
			Name:        "绑定车牌",
			Key:         "vehicle_plate",
			Required:    false,
			Description: "绑定的车辆车牌号（可选）",
			Example:     "京A12345",
		},
		{
			Name:        "状态",
			Key:         "status",
			Required:    false,
			Description: "设备状态: 启用/禁用，默认为启用",
			Example:     "启用",
		},
		{
			Name:        "备注",
			Key:         "remark",
			Required:    false,
			Description: "设备备注信息",
			Example:     "测试设备",
		},
	}
}

// ValidProtocols 有效的协议类型
var ValidProtocols = map[string]bool{
	"JT808":  true,
	"GT06":   true,
	"Wialon": true,
}

// ParseStatus 解析状态字符串
func ParseStatus(status string) int {
	switch status {
	case "启用", "active", "1", "true":
		return 1
	default:
		return 0
	}
}
