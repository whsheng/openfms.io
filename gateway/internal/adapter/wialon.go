// Wialon IPS 协议适配器
// Wialon IPS 是 Wialon 平台的通用协议

package adapter

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"openfms/gateway/internal/protocol"
)

// WialonAdapter Wialon IPS协议适配器
type WialonAdapter struct{}

// NewWialonAdapter 创建Wialon适配器
func NewWialonAdapter() *WialonAdapter {
	return &WialonAdapter{}
}

// Match 匹配Wialon协议 (文本协议，以 # 开头)
func (a *WialonAdapter) Match(header []byte) bool {
	return len(header) > 0 && (header[0] == '#' || header[0] == '$')
}

// Decode 解码Wialon数据包
func (a *WialonAdapter) Decode(packet []byte) (*protocol.StandardMessage, error) {
	packetStr := string(packet)
	packetStr = strings.TrimSpace(packetStr)

	msg := &protocol.StandardMessage{
		Timestamp: time.Now().Unix(),
		Extras:    make(map[string]interface{}),
	}

	// 解析消息类型
	if strings.HasPrefix(packetStr, "#L#") {
		// 登录消息
		msg.Type = "AUTH"
		parts := strings.Split(packetStr[3:], ";")
		if len(parts) >= 1 {
			msg.DeviceID = parts[0]
		}
		return msg, nil
	}

	if strings.HasPrefix(packetStr, "#SD#") || strings.HasPrefix(packetStr, "#D#") {
		// 短数据或数据消息
		msg.Type = "LOCATION"
		prefix := "#SD#"
		if strings.HasPrefix(packetStr, "#D#") {
			prefix = "#D#"
		}
		
		parts := strings.Split(packetStr[len(prefix):], ";")
		if len(parts) >= 6 {
			msg.DeviceID = parts[0]
			
			// 解析日期时间
			if len(parts) >= 2 {
				msg.Timestamp = a.parseDateTime(parts[1])
			}
			
			// 解析经纬度
			if len(parts) >= 5 {
				lat, _ := strconv.ParseFloat(parts[2], 64)
				lon, _ := strconv.ParseFloat(parts[4], 64)
				
				// Wialon 使用度分格式，需要转换
				msg.Lat = a.convertCoord(lat)
				msg.Lon = a.convertCoord(lon)
			}
			
			// 解析速度和方向
			if len(parts) >= 7 {
				speed, _ := strconv.ParseFloat(parts[5], 64)
				msg.Speed = speed
			}
			if len(parts) >= 8 {
				direction, _ := strconv.ParseFloat(parts[6], 64)
				msg.Direction = direction
			}
			
			// 解析扩展数据
			if len(parts) >= 9 {
				extras := parts[8]
				// 解析 IO 数据，如：IO=ignition:1;fuel:50
				if strings.HasPrefix(extras, "IO=") {
					ioData := extras[3:]
					ioParts := strings.Split(ioData, ",")
					for _, io := range ioParts {
						kv := strings.Split(io, ":")
						if len(kv) == 2 {
							msg.Extras[kv[0]] = kv[1]
						}
					}
				}
			}
		}
		return msg, nil
	}

	if strings.HasPrefix(packetStr, "#P#") {
		// 心跳包
		msg.Type = "HEARTBEAT"
		parts := strings.Split(packetStr[3:], ";")
		if len(parts) >= 1 {
			msg.DeviceID = parts[0]
		}
		return msg, nil
	}

	if strings.HasPrefix(packetStr, "#B#") {
		// 批量数据
		msg.Type = "BATCH"
		// 批量数据解析逻辑...
		return msg, nil
	}

	msg.Type = "UNKNOWN"
	return msg, nil
}

// Encode 编码Wialon响应
func (a *WialonAdapter) Encode(cmd protocol.StandardCommand) ([]byte, error) {
	switch cmd.Type {
	case "AUTH_ACK":
		return []byte("#AL#1\r\n"), nil
	
	case "HEARTBEAT_ACK":
		return []byte("#AP#\r\n"), nil
	
	case "DATA_ACK":
		return []byte("#AD#1\r\n"), nil
	
	default:
		return nil, fmt.Errorf("unsupported command: %s", cmd.Type)
	}
}

// IsHeartbeat 判断是否心跳包
func (a *WialonAdapter) IsHeartbeat(packet []byte) bool {
	return bytes.HasPrefix(packet, []byte("#P#"))
}

// GenerateHeartbeatAck 生成心跳响应
func (a *WialonAdapter) GenerateHeartbeatAck(packet []byte) []byte {
	ack, _ := a.Encode(protocol.StandardCommand{Type: "HEARTBEAT_ACK"})
	return ack
}

// ReadPacket 从reader读取一个完整包
func (a *WialonAdapter) ReadPacket(reader *bufio.Reader) ([]byte, error) {
	// Wialon 是文本协议，以 \r\n 结尾
	return reader.ReadBytes('\n')
}

// 辅助方法

func (a *WialonAdapter) parseDateTime(dt string) int64 {
	// 格式: DDMMYY;HHMMSS
	parts := strings.Split(dt, ";")
	if len(parts) != 2 {
		return time.Now().Unix()
	}
	
	dateStr := parts[0] // DDMMYY
	timeStr := parts[1] // HHMMSS
	
	if len(dateStr) != 6 || len(timeStr) != 6 {
		return time.Now().Unix()
	}
	
	day, _ := strconv.Atoi(dateStr[0:2])
	month, _ := strconv.Atoi(dateStr[2:4])
	year, _ := strconv.Atoi(dateStr[4:6])
	
	hour, _ := strconv.Atoi(timeStr[0:2])
	minute, _ := strconv.Atoi(timeStr[2:4])
	second, _ := strconv.Atoi(timeStr[4:6])
	
	t := time.Date(2000+year, time.Month(month), day, hour, minute, second, 0, time.UTC)
	return t.Unix()
}

func (a *WialonAdapter) convertCoord(coord float64) float64 {
	// Wialon 使用度分格式: DDMM.MMMM
	// 转换为度: DD + MM.MMMM/60
	degrees := float64(int(coord / 100))
	minutes := coord - degrees*100
	return degrees + minutes/60
}
