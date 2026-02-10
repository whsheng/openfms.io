package model

import (
	"encoding/json"
	"time"
)

// AlarmType 报警类型
type AlarmType string

const (
	AlarmTypeGeofenceEnter AlarmType = "GEOFENCE_ENTER"
	AlarmTypeGeofenceExit  AlarmType = "GEOFENCE_EXIT"
	AlarmTypeOverspeed     AlarmType = "OVERSPEED"
	AlarmTypeLowBattery    AlarmType = "LOW_BATTERY"
	AlarmTypeOffline       AlarmType = "OFFLINE"
	AlarmTypeSOS           AlarmType = "SOS"
	AlarmTypePowerCut      AlarmType = "POWER_CUT"
	AlarmTypeVibration     AlarmType = "VIBRATION"
	AlarmTypeIllegalMove   AlarmType = "ILLEGAL_MOVE"
)

// AlarmLevel 报警级别
type AlarmLevel string

const (
	AlarmLevelInfo     AlarmLevel = "info"
	AlarmLevelWarning  AlarmLevel = "warning"
	AlarmLevelCritical AlarmLevel = "critical"
)

// AlarmStatus 报警状态
type AlarmStatus string

const (
	AlarmStatusUnread   AlarmStatus = "unread"
	AlarmStatusRead     AlarmStatus = "read"
	AlarmStatusResolved AlarmStatus = "resolved"
)

// Alarm 报警记录
type Alarm struct {
	ID           int             `json:"id" gorm:"primaryKey"`
	Type         AlarmType       `json:"type" gorm:"type:alarm_type;not null"`
	Level        AlarmLevel      `json:"level" gorm:"type:alarm_level;not null;default:'warning'"`
	DeviceID     string          `json:"device_id" gorm:"column:device_id;type:varchar(20);not null;index"`
	DeviceName   string          `json:"device_name" gorm:"column:device_name;type:varchar(100)"`
	Title        string          `json:"title" gorm:"type:varchar(200);not null"`
	Content      string          `json:"content" gorm:"type:text"`
	Lat          *float64        `json:"lat,omitempty" gorm:"type:double precision"`
	Lon          *float64        `json:"lon,omitempty" gorm:"type:double precision"`
	LocationName string          `json:"location_name,omitempty" gorm:"column:location_name;type:varchar(200)"`
	Speed        *int16          `json:"speed,omitempty"`
	SpeedLimit   *int16          `json:"speed_limit,omitempty" gorm:"column:speed_limit"`
	Status       AlarmStatus     `json:"status" gorm:"type:alarm_status;not null;default:'unread';index"`
	ResolvedAt   *time.Time      `json:"resolved_at,omitempty" gorm:"column:resolved_at"`
	ResolvedBy   *int            `json:"resolved_by,omitempty" gorm:"column:resolved_by"`
	ResolveNote  string          `json:"resolve_note,omitempty" gorm:"column:resolve_note;type:text"`
	GeofenceID   *int            `json:"geofence_id,omitempty" gorm:"column:geofence_id"`
	GeofenceName string          `json:"geofence_name,omitempty" gorm:"column:geofence_name;type:varchar(100)"`
	Extras       json.RawMessage `json:"extras,omitempty" gorm:"type:jsonb"`
	CreatedAt    time.Time       `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt    time.Time       `json:"updated_at" gorm:"not null;default:now()"`
}

func (Alarm) TableName() string {
	return "alarms"
}

// AlarmRule 报警规则
type AlarmRule struct {
	ID           int             `json:"id" gorm:"primaryKey"`
	Name         string          `json:"name" gorm:"type:varchar(100);not null"`
	Type         AlarmType       `json:"type" gorm:"type:alarm_type;not null;index"`
	Description  string          `json:"description,omitempty" gorm:"type:text"`
	Conditions   json.RawMessage `json:"conditions" gorm:"type:jsonb;not null;default:'{}'"`
	AllDevices   bool            `json:"all_devices" gorm:"column:all_devices;not null;default:true"`
	DeviceIDs    []int           `json:"device_ids,omitempty" gorm:"column:device_ids;type:integer[]"`
	NotifyWebhook bool           `json:"notify_webhook" gorm:"column:notify_webhook;not null;default:false"`
	WebhookURL   string          `json:"webhook_url,omitempty" gorm:"column:webhook_url;type:varchar(500)"`
	NotifyWS     bool            `json:"notify_ws" gorm:"column:notify_ws;not null;default:true"`
	NotifySound  bool            `json:"notify_sound" gorm:"column:notify_sound;not null;default:true"`
	Enabled      bool            `json:"enabled" gorm:"not null;default:true;index"`
	CreatedAt    time.Time       `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt    time.Time       `json:"updated_at" gorm:"not null;default:now()"`
}

func (AlarmRule) TableName() string {
	return "alarm_rules"
}

// AlarmSilence 报警静默规则
type AlarmSilence struct {
	ID         int       `json:"id" gorm:"primaryKey"`
	DeviceID   string    `json:"device_id" gorm:"column:device_id;type:varchar(20);not null;index"`
	AlarmType  AlarmType `json:"alarm_type" gorm:"type:alarm_type;not null"`
	SilenceUntil time.Time `json:"silence_until" gorm:"column:silence_until;not null"`
	Reason     string    `json:"reason,omitempty" gorm:"type:text"`
	CreatedBy  *int      `json:"created_by,omitempty" gorm:"column:created_by"`
	CreatedAt  time.Time `json:"created_at" gorm:"not null;default:now()"`
}

func (AlarmSilence) TableName() string {
	return "alarm_silences"
}

// AlarmStats 报警统计
type AlarmStats struct {
	Total       int64            `json:"total"`
	TotalToday  int64            `json:"total_today"`
	Unread      int64            `json:"unread"`
	Read        int64            `json:"read"`
	Resolved    int64            `json:"resolved"`
	Today       int64            `json:"today"`
	Critical    int64            `json:"critical"`
	Warning     int64            `json:"warning"`
	Info        int64            `json:"info"`
	ByType      map[string]int64 `json:"by_type,omitempty"`
	ByLevel     map[string]int64 `json:"by_level,omitempty"`
}

// AlarmTypeStats 按类型统计
type AlarmTypeStats struct {
	Type  AlarmType `json:"type"`
	Count int64     `json:"count"`
}

// CreateAlarmRequest 创建报警请求
type CreateAlarmRequest struct {
	Type         AlarmType       `json:"type" binding:"required"`
	Level        AlarmLevel      `json:"level" binding:"required"`
	DeviceID     string          `json:"device_id" binding:"required"`
	DeviceName   string          `json:"device_name"`
	Title        string          `json:"title" binding:"required"`
	Content      string          `json:"content"`
	Lat          *float64        `json:"lat"`
	Lon          *float64        `json:"lon"`
	LocationName string          `json:"location_name"`
	Speed        *int16          `json:"speed"`
	SpeedLimit   *int16          `json:"speed_limit"`
	GeofenceID   *int            `json:"geofence_id"`
	GeofenceName string          `json:"geofence_name"`
	Extras       json.RawMessage `json:"extras"`
}

// UpdateAlarmStatusRequest 更新报警状态请求
type UpdateAlarmStatusRequest struct {
	Status      AlarmStatus `json:"status" binding:"required"`
	ResolveNote string      `json:"resolve_note"`
}

// BatchResolveRequest 批量处理请求
type BatchResolveRequest struct {
	IDs         []int  `json:"ids" binding:"required"`
	ResolveNote string `json:"resolve_note"`
}

// AlarmListQuery 报警列表查询参数
type AlarmListQuery struct {
	DeviceID   string      `form:"device_id"`
	Type       AlarmType   `form:"type"`
	Level      AlarmLevel  `form:"level"`
	Status     AlarmStatus `form:"status"`
	StartTime  *time.Time  `form:"start_time"`
	EndTime    *time.Time  `form:"end_time"`
	Page       int         `form:"page,default=1"`
	PageSize   int         `form:"page_size,default=20"`
}

// AlarmListResponse 报警列表响应
type AlarmListResponse struct {
	List     []Alarm `json:"list"`
	Total    int64   `json:"total"`
	Page     int     `json:"page"`
	PageSize int     `json:"page_size"`
}

// WSAlarmMessage WebSocket报警消息
type WSAlarmMessage struct {
	Type string      `json:"type"` // "ALARM"
	Data Alarm       `json:"data"`
}
