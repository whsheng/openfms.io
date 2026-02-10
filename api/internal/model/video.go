package model

import (
	"time"
)

// VideoStream 视频流会话
type VideoStream struct {
	ID          int       `json:"id" gorm:"primaryKey"`
	DeviceID    string    `json:"device_id" gorm:"column:device_id;type:varchar(20);not null;index"`
	Channel     int       `json:"channel" gorm:"not null;default:1"` // 通道号 1-8
	StreamType  string    `json:"stream_type" gorm:"type:varchar(20);not null"` // realtime, playback
	StreamURL   string    `json:"stream_url" gorm:"column:stream_url;type:varchar(500)"`
	Status      string    `json:"status" gorm:"type:varchar(20);not null;default:'pending'"` // pending, streaming, stopped, error
	StartTime   time.Time `json:"start_time" gorm:"column:start_time"`
	EndTime     *time.Time `json:"end_time,omitempty" gorm:"column:end_time"`
	ErrorMsg    string    `json:"error_msg,omitempty" gorm:"column:error_msg;type:text"`
	CreatedBy   int       `json:"created_by" gorm:"column:created_by"`
	CreatedAt   time.Time `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"not null;default:now()"`
}

func (VideoStream) TableName() string {
	return "video_streams"
}

// VideoRecord 录像记录
type VideoRecord struct {
	ID          int       `json:"id" gorm:"primaryKey"`
	DeviceID    string    `json:"device_id" gorm:"column:device_id;type:varchar(20);not null;index"`
	Channel     int       `json:"channel" gorm:"not null;default:1"`
	StartTime   time.Time `json:"start_time" gorm:"column:start_time;not null"`
	EndTime     time.Time `json:"end_time" gorm:"column:end_time;not null"`
	Duration    int       `json:"duration" gorm:"not null"` // 秒
	FileSize    int64     `json:"file_size" gorm:"column:file_size"` // 字节
	FilePath    string    `json:"file_path" gorm:"column:file_path;type:varchar(500)"`
	RecordType  string    `json:"record_type" gorm:"type:varchar(20);not null;default:'auto'"` // auto, alarm, manual
	AlarmID     *int      `json:"alarm_id,omitempty" gorm:"column:alarm_id"`
	CreatedAt   time.Time `json:"created_at" gorm:"not null;default:now()"`
}

func (VideoRecord) TableName() string {
	return "video_records"
}

// VideoDeviceConfig 设备视频配置
type VideoDeviceConfig struct {
	ID              int       `json:"id" gorm:"primaryKey"`
	DeviceID        string    `json:"device_id" gorm:"column:device_id;type:varchar(20);not null;uniqueIndex"`
	ChannelCount    int       `json:"channel_count" gorm:"column:channel_count;not null;default:1"`
	VideoCodec      string    `json:"video_codec" gorm:"column:video_codec;type:varchar(20);default:'H264'"` // H264, H265
	AudioCodec      string    `json:"audio_codec" gorm:"column:audio_codec;type:varchar(20);default:'G711A'"` // G711A, G711U, AAC
	Resolution      string    `json:"resolution" gorm:"type:varchar(20);default:'D1'"` // QCIF, CIF, HD1, D1, 720P, 1080P
	FrameRate       int       `json:"frame_rate" gorm:"column:frame_rate;default:25"`
	BitRate         int       `json:"bit_rate" gorm:"column:bit_rate;default:512"` // kbps
	StoragePolicy   string    `json:"storage_policy" gorm:"type:varchar(20);default:'overlay'"` // overlay, stop
	StorageDays     int       `json:"storage_days" gorm:"column:storage_days;default:7"`
	CreatedAt       time.Time `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"not null;default:now()"`
}

func (VideoDeviceConfig) TableName() string {
	return "video_device_configs"
}

// StartVideoRequest 开始视频请求
type StartVideoRequest struct {
	DeviceID   string `json:"device_id" binding:"required"`
	Channel    int    `json:"channel" binding:"required,min=1,max=8"`
	StreamType string `json:"stream_type" binding:"required,oneof=realtime playback"`
	StartTime  *int64 `json:"start_time,omitempty"` // 回放开始时间戳
	EndTime    *int64 `json:"end_time,omitempty"`   // 回放结束时间戳
}

// StopVideoRequest 停止视频请求
type StopVideoRequest struct {
	StreamID int `json:"stream_id" binding:"required"`
}

// VideoStreamResponse 视频流响应
type VideoStreamResponse struct {
	StreamID   int    `json:"stream_id"`
	DeviceID   string `json:"device_id"`
	Channel    int    `json:"channel"`
	Status     string `json:"status"`
	StreamURL  string `json:"stream_url,omitempty"`
	WSFLVURL   string `json:"ws_flv_url,omitempty"`   // WebSocket-FLV
	WebRTCURL  string `json:"webrtc_url,omitempty"`
	HLSURL     string `json:"hls_url,omitempty"`
}

// VideoRecordQuery 录像查询
type VideoRecordQuery struct {
	DeviceID   string `form:"device_id" binding:"required"`
	Channel    int    `form:"channel,default=1"`
	StartTime  int64  `form:"start_time" binding:"required"`
	EndTime    int64  `form:"end_time" binding:"required"`
	RecordType string `form:"record_type"`
}

// SnapshotRequest 截图请求
type SnapshotRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
	Channel  int    `json:"channel" binding:"required,min=1,max=8"`
}

// SnapshotResponse 截图响应
type SnapshotResponse struct {
	DeviceID   string `json:"device_id"`
	Channel    int    `json:"channel"`
	ImageURL   string `json:"image_url"`
	Timestamp  int64  `json:"timestamp"`
}

// JT1078Command JT1078相关指令
type JT1078Command struct {
	DeviceID   string `json:"device_id"`
	Channel    int    `json:"channel"`
	Command    string `json:"command"` // play, stop, pause, resume, speed
	Params     map[string]interface{} `json:"params,omitempty"`
}
