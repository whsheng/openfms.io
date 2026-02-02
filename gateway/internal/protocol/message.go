package protocol

// StandardMessage represents the unified message format across all protocols
type StandardMessage struct {
	DeviceID  string                 `json:"device_id"`
	Type      string                 `json:"type"`      // "AUTH", "LOCATION", "HEARTBEAT", etc.
	Timestamp int64                  `json:"timestamp"`
	Lat       float64                `json:"lat"`
	Lon       float64                `json:"lon"`
	Speed     float64                `json:"speed"`
	Direction float64                `json:"direction"`
	Extras    map[string]interface{} `json:"extras"`    // 扩展字段：油量、温度、门开关
}

// StandardCommand represents a command to be sent to a device
type StandardCommand struct {
	Type   string                 `json:"type"`
	Params map[string]interface{} `json:"params"`
}

// Message types
const (
	MsgTypeAuth      = "AUTH"
	MsgTypeLocation  = "LOCATION"
	MsgTypeHeartbeat = "HEARTBEAT"
	MsgTypeAlarm     = "ALARM"
	MsgTypeMedia     = "MEDIA"
)
