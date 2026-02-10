package model

import (
	"time"
)

// LoginLog 登录日志
type LoginLog struct {
	ID        int       `json:"id" gorm:"primaryKey"`
	UserID    int       `json:"user_id" gorm:"column:user_id"`
	Username  string    `json:"username" gorm:"type:varchar(50)"`
	Action    string    `json:"action" gorm:"type:varchar(20);not null"` // login, logout
	IP        string    `json:"ip" gorm:"type:varchar(50)"`
	UserAgent string    `json:"user_agent" gorm:"column:user_agent;type:varchar(500)"`
	Success   bool      `json:"success" gorm:"not null;default:true"`
	ErrorMsg  string    `json:"error_msg,omitempty" gorm:"column:error_msg;type:text"`
	CreatedAt time.Time `json:"created_at" gorm:"not null;default:now()"`
}

func (LoginLog) TableName() string {
	return "login_logs"
}

// OperationLog 操作日志
type OperationLog struct {
	ID          int       `json:"id" gorm:"primaryKey"`
	UserID      int       `json:"user_id" gorm:"column:user_id"`
	Username    string    `json:"username" gorm:"type:varchar(50)"`
	Module      string    `json:"module" gorm:"type:varchar(50)"` // device, vehicle, user, etc.
	Action      string    `json:"action" gorm:"type:varchar(50)"` // create, update, delete
	ResourceID  string    `json:"resource_id" gorm:"column:resource_id"`
	OldValue    string    `json:"old_value,omitempty" gorm:"column:old_value;type:jsonb"`
	NewValue    string    `json:"new_value,omitempty" gorm:"column:new_value;type:jsonb"`
	IP          string    `json:"ip" gorm:"type:varchar(50)"`
	CreatedAt   time.Time `json:"created_at" gorm:"not null;default:now()"`
}

func (OperationLog) TableName() string {
	return "operation_logs"
}
