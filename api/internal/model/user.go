package model

import (
	"time"

	"gorm.io/gorm"
)

// User represents a system user
type User struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Username  string         `json:"username" gorm:"uniqueIndex;size:50"`
	Password  string         `json:"-" gorm:"size:255"` // hashed password
	Email     string         `json:"email" gorm:"uniqueIndex;size:100"`
	Phone     string         `json:"phone" gorm:"size:20"`
	Role      string         `json:"role" gorm:"size:20;default:'user'"` // admin, manager, user
	Status    int            `json:"status" gorm:"default:1"`            // 0: inactive, 1: active
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// LoginRequest represents login credentials
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents login response
type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// Geofence represents a geographic fence
type Geofence struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"size:100"`
	Type        string         `json:"type" gorm:"size:20"` // circle, polygon
	Coordinates JSONMap        `json:"coordinates" gorm:"type:jsonb"` // {center: {lat, lon}, radius} or {points: [{lat, lon}, ...]}
	AlertType   string         `json:"alert_type" gorm:"size:20"`     // enter, exit, both
	Status      int            `json:"status" gorm:"default:1"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}
