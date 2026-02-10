// 油耗管理服务

package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"gorm.io/gorm"

	"openfms/api/internal/model"
)

// FuelService 油耗服务
type FuelService struct {
	db *gorm.DB
}

// NewFuelService 创建油耗服务
func NewFuelService(db *gorm.DB) *FuelService {
	return &FuelService{db: db}
}

// CreateFuelRecord 创建加油记录
func (s *FuelService) CreateFuelRecord(ctx context.Context, req *model.CreateFuelRecordRequest, userID int) (*model.FuelRecord, error) {
	// 获取上次加油记录
	var lastRecord model.FuelRecord
	s.db.Where("vehicle_id = ?", req.VehicleID).
		Order("fuel_time DESC").
		First(&lastRecord)

	// 计算行驶里程
	var tripDistance float64
	if lastRecord.ID > 0 {
		tripDistance = req.CurrentMileage - lastRecord.CurrentMileage
	}

	// 计算油耗
	var consumption float64
	if tripDistance > 0 {
		consumption = (req.FuelVolume / tripDistance) * 100
	}

	// 计算总成本
	var totalCost float64
	if req.FuelPrice > 0 {
		totalCost = req.FuelVolume * req.FuelPrice
	}

	fuelTime, _ := time.Parse(time.RFC3339, req.FuelTime)

	record := &model.FuelRecord{
		VehicleID:       req.VehicleID,
		FuelTime:        fuelTime,
		FuelType:        req.FuelType,
		FuelVolume:      req.FuelVolume,
		FuelPrice:       &req.FuelPrice,
		TotalCost:       &totalCost,
		CurrentMileage:  req.CurrentMileage,
		LastMileage:     &lastRecord.CurrentMileage,
		TripDistance:    &tripDistance,
		FuelConsumption: &consumption,
		StationName:     req.StationName,
		StationLocation: req.StationLocation,
		Lat:             &req.Lat,
		Lon:             &req.Lon,
		Remark:          req.Remark,
		CreatedBy:       &userID,
	}

	if err := s.db.Create(record).Error; err != nil {
		return nil, err
	}

	// 检查油耗异常
	if consumption > 0 {
		s.checkFuelAnomaly(record, consumption)
	}

	// 更新日统计
	s.updateDailyStats(req.VehicleID, fuelTime)

	return record, nil
}

// GetFuelRecords 获取加油记录
func (s *FuelService) GetFuelRecords(ctx context.Context, vehicleID int, startTime, endTime time.Time, page, pageSize int) ([]model.FuelRecord, int64, error) {
	var records []model.FuelRecord
	var total int64

	query := s.db.Model(&model.FuelRecord{}).Where("vehicle_id = ?", vehicleID)
	
	if !startTime.IsZero() {
		query = query.Where("fuel_time >= ?", startTime)
	}
	if !endTime.IsZero() {
		query = query.Where("fuel_time <= ?", endTime)
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Order("fuel_time DESC").Offset(offset).Limit(pageSize).Find(&records).Error

	return records, total, err
}

// GetFuelStats 获取油耗统计
func (s *FuelService) GetFuelStats(ctx context.Context, vehicleID int, startTime, endTime time.Time) (*model.FuelStatsResponse, error) {
	var stats model.FuelStatsResponse

	// 总加油量和总成本
	var totalVolume, totalCost float64
	s.db.Model(&model.FuelRecord{}).
		Where("vehicle_id = ? AND fuel_time >= ? AND fuel_time <= ?", vehicleID, startTime, endTime).
		Select("COALESCE(SUM(fuel_volume), 0), COALESCE(SUM(total_cost), 0)").
		Row().Scan(&totalVolume, &totalCost)

	// 加油次数
	var refuelCount int64
	s.db.Model(&model.FuelRecord{}).
		Where("vehicle_id = ? AND fuel_time >= ? AND fuel_time <= ?", vehicleID, startTime, endTime).
		Count(&refuelCount)

	// 行驶里程
	var totalDistance float64
	s.db.Model(&model.FuelRecord{}).
		Where("vehicle_id = ? AND fuel_time >= ? AND fuel_time <= ? AND trip_distance IS NOT NULL", 
			vehicleID, startTime, endTime).
		Select("COALESCE(SUM(trip_distance), 0)").
		Scan(&totalDistance)

	// 计算平均值
	if totalDistance > 0 {
		stats.AvgConsumption = (totalVolume / totalDistance) * 100
		stats.AvgCostPerKm = totalCost / totalDistance
	}
	if totalVolume > 0 {
		stats.AvgPrice = totalCost / totalVolume
	}

	stats.TotalVolume = totalVolume
	stats.TotalCost = totalCost
	stats.TotalDistance = totalDistance
	stats.RefuelCount = int(refuelCount)

	return &stats, nil
}

// GetFuelTrend 获取油耗趋势
func (s *FuelService) GetFuelTrend(ctx context.Context, vehicleID int, days int) ([]model.FuelTrendItem, error) {
	var items []model.FuelTrendItem

	// 查询日统计
	var stats []model.FuelDailyStats
	err := s.db.Where("vehicle_id = ?", vehicleID).
		Order("date DESC").
		Limit(days).
		Find(&stats).Error
	if err != nil {
		return nil, err
	}

	for _, stat := range stats {
		items = append(items, model.FuelTrendItem{
			Date:        stat.Date,
			Mileage:     stat.DailyMileage,
			FuelVolume:  stat.FuelVolume,
			FuelCost:    stat.FuelCost,
			Consumption: stat.AvgConsumption,
			CostPerKm:   stat.CostPerKm,
		})
	}

	return items, nil
}

// checkFuelAnomaly 检查油耗异常
func (s *FuelService) checkFuelAnomaly(record *model.FuelRecord, consumption float64) {
	// 获取历史平均油耗
	var avgConsumption float64
	s.db.Model(&model.FuelRecord{}).
		Where("vehicle_id = ? AND fuel_consumption IS NOT NULL", record.VehicleID).
		Select("AVG(fuel_consumption)").
		Scan(&avgConsumption)

	if avgConsumption == 0 {
		return
	}

	// 判断异常（油耗突然增加50%以上）
	if consumption > avgConsumption*1.5 {
		anomaly := &model.FuelAnomaly{
			VehicleID:           record.VehicleID,
			AnomalyType:         "abnormal_high",
			AnomalyTime:         record.FuelTime,
			ExpectedConsumption: avgConsumption,
			ActualConsumption:   consumption,
			Lat:                 record.Lat,
			Lon:                 record.Lon,
		}
		s.db.Create(anomaly)
	}

	// 判断异常（油耗突然降低，可能偷油）
	if consumption < avgConsumption*0.5 && consumption > 0 {
		anomaly := &model.FuelAnomaly{
			VehicleID:           record.VehicleID,
			AnomalyType:         "sudden_drop",
			AnomalyTime:         record.FuelTime,
			ExpectedConsumption: avgConsumption,
			ActualConsumption:   consumption,
			Lat:                 record.Lat,
			Lon:                 record.Lon,
		}
		s.db.Create(anomaly)
	}
}

// updateDailyStats 更新日统计
func (s *FuelService) updateDailyStats(vehicleID int, date time.Time) {
	dateStr := date.Format("2006-01-02")

	// 计算当日统计
	var stats model.FuelDailyStats
	s.db.Where("vehicle_id = ? AND date = ?", vehicleID, dateStr).First(&stats)

	// 当日加油总量
	var totalVolume, totalCost float64
	s.db.Model(&model.FuelRecord{}).
		Where("vehicle_id = ? AND DATE(fuel_time) = ?", vehicleID, dateStr).
		Select("COALESCE(SUM(fuel_volume), 0), COALESCE(SUM(total_cost), 0)").
		Row().Scan(&totalVolume, &totalCost)

	stats.VehicleID = vehicleID
	stats.Date = dateStr
	stats.FuelVolume = &totalVolume
	stats.FuelCost = &totalCost

	// 保存统计
	if stats.ID > 0 {
		s.db.Save(&stats)
	} else {
		s.db.Create(&stats)
	}
}

// DeleteFuelRecord 删除加油记录
func (s *FuelService) DeleteFuelRecord(ctx context.Context, id int) error {
	return s.db.Delete(&model.FuelRecord{}, id).Error
}
