package model

import (
	"time"
)

// FuelRecord 加油记录
type FuelRecord struct {
	ID              int             `json:"id" gorm:"primaryKey"`
	VehicleID       int             `json:"vehicle_id" gorm:"column:vehicle_id;not null"`
	DeviceID        *string         `json:"device_id,omitempty" gorm:"column:device_id;type:varchar(20)"`
	FuelTime        time.Time       `json:"fuel_time" gorm:"column:fuel_time;not null"`
	FuelType        string          `json:"fuel_type" gorm:"column:fuel_type;type:varchar(20);default:'汽油'"`
	FuelVolume      float64         `json:"fuel_volume" gorm:"column:fuel_volume;not null"`
	FuelPrice       *float64        `json:"fuel_price,omitempty" gorm:"column:fuel_price"`
	TotalCost       *float64        `json:"total_cost,omitempty" gorm:"column:total_cost"`
	CurrentMileage  float64         `json:"current_mileage" gorm:"column:current_mileage;not null"`
	LastMileage     *float64        `json:"last_mileage,omitempty" gorm:"column:last_mileage"`
	TripDistance    *float64        `json:"trip_distance,omitempty" gorm:"column:trip_distance"`
	FuelConsumption *float64        `json:"fuel_consumption,omitempty" gorm:"column:fuel_consumption"`
	StationName     string          `json:"station_name,omitempty" gorm:"column:station_name;type:varchar(100)"`
	StationLocation string          `json:"station_location,omitempty" gorm:"column:station_location;type:varchar(200)"`
	Lat             *float64        `json:"lat,omitempty"`
	Lon             *float64        `json:"lon,omitempty"`
	Remark          string          `json:"remark,omitempty" gorm:"type:text"`
	CreatedBy       *int            `json:"created_by,omitempty" gorm:"column:created_by"`
	CreatedAt       time.Time       `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt       time.Time       `json:"updated_at" gorm:"not null;default:now()"`
	
	// 关联
	Vehicle *Vehicle `json:"vehicle,omitempty" gorm:"foreignKey:VehicleID"`
}

func (FuelRecord) TableName() string {
	return "fuel_records"
}

// FuelDailyStats 油耗日统计
type FuelDailyStats struct {
	ID             int       `json:"id" gorm:"primaryKey"`
	VehicleID      int       `json:"vehicle_id" gorm:"column:vehicle_id;not null"`
	Date           string    `json:"date" gorm:"type:date;not null"`
	StartMileage   *float64  `json:"start_mileage,omitempty" gorm:"column:start_mileage"`
	EndMileage     *float64  `json:"end_mileage,omitempty" gorm:"column:end_mileage"`
	DailyMileage   *float64  `json:"daily_mileage,omitempty" gorm:"column:daily_mileage"`
	FuelVolume     *float64  `json:"fuel_volume,omitempty" gorm:"column:fuel_volume"`
	FuelCost       *float64  `json:"fuel_cost,omitempty" gorm:"column:fuel_cost"`
	AvgConsumption *float64  `json:"avg_consumption,omitempty" gorm:"column:avg_consumption"`
	CostPerKm      *float64  `json:"cost_per_km,omitempty" gorm:"column:cost_per_km"`
	CreatedAt      time.Time `json:"created_at" gorm:"not null;default:now()"`
}

func (FuelDailyStats) TableName() string {
	return "fuel_daily_stats"
}

// FuelAnomaly 油耗异常
type FuelAnomaly struct {
	ID                int       `json:"id" gorm:"primaryKey"`
	VehicleID         int       `json:"vehicle_id" gorm:"column:vehicle_id;not null"`
	AnomalyType       string    `json:"anomaly_type" gorm:"column:anomaly_type;type:varchar(50);not null"`
	AnomalyTime       time.Time `json:"anomaly_time" gorm:"column:anomaly_time;not null"`
	ExpectedConsumption float64 `json:"expected_consumption" gorm:"column:expected_consumption"`
	ActualConsumption   float64 `json:"actual_consumption" gorm:"column:actual_consumption"`
	Lat               *float64  `json:"lat,omitempty"`
	Lon               *float64  `json:"lon,omitempty"`
	Status            string    `json:"status" gorm:"type:varchar(20);default:'unprocessed'"`
	Remark            string    `json:"remark,omitempty" gorm:"type:text"`
	CreatedAt         time.Time `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt         time.Time `json:"updated_at" gorm:"not null;default:now()"`
}

func (FuelAnomaly) TableName() string {
	return "fuel_anomalies"
}

// CreateFuelRecordRequest 创建加油记录请求
type CreateFuelRecordRequest struct {
	VehicleID       int     `json:"vehicle_id" binding:"required"`
	FuelTime        string  `json:"fuel_time" binding:"required"`
	FuelType        string  `json:"fuel_type"`
	FuelVolume      float64 `json:"fuel_volume" binding:"required,gt=0"`
	FuelPrice       float64 `json:"fuel_price"`
	CurrentMileage  float64 `json:"current_mileage" binding:"required,gt=0"`
	StationName     string  `json:"station_name"`
	StationLocation string  `json:"station_location"`
	Lat             float64 `json:"lat"`
	Lon             float64 `json:"lon"`
	Remark          string  `json:"remark"`
}

// FuelStatsResponse 油耗统计响应
type FuelStatsResponse struct {
	TotalVolume      float64 `json:"total_volume"`       // 总加油量
	TotalCost        float64 `json:"total_cost"`         // 总油费
	TotalDistance    float64 `json:"total_distance"`     // 总里程
	AvgConsumption   float64 `json:"avg_consumption"`    // 平均油耗
	AvgCostPerKm     float64 `json:"avg_cost_per_km"`    // 平均每公里成本
	RefuelCount      int     `json:"refuel_count"`       // 加油次数
	AvgPrice         float64 `json:"avg_price"`          // 平均油价
}

// FuelTrendItem 油耗趋势项
type FuelTrendItem struct {
	Date           string  `json:"date"`
	Mileage        float64 `json:"mileage"`
	FuelVolume     float64 `json:"fuel_volume"`
	FuelCost       float64 `json:"fuel_cost"`
	Consumption    float64 `json:"consumption"`
	CostPerKm      float64 `json:"cost_per_km"`
}
