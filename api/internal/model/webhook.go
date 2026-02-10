package model

import (
	"encoding/json"
	"time"
)

// WebhookEventType Webhook 事件类型
type WebhookEventType string

const (
	WebhookEventAlarmCreated      WebhookEventType = "alarm.created"
	WebhookEventAlarmResolved     WebhookEventType = "alarm.resolved"
	WebhookEventGeofenceEnter     WebhookEventType = "geofence.enter"
	WebhookEventGeofenceExit      WebhookEventType = "geofence.exit"
	WebhookEventDeviceOnline      WebhookEventType = "device.online"
	WebhookEventDeviceOffline     WebhookEventType = "device.offline"
	WebhookEventDevicePosition    WebhookEventType = "device.position"
	WebhookEventCommandResult     WebhookEventType = "device.command_result"
	WebhookEventVehicleCreated    WebhookEventType = "vehicle.created"
	WebhookEventVehicleUpdated    WebhookEventType = "vehicle.updated"
	WebhookEventAll               WebhookEventType = "all"
)

// WebhookStatus Webhook 状态
type WebhookStatus string

const (
	WebhookStatusActive   WebhookStatus = "active"
	WebhookStatusInactive WebhookStatus = "inactive"
	WebhookStatusFailed   WebhookStatus = "failed"
)

// Webhook Webhook 配置
type Webhook struct {
	ID              int                `json:"id" gorm:"primaryKey"`
	Name            string             `json:"name" gorm:"type:varchar(100);not null"`
	Description     string             `json:"description,omitempty" gorm:"type:text"`
	URL             string             `json:"url" gorm:"column:url;type:varchar(500);not null"`
	Secret          string             `json:"-" gorm:"type:varchar(255)"` // 不在 JSON 中暴露
	Events          []string           `json:"events" gorm:"type:webhook_event_type[];not null;default:'{}'"`
	Status          WebhookStatus      `json:"status" gorm:"type:webhook_status;not null;default:'active'"`
	RetryCount      int                `json:"retry_count" gorm:"column:retry_count;not null;default:3"`
	RetryInterval   int                `json:"retry_interval" gorm:"column:retry_interval;not null;default:5"`
	Timeout         int                `json:"timeout" gorm:"not null;default:30"`
	SuccessCount    int                `json:"success_count" gorm:"column:success_count;not null;default:0"`
	FailCount       int                `json:"fail_count" gorm:"column:fail_count;not null;default:0"`
	LastTriggeredAt *time.Time         `json:"last_triggered_at,omitempty" gorm:"column:last_triggered_at"`
	LastError       string             `json:"last_error,omitempty" gorm:"column:last_error;type:text"`
	CreatedBy       *int               `json:"created_by,omitempty" gorm:"column:created_by"`
	CreatedAt       time.Time          `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt       time.Time          `json:"updated_at" gorm:"not null;default:now()"`
	DeletedAt       *time.Time         `json:"-" gorm:"column:deleted_at"`
}

func (Webhook) TableName() string {
	return "webhooks"
}

// WebhookDelivery Webhook 投递日志
type WebhookDelivery struct {
	ID             int             `json:"id" gorm:"primaryKey"`
	WebhookID      int             `json:"webhook_id" gorm:"column:webhook_id;not null"`
	EventType      string          `json:"event_type" gorm:"column:event_type;type:webhook_event_type;not null"`
	Payload        json.RawMessage `json:"payload" gorm:"type:jsonb;not null"`
	ResponseStatus *int            `json:"response_status,omitempty" gorm:"column:response_status"`
	ResponseBody   string          `json:"response_body,omitempty" gorm:"column:response_body;type:text"`
	AttemptCount   int             `json:"attempt_count" gorm:"column:attempt_count;not null;default:1"`
	DurationMs     *int            `json:"duration_ms,omitempty" gorm:"column:duration_ms"`
	ErrorMessage   string          `json:"error_message,omitempty" gorm:"column:error_message;type:text"`
	CreatedAt      time.Time       `json:"created_at" gorm:"not null;default:now()"`
	CompletedAt    *time.Time      `json:"completed_at,omitempty" gorm:"column:completed_at"`
}

func (WebhookDelivery) TableName() string {
	return "webhook_deliveries"
}

// WebhookEvent Webhook 事件数据结构
type WebhookEvent struct {
	ID        string          `json:"id"`         // 事件唯一ID
	Type      string          `json:"type"`       // 事件类型
	Timestamp int64           `json:"timestamp"`  // 时间戳（毫秒）
	Data      json.RawMessage `json:"data"`       // 事件数据
}

// WebhookPayload Webhook 请求体
type WebhookPayload struct {
	EventID   string      `json:"event_id"`
	EventType string      `json:"event_type"`
	Timestamp int64       `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// WebhookSignature Webhook 签名头
type WebhookSignature struct {
	Timestamp string `json:"timestamp"`
	Signature string `json:"signature"`
}

// CreateWebhookRequest 创建 Webhook 请求
type CreateWebhookRequest struct {
	Name          string   `json:"name" binding:"required,max=100"`
	Description   string   `json:"description"`
	URL           string   `json:"url" binding:"required,url,max=500"`
	Secret        string   `json:"secret" binding:"max=255"`
	Events        []string `json:"events" binding:"required"`
	RetryCount    int      `json:"retry_count" binding:"min=0,max=10"`
	RetryInterval int      `json:"retry_interval" binding:"min=1,max=300"`
	Timeout       int      `json:"timeout" binding:"min=1,max=300"`
}

// UpdateWebhookRequest 更新 Webhook 请求
type UpdateWebhookRequest struct {
	Name          string   `json:"name" binding:"omitempty,max=100"`
	Description   string   `json:"description"`
	URL           string   `json:"url" binding:"omitempty,url,max=500"`
	Secret        string   `json:"secret" binding:"max=255"`
	Events        []string `json:"events"`
	RetryCount    int      `json:"retry_count" binding:"min=0,max=10"`
	RetryInterval int      `json:"retry_interval" binding:"min=1,max=300"`
	Timeout       int      `json:"timeout" binding:"min=1,max=300"`
}

// WebhookListQuery Webhook 列表查询参数
type WebhookListQuery struct {
	Status   WebhookStatus `form:"status"`
	Event    string        `form:"event"`
	Page     int           `form:"page,default=1"`
	PageSize int           `form:"page_size,default=20"`
}

// WebhookListResponse Webhook 列表响应
type WebhookListResponse struct {
	List     []Webhook `json:"list"`
	Total    int64     `json:"total"`
	Page     int       `json:"page"`
	PageSize int       `json:"page_size"`
}

// WebhookDeliveryQuery 投递日志查询参数
type WebhookDeliveryQuery struct {
	WebhookID int    `form:"webhook_id"`
	EventType string `form:"event_type"`
	Status    string `form:"status"` // success, failed
	Page      int    `form:"page,default=1"`
	PageSize  int    `form:"page_size,default=20"`
}

// WebhookDeliveryResponse 投递日志响应
type WebhookDeliveryResponse struct {
	List     []WebhookDelivery `json:"list"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

// TestWebhookRequest 测试 Webhook 请求
type TestWebhookRequest struct {
	EventType string          `json:"event_type" binding:"required"`
	Payload   json.RawMessage `json:"payload" binding:"required"`
}

// TestWebhookResponse 测试 Webhook 响应
type TestWebhookResponse struct {
	Success       bool   `json:"success"`
	StatusCode    int    `json:"status_code,omitempty"`
	ResponseBody  string `json:"response_body,omitempty"`
	DurationMs    int    `json:"duration_ms"`
	ErrorMessage  string `json:"error_message,omitempty"`
}

// WebhookStats Webhook 统计
type WebhookStats struct {
	TotalWebhooks   int64 `json:"total_webhooks"`
	ActiveWebhooks  int64 `json:"active_webhooks"`
	InactiveWebhooks int64 `json:"inactive_webhooks"`
	FailedWebhooks  int64 `json:"failed_webhooks"`
	TotalDeliveries int64 `json:"total_deliveries"`
	SuccessDeliveries int64 `json:"success_deliveries"`
	FailedDeliveries  int64 `json:"failed_deliveries"`
	TodayDeliveries   int64 `json:"today_deliveries"`
}

// WebhookEventData 各类事件数据结构

// AlarmEventData 报警事件数据
type AlarmEventData struct {
	AlarmID      int             `json:"alarm_id"`
	Type         AlarmType       `json:"type"`
	Level        AlarmLevel      `json:"level"`
	DeviceID     string          `json:"device_id"`
	DeviceName   string          `json:"device_name"`
	Title        string          `json:"title"`
	Content      string          `json:"content"`
	Lat          *float64        `json:"lat,omitempty"`
	Lon          *float64        `json:"lon,omitempty"`
	Speed        *int16          `json:"speed,omitempty"`
	GeofenceID   *int            `json:"geofence_id,omitempty"`
	GeofenceName string          `json:"geofence_name,omitempty"`
	Extras       json.RawMessage `json:"extras,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

// GeofenceEventData 围栏事件数据
type GeofenceEventData struct {
	GeofenceID   uint      `json:"geofence_id"`
	GeofenceName string    `json:"geofence_name"`
	DeviceID     string    `json:"device_id"`
	DeviceName   string    `json:"device_name"`
	EventType    string    `json:"event_type"` // enter, exit
	Lat          float64   `json:"lat"`
	Lon          float64   `json:"lon"`
	Speed        float64   `json:"speed"`
	Timestamp    int64     `json:"timestamp"`
}

// DeviceEventData 设备事件数据
type DeviceEventData struct {
	DeviceID   string    `json:"device_id"`
	DeviceName string    `json:"device_name"`
	Status     string    `json:"status"` // online, offline
	Timestamp  int64     `json:"timestamp"`
	Reason     string    `json:"reason,omitempty"`
}

// PositionEventData 位置事件数据
type PositionEventData struct {
	DeviceID  string  `json:"device_id"`
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	Speed     float64 `json:"speed"`
	Angle     float64 `json:"angle"`
	Timestamp int64   `json:"timestamp"`
}

// CommandResultEventData 指令结果事件数据
type CommandResultEventData struct {
	DeviceID   string          `json:"device_id"`
	CommandID  string          `json:"command_id"`
	Command    string          `json:"command"`
	Status     string          `json:"status"` // success, failed, timeout
	Result     json.RawMessage `json:"result,omitempty"`
	Error      string          `json:"error,omitempty"`
	Timestamp  int64           `json:"timestamp"`
}

// VehicleEventData 车辆事件数据
type VehicleEventData struct {
	VehicleID    uint   `json:"vehicle_id"`
	PlateNumber  string `json:"plate_number"`
	Type         string `json:"type"`
	Status       string `json:"status"`
	Organization string `json:"organization,omitempty"`
}
