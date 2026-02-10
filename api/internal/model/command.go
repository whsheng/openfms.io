package model

import (
	"time"
)

// DeviceCommand 设备指令记录
type DeviceCommand struct {
	ID        int       `json:"id" gorm:"primaryKey"`
	DeviceID  string    `json:"device_id" gorm:"column:device_id;type:varchar(20);not null;index"`
	Command   string    `json:"command" gorm:"type:varchar(50);not null"`
	Params    string    `json:"params,omitempty" gorm:"type:jsonb"`
	Status    string    `json:"status" gorm:"type:varchar(20);not null;default:'pending'"` // pending, sent, success, failed, timeout
	Response  string    `json:"response,omitempty" gorm:"type:jsonb"`
	ErrorMsg  string    `json:"error_msg,omitempty" gorm:"column:error_msg;type:text"`
	SentBy    *int      `json:"sent_by,omitempty" gorm:"column:sent_by"`
	CreatedAt time.Time `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt time.Time `json:"updated_at" gorm:"not null;default:now()"`
}

func (DeviceCommand) TableName() string {
	return "device_commands"
}

// CommandType 指令类型常量
const (
	CmdLocationQuery   = "LOCATION_QUERY"   // 位置查询 0x8201
	CmdSetParams       = "SET_PARAMS"       // 设置参数 0x8103
	CmdGetParams       = "GET_PARAMS"       // 查询参数 0x8104
	CmdTerminalControl = "TERMINAL_CONTROL" // 终端控制 0x8105
	CmdVehicleControl  = "VEHICLE_CONTROL"  // 车辆控制 0x8500
	CmdTextMessage     = "TEXT_MESSAGE"     // 文本信息下发 0x8300
)

// SendCommandRequest 发送指令请求
type SendCommandRequest struct {
	Command string                 `json:"command" binding:"required"`
	Params  map[string]interface{} `json:"params"`
	Timeout int                    `json:"timeout"` // 秒，默认30
}

// BatchCommandRequest 批量指令请求
type BatchCommandRequest struct {
	DeviceIDs []string               `json:"device_ids" binding:"required"`
	Command   string                 `json:"command" binding:"required"`
	Params    map[string]interface{} `json:"params"`
}

// CommandResponse 指令响应
type CommandResponse struct {
	CommandID string                 `json:"command_id"`
	DeviceID  string                 `json:"device_id"`
	Success   bool                   `json:"success"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Error     string                 `json:"error,omitempty"`
}
