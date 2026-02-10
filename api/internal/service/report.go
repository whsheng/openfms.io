package service

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"openfms/api/internal/model"
)

// ReportService 报表服务
type ReportService struct {
	db *gorm.DB
}

// NewReportService 创建报表服务
func NewReportService(db *gorm.DB) *ReportService {
	return &ReportService{db: db}
}

// GetMileageReport 获取里程报表
func (s *ReportService) GetMileageReport(ctx context.Context, deviceIDs []string, startDate, endDate string) ([]model.DeviceDailyStats, error) {
	var stats []model.DeviceDailyStats
	
	query := s.db.Where("date >= ? AND date <= ?", startDate, endDate)
	if len(deviceIDs) > 0 {
		query = query.Where("device_id IN ?", deviceIDs)
	}
	
	err := query.Order("date DESC, device_id").Find(&stats).Error
	return stats, err
}

// GetMileageSummary 获取里程汇总
func (s *ReportService) GetMileageSummary(ctx context.Context, deviceIDs []string, startDate, endDate string) (map[string]interface{}, error) {
	type Result struct {
		TotalMileage float64 `gorm:"column:total_mileage"`
		AvgMileage   float64 `gorm:"column:avg_mileage"`
		MaxMileage   float64 `gorm:"column:max_mileage"`
		TotalDays    int     `gorm:"column:total_days"`
	}
	
	var result Result
	query := s.db.Model(&model.DeviceDailyStats{}).
		Select("SUM(daily_mileage) as total_mileage, " +
			"AVG(daily_mileage) as avg_mileage, " +
			"MAX(daily_mileage) as max_mileage, " +
			"COUNT(DISTINCT date) as total_days").
		Where("date >= ? AND date <= ?", startDate, endDate)
	
	if len(deviceIDs) > 0 {
		query = query.Where("device_id IN ?", deviceIDs)
	}
	
	if err := query.Scan(&result).Error; err != nil {
		return nil, err
	}
	
	return map[string]interface{}{
		"total_mileage": result.TotalMileage,
		"avg_mileage":   result.AvgMileage,
		"max_mileage":   result.MaxMileage,
		"total_days":    result.TotalDays,
	}, nil
}

// GetStopReport 获取停车报表
func (s *ReportService) GetStopReport(ctx context.Context, deviceID string, startTime, endTime time.Time, minDuration int) ([]model.StopPoint, error) {
	var stops []model.StopPoint
	
	query := s.db.Where("device_id = ? AND start_time >= ? AND end_time <= ?",
		deviceID, startTime, endTime)
	
	if minDuration > 0 {
		query = query.Where("duration >= ?", minDuration)
	}
	
	err := query.Order("start_time DESC").Find(&stops).Error
	return stops, err
}

// GetDrivingBehaviorReport 获取驾驶行为报表
func (s *ReportService) GetDrivingBehaviorReport(ctx context.Context, deviceIDs []string, startDate, endDate string) (map[string]interface{}, error) {
	type EventCount struct {
		EventType string `gorm:"column:event_type"`
		Count     int    `gorm:"column:count"`
	}
	
	var counts []EventCount
	query := s.db.Model(&model.DrivingEvent{}).
		Select("event_type, COUNT(*) as count").
		Where("event_time >= ? AND event_time <= ?", startDate, endDate).
		Group("event_type")
	
	if len(deviceIDs) > 0 {
		query = query.Where("device_id IN ?", deviceIDs)
	}
	
	if err := query.Scan(&counts).Error; err != nil {
		return nil, err
	}
	
	result := map[string]interface{}{
		"harsh_acceleration": 0,
		"harsh_braking":      0,
		"harsh_turning":      0,
		"speeding":           0,
	}
	
	for _, c := range counts {
		result[c.EventType] = c.Count
	}
	
	return result, nil
}

// GetDailyStats 获取日统计数据
func (s *ReportService) GetDailyStats(ctx context.Context, startDate, endDate string) ([]model.DailyStats, error) {
	var stats []model.DailyStats
	err := s.db.Where("date >= ? AND date <= ?", startDate, endDate).
		Order("date DESC").
		Find(&stats).Error
	return stats, err
}

// GetDashboardStats 获取仪表盘统计数据
func (s *ReportService) GetDashboardStats(ctx context.Context) (map[string]interface{}, error) {
	today := time.Now().Format("2006-01-02")
	
	// 今日统计
	var todayStats model.DailyStats
	s.db.Where("date = ?", today).First(&todayStats)
	
	// 设备统计
	var totalDevices, onlineDevices int64
	s.db.Model(&model.Device{}).Count(&totalDevices)
	s.db.Model(&model.Device{}).Where("status = ?", model.DeviceStatusOnline).Count(&onlineDevices)
	
	// 报警统计（今日）
	var todayAlarms int64
	s.db.Model(&model.Alarm{}).Where("DATE(created_at) = ?", today).Count(&todayAlarms)
	
	// 未处理报警
	var unreadAlarms int64
	s.db.Model(&model.Alarm{}).Where("status = ?", model.AlarmStatusUnread).Count(&unreadAlarms)
	
	return map[string]interface{}{
		"today": map[string]interface{}{
			"mileage":       todayStats.TotalMileage,
			"alarms":        todayAlarms,
			"online_devices": onlineDevices,
		},
		"total_devices":  totalDevices,
		"online_devices": onlineDevices,
		"unread_alarms":  unreadAlarms,
	}, nil
}

// CalculateDailyStats 计算日统计（定时任务调用）
func (s *ReportService) CalculateDailyStats(date string) error {
	// 检查是否已存在
	var existing model.DailyStats
	if err := s.db.Where("date = ?", date).First(&existing).Error; err == nil {
		// 已存在，删除重新计算
		s.db.Delete(&existing)
	}
	
	stats := model.DailyStats{
		Date: date,
	}
	
	// 统计设备
	var totalDevices, onlineDevices int64
	s.db.Model(&model.Device{}).Count(&totalDevices)
	s.db.Model(&model.Device{}).Where("status = ?", model.DeviceStatusOnline).Count(&onlineDevices)
	stats.TotalDevices = int(totalDevices)
	stats.OnlineDevices = int(onlineDevices)
	stats.OfflineDevices = int(totalDevices - onlineDevices)
	
	// 统计报警
	var totalAlarms, criticalAlarms, warningAlarms, infoAlarms, resolvedAlarms int64
	s.db.Model(&model.Alarm{}).Where("DATE(created_at) = ?", date).Count(&totalAlarms)
	s.db.Model(&model.Alarm{}).Where("DATE(created_at) = ? AND level = ?", date, model.AlarmLevelCritical).Count(&criticalAlarms)
	s.db.Model(&model.Alarm{}).Where("DATE(created_at) = ? AND level = ?", date, model.AlarmLevelWarning).Count(&warningAlarms)
	s.db.Model(&model.Alarm{}).Where("DATE(created_at) = ? AND level = ?", date, model.AlarmLevelInfo).Count(&infoAlarms)
	s.db.Model(&model.Alarm{}).Where("DATE(created_at) = ? AND status = ?", date, model.AlarmStatusResolved).Count(&resolvedAlarms)
	
	stats.TotalAlarms = int(totalAlarms)
	stats.CriticalAlarms = int(criticalAlarms)
	stats.WarningAlarms = int(warningAlarms)
	stats.InfoAlarms = int(infoAlarms)
	stats.ResolvedAlarms = int(resolvedAlarms)
	
	// 统计里程
	var totalMileage float64
	s.db.Model(&model.DeviceDailyStats{}).Where("date = ?", date).Select("SUM(daily_mileage)").Scan(&totalMileage)
	stats.TotalMileage = totalMileage
	
	if totalDevices > 0 {
		stats.AvgMileagePerDevice = totalMileage / float64(totalDevices)
	}
	
	return s.db.Create(&stats).Error
}

// GenerateReport 生成报表文件
func (s *ReportService) GenerateReport(ctx context.Context, reportType string, deviceIDs []string, startDate, endDate string, userID int) (string, error) {
	// 创建报表任务
	job := model.ReportJob{
		Name:        fmt.Sprintf("%s报表_%s", reportType, time.Now().Format("20060102")),
		ReportType:  reportType,
		DeviceIDs:   deviceIDs,
		StartDate:   startDate,
		EndDate:     endDate,
		Status:      "running",
		CreatedBy:   userID,
	}
	s.db.Create(&job)
	
	// 异步生成报表
	go s.generateReportFile(&job)
	
	return fmt.Sprintf("/api/v1/reports/%d/download", job.ID), nil
}

// generateReportFile 生成报表文件（内部方法）
func (s *ReportService) generateReportFile(job *model.ReportJob) {
	// 这里简化处理，实际应该生成 Excel/PDF 文件
	// 可以使用 github.com/xuri/excelize 生成 Excel
	
	// 模拟生成过程
	time.Sleep(2 * time.Second)
	
	job.Status = "completed"
	job.Progress = 100
	now := time.Now()
	job.CompletedAt = &now
	job.FileURL = fmt.Sprintf("/reports/%s_%d.xlsx", job.ReportType, job.ID)
	s.db.Save(job)
}
