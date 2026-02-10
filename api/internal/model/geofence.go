package model

import (
	"time"

	"gorm.io/gorm"
)

// Geofence represents an electronic fence for geo-fencing functionality
type Geofence struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"size:100;not null"`
	Description string         `json:"description"`
	Type        string         `json:"type" gorm:"size:20;not null"` // circle, polygon
	Coordinates JSONMap        `json:"coordinates" gorm:"type:jsonb;not null"`
	AlertType   string         `json:"alert_type" gorm:"size:20;default:both"` // enter, exit, both
	Status      int            `json:"status" gorm:"default:1"`                // 0: inactive, 1: active
	UserID      *uint          `json:"user_id"`
	User        *User          `json:"user,omitempty"`
	Devices     []Device       `json:"devices,omitempty" gorm:"many2many:geofence_devices;"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// GeofenceDevice represents the association between geofence and device
type GeofenceDevice struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	GeofenceID  uint      `json:"geofence_id" gorm:"not null"`
	DeviceID    uint      `json:"device_id" gorm:"not null"`
	Device      Device    `json:"device,omitempty"`
	Geofence    Geofence  `json:"geofence,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// GeofenceEvent represents a geofence entry/exit event
type GeofenceEvent struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	GeofenceID   uint      `json:"geofence_id" gorm:"not null"`
	DeviceID     uint      `json:"device_id" gorm:"not null"`
	Geofence     Geofence  `json:"geofence,omitempty"`
	Device       Device    `json:"device,omitempty"`
	EventType    string    `json:"event_type" gorm:"size:20;not null"` // enter, exit
	Location     JSONMap   `json:"location" gorm:"type:jsonb;not null"`
	Speed        float64   `json:"speed"`
	TriggeredAt  time.Time `json:"triggered_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// DeviceGeofenceState represents the current state of a device relative to a geofence
type DeviceGeofenceState struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	DeviceID        uint           `json:"device_id" gorm:"not null"`
	GeofenceID      uint           `json:"geofence_id" gorm:"not null"`
	IsInside        bool           `json:"is_inside" gorm:"default:false"`
	LastEventType   string         `json:"last_event_type" gorm:"size:20"`
	LastTriggeredAt *time.Time     `json:"last_triggered_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	CreatedAt       time.Time      `json:"created_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

// GeofenceCoordinates represents the coordinates for different geofence types

// CircleGeofenceCoordinates for circle type geofence
// {
//   "center": {"lat": 39.9042, "lon": 116.4074},
//   "radius": 1000
// }
type CircleGeofenceCoordinates struct {
	Center struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	} `json:"center"`
	Radius float64 `json:"radius"` // in meters
}

// PolygonGeofenceCoordinates for polygon type geofence
// {
//   "points": [
//     {"lat": 39.9042, "lon": 116.4074},
//     {"lat": 39.9142, "lon": 116.4174},
//     ...
//   ]
// }
type PolygonGeofenceCoordinates struct {
	Points []struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	} `json:"points"`
}

// Location represents a GPS location point
type Location struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// GeofenceAlert represents a geofence alert for NATS messaging
type GeofenceAlert struct {
	GeofenceID  uint      `json:"geofence_id"`
	GeofenceName string   `json:"geofence_name"`
	DeviceID    uint      `json:"device_id"`
	DeviceName  string    `json:"device_name"`
	EventType   string    `json:"event_type"` // enter, exit
	Location    Location  `json:"location"`
	Speed       float64   `json:"speed"`
	Timestamp   int64     `json:"timestamp"`
}
