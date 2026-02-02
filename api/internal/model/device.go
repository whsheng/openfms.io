package model

import (
	"time"

	"gorm.io/gorm"
)

// Device represents a GPS tracking device
type Device struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	DeviceID    string         `json:"device_id" gorm:"uniqueIndex;size:32"` // SIM number / Device ID
	Name        string         `json:"name" gorm:"size:100"`
	Protocol    string         `json:"protocol" gorm:"size:20"` // JT808, GT06, etc.
	VehicleID   *uint          `json:"vehicle_id"`
	Vehicle     *Vehicle       `json:"vehicle,omitempty"`
	Status      int            `json:"status" gorm:"default:0"` // 0: inactive, 1: active
	LastOnline  *time.Time     `json:"last_online"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// DeviceShadow represents the real-time state of a device (stored in Redis)
type DeviceShadow struct {
	DeviceID  string  `json:"device_id"`
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	Speed     float64 `json:"spd"`
	Direction float64 `json:"dir"`
	Status    int     `json:"st"`
	Fuel      float64 `json:"fuel,omitempty"`
	Timestamp int64   `json:"ts"`
}

// Vehicle represents a vehicle in the fleet
type Vehicle struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	PlateNumber  string         `json:"plate_number" gorm:"uniqueIndex;size:20"`
	Type         string         `json:"type" gorm:"size:50"` // truck, van, car, etc.
	Brand        string         `json:"brand" gorm:"size:50"`
	Model        string         `json:"model" gorm:"size:50"`
	Color        string         `json:"color" gorm:"size:20"`
	Year         int            `json:"year"`
	Status       int            `json:"status" gorm:"default:1"` // 0: inactive, 1: active
	Organization string         `json:"organization" gorm:"size:100"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

// Position represents a GPS position record (TimescaleDB)
type Position struct {
	Time      time.Time `json:"time" gorm:"primaryKey"`
	DeviceID  string    `json:"device_id" gorm:"primaryKey;size:32"`
	Lat       float64   `json:"lat"`
	Lon       float64   `json:"lon"`
	Speed     int16     `json:"speed"`
	Angle     int16     `json:"angle"`
	Flags     int32     `json:"flags"`
	Extras    JSONMap   `json:"extras" gorm:"type:jsonb"`
}

// JSONMap is a helper type for JSONB fields
type JSONMap map[string]interface{}
