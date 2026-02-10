// GT06 协议适配器
// GT06 是常见的海外 GPS 设备协议

package adapter

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"

	"openfms/gateway/internal/protocol"
)

// GT06Adapter GT06协议适配器
type GT06Adapter struct{}

// NewGT06Adapter 创建GT06适配器
func NewGT06Adapter() *GT06Adapter {
	return &GT06Adapter{}
}

// Match 匹配GT06协议 (0x78 0x78 开头)
func (a *GT06Adapter) Match(header []byte) bool {
	return len(header) >= 2 && header[0] == 0x78 && header[1] == 0x78
}

// Decode 解码GT06数据包
func (a *GT06Adapter) Decode(packet []byte) (*protocol.StandardMessage, error) {
	if len(packet) < 10 {
		return nil, fmt.Errorf("packet too short")
	}

	// GT06 包结构: 0x78 0x78 + 长度(1) + 协议号(1) + 内容(N) + 序列号(2) + 校验(2) + 0x0D 0x0A
	if packet[0] != 0x78 || packet[1] != 0x78 {
		return nil, fmt.Errorf("invalid header")
	}

	length := packet[2]
	protocolNum := packet[3]

	msg := &protocol.StandardMessage{
		Timestamp: time.Now().Unix(),
		Extras:    make(map[string]interface{}),
	}

	switch protocolNum {
	case 0x01: // 登录包
		msg.Type = "AUTH"
		msg.DeviceID = a.parseDeviceID(packet[4:12])

	case 0x12: // 位置数据包
		msg.Type = "LOCATION"
		msg.DeviceID = a.parseDeviceID(packet[4:12])
		
		// 解析位置信息
		if len(packet) >= 30 {
			dateTime := packet[12:18]
			msg.Timestamp = a.parseDateTime(dateTime)
			
			// GPS 信息
			gpsLen := packet[18]
			satellites := gpsLen & 0x0F
			msg.Extras["satellites"] = satellites
			
			// 经纬度
			lat := binary.BigEndian.Uint32(packet[19:23])
			lon := binary.BigEndian.Uint32(packet[23:27])
			
			msg.Lat = float64(lat) / 30000.0 / 60.0
			msg.Lon = float64(lon) / 30000.0 / 60.0
			
			// 速度
			msg.Speed = float64(packet[27])
			
			// 方向
			course := binary.BigEndian.Uint16(packet[28:30])
			msg.Direction = float64(course & 0x3FF)
		}

	case 0x13: // 心跳包
		msg.Type = "HEARTBEAT"
		msg.DeviceID = a.parseDeviceID(packet[4:12])

	default:
		msg.Type = "UNKNOWN"
	}

	return msg, nil
}

// Encode 编码GT06响应
func (a *GT06Adapter) Encode(cmd protocol.StandardCommand) ([]byte, error) {
	switch cmd.Type {
	case "AUTH_ACK":
		// 登录响应
		return []byte{0x78, 0x78, 0x05, 0x01, 0x00, 0x01, 0x00, 0x00, 0x0D, 0x0A}, nil
	
	case "HEARTBEAT_ACK":
		// 心跳响应
		return []byte{0x78, 0x78, 0x05, 0x13, 0x00, 0x01, 0x00, 0x00, 0x0D, 0x0A}, nil
	
	default:
		return nil, fmt.Errorf("unsupported command: %s", cmd.Type)
	}
}

// IsHeartbeat 判断是否心跳包
func (a *GT06Adapter) IsHeartbeat(packet []byte) bool {
	return len(packet) > 3 && packet[3] == 0x13
}

// GenerateHeartbeatAck 生成心跳响应
func (a *GT06Adapter) GenerateHeartbeatAck(packet []byte) []byte {
	ack, _ := a.Encode(protocol.StandardCommand{Type: "HEARTBEAT_ACK"})
	return ack
}

// 辅助方法

func (a *GT06Adapter) parseDeviceID(data []byte) string {
	// GT06 设备ID是BCD编码的IMEI
	if len(data) < 8 {
		return ""
	}
	var result bytes.Buffer
	for _, b := range data {
		result.WriteString(fmt.Sprintf("%02x", b))
	}
	return result.String()
}

func (a *GT06Adapter) parseDateTime(data []byte) int64 {
	if len(data) < 6 {
		return time.Now().Unix()
	}
	// YY MM DD HH MM SS
	year := 2000 + int(data[0])
	month := time.Month(data[1])
	day := int(data[2])
	hour := int(data[3])
	minute := int(data[4])
	second := int(data[5])
	
	t := time.Date(year, month, day, hour, minute, second, 0, time.UTC)
	return t.Unix()
}
