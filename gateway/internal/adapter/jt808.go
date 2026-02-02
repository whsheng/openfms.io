package adapter

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"openfms/gateway/internal/protocol"
)

const (
	// JT808 protocol constants
	JT808Header byte = 0x7E

	// Message IDs
	MsgIDTerminalAuth     uint16 = 0x0102
	MsgIDLocationReport   uint16 = 0x0200
	MsgIDHeartbeat        uint16 = 0x0002
	MsgIDTerminalRegister uint16 = 0x0100

	// Server response IDs
	MsgIDPlatformGeneralAck uint16 = 0x8001
)

// JT808Adapter implements ProtocolAdapter for JT808 protocol
type JT808Adapter struct{}

// NewJT808Adapter creates a new JT808 adapter
func NewJT808Adapter() *JT808Adapter {
	return &JT808Adapter{}
}

// Protocol returns protocol identifier
func (j *JT808Adapter) Protocol() string {
	return "JT808"
}

// Decode translates JT808 packet to standard message
func (j *JT808Adapter) Decode(packet []byte) (*protocol.StandardMessage, error) {
	if len(packet) < 12 {
		return nil, errors.New("packet too short")
	}

	// Unescape: 0x7d 0x02 -> 0x7e, 0x7d 0x01 -> 0x7d
	unescaped := j.unescape(packet)

	// Parse header
	if unescaped[0] != JT808Header || unescaped[len(unescaped)-1] != JT808Header {
		return nil, errors.New("invalid packet format")
	}

	// Remove start/end markers
	content := unescaped[1 : len(unescaped)-1]

	// Verify checksum
	if !j.verifyChecksum(content) {
		return nil, errors.New("checksum mismatch")
	}

	// Parse message header
	msgID := binary.BigEndian.Uint16(content[0:2])
	bodyProps := binary.BigEndian.Uint16(content[2:4])
	phoneNum := bcdToString(content[4:10])
	msgSerial := binary.BigEndian.Uint16(content[10:12])

	// Check if body is encrypted (simplified - assume no encryption for now)
	_ = bodyProps
	_ = msgSerial

	// Body starts after header (12 bytes)
	body := content[12 : len(content)-1]

	msg := &protocol.StandardMessage{
		DeviceID:  phoneNum,
		Timestamp: time.Now().Unix(),
		Extras:    make(map[string]interface{}),
	}

	switch msgID {
	case MsgIDTerminalAuth:
		msg.Type = protocol.MsgTypeAuth
		if len(body) > 0 {
			authCodeLen := int(body[0])
			if len(body) >= 1+authCodeLen {
				msg.Extras["auth_code"] = string(body[1 : 1+authCodeLen])
			}
		}

	case MsgIDLocationReport:
		msg.Type = protocol.MsgTypeLocation
		if err := j.parseLocation(body, msg); err != nil {
			return nil, err
		}

	case MsgIDHeartbeat:
		msg.Type = protocol.MsgTypeHeartbeat

	case MsgIDTerminalRegister:
		msg.Type = "REGISTER"
		if len(body) >= 10 {
			msg.Extras["province_id"] = binary.BigEndian.Uint16(body[0:2])
			msg.Extras["city_id"] = binary.BigEndian.Uint16(body[2:4])
			msg.Extras["manufacturer_id"] = string(body[4:9])
			msg.Extras["terminal_model"] = string(body[9:17])
		}

	default:
		msg.Type = fmt.Sprintf("UNKNOWN_0x%04X", msgID)
	}

	return msg, nil
}

// Encode translates standard command to JT808 binary
func (j *JT808Adapter) Encode(cmd protocol.StandardCommand) ([]byte, error) {
	switch cmd.Type {
	case "GENERAL_ACK":
		return j.encodeGeneralAck(cmd.Params)
	default:
		return nil, fmt.Errorf("unsupported command type: %s", cmd.Type)
	}
}

// IsHeartbeat checks if packet is a heartbeat
func (j *JT808Adapter) IsHeartbeat(packet []byte) bool {
	if len(packet) < 4 {
		return false
	}
	unescaped := j.unescape(packet)
	if len(unescaped) < 4 {
		return false
	}
	msgID := binary.BigEndian.Uint16(unescaped[1:3])
	return msgID == MsgIDHeartbeat
}

// GenerateHeartbeatAck creates heartbeat acknowledgment
func (j *JT808Adapter) GenerateHeartbeatAck(packet []byte) ([]byte, error) {
	// Parse original packet to get phone number and serial
	unescaped := j.unescape(packet)
	content := unescaped[1 : len(unescaped)-1]

	phoneNum := content[4:10]
	msgSerial := content[10:12]

	// Build ACK: MsgID(0x8001) + BodyProps + Phone + Serial + Result(0)
	ackBody := make([]byte, 5)
	copy(ackBody[0:2], content[0:2]) // Original MsgID
	copy(ackBody[2:4], msgSerial)    // Original Serial
	ackBody[4] = 0                   // Result: 0 = success

	return j.buildPacket(MsgIDPlatformGeneralAck, phoneNum, ackBody), nil
}

// Helper functions

func (j *JT808Adapter) unescape(data []byte) []byte {
	result := make([]byte, 0, len(data))
	for i := 0; i < len(data); i++ {
		if data[i] == 0x7d && i+1 < len(data) {
			if data[i+1] == 0x02 {
				result = append(result, 0x7e)
				i++
				continue
			} else if data[i+1] == 0x01 {
				result = append(result, 0x7d)
				i++
				continue
			}
		}
		result = append(result, data[i])
	}
	return result
}

func (j *JT808Adapter) escape(data []byte) []byte {
	result := make([]byte, 0, len(data))
	for _, b := range data {
		switch b {
		case 0x7e:
			result = append(result, 0x7d, 0x02)
		case 0x7d:
			result = append(result, 0x7d, 0x01)
		default:
			result = append(result, b)
		}
	}
	return result
}

func (j *JT808Adapter) verifyChecksum(data []byte) bool {
	if len(data) < 2 {
		return false
	}
	var checksum byte
	for i := 0; i < len(data)-1; i++ {
		checksum ^= data[i]
	}
	return checksum == data[len(data)-1]
}

func (j *JT808Adapter) calculateChecksum(data []byte) byte {
	var checksum byte
	for _, b := range data {
		checksum ^= b
	}
	return checksum
}

func (j *JT808Adapter) buildPacket(msgID uint16, phoneNum []byte, body []byte) []byte {
	// Header: MsgID(2) + BodyProps(2) + Phone(6) + Serial(2) = 12 bytes
	header := make([]byte, 12)
	binary.BigEndian.PutUint16(header[0:2], msgID)
	// Body properties: length + encryption + subpackage
	bodyProps := uint16(len(body)) & 0x03FF
	binary.BigEndian.PutUint16(header[2:4], bodyProps)
	copy(header[4:10], phoneNum)
	// Use serial 0 for now
	binary.BigEndian.PutUint16(header[10:12], 0)

	// Combine header + body
	content := append(header, body...)

	// Calculate and append checksum
	checksum := j.calculateChecksum(content)
	content = append(content, checksum)

	// Escape and wrap with markers
	escaped := j.escape(content)
	packet := append([]byte{JT808Header}, escaped...)
	packet = append(packet, JT808Header)

	return packet
}

func (j *JT808Adapter) parseLocation(body []byte, msg *protocol.StandardMessage) error {
	if len(body) < 28 {
		return errors.New("location body too short")
	}

	// Alarm flag (4 bytes)
	alarmFlag := binary.BigEndian.Uint32(body[0:4])
	msg.Extras["alarm_flag"] = alarmFlag

	// Status (4 bytes)
	status := binary.BigEndian.Uint32(body[4:8])
	msg.Extras["status"] = status
	msg.Extras["acc_on"] = (status & 0x00000001) != 0
	msg.Extras["location_valid"] = (status & 0x00000002) != 0

	// Latitude (4 bytes) - in 1/1000000 degree
	lat := binary.BigEndian.Uint32(body[8:12])
	msg.Lat = float64(int32(lat)) / 1000000.0

	// Longitude (4 bytes) - in 1/1000000 degree
	lon := binary.BigEndian.Uint32(body[12:16])
	msg.Lon = float64(int32(lon)) / 1000000.0

	// Altitude (2 bytes) - in meters
	altitude := binary.BigEndian.Uint16(body[16:18])
	msg.Extras["altitude"] = altitude

	// Speed (2 bytes) - in 0.1 km/h
	speed := binary.BigEndian.Uint16(body[18:20])
	msg.Speed = float64(speed) / 10.0

	// Direction (2 bytes) - in degrees
	direction := binary.BigEndian.Uint16(body[20:22])
	msg.Direction = float64(direction)

	// Time (6 bytes) - BCD: YYMMDDHHMMSS
	timeStr := bcdToString(body[22:28])
	msg.Extras["gps_time"] = timeStr

	// Parse additional info (28 bytes onwards)
	if len(body) > 28 {
		j.parseLocationExtras(body[28:], msg)
	}

	return nil
}

func (j *JT808Adapter) parseLocationExtras(data []byte, msg *protocol.StandardMessage) {
	for len(data) >= 2 {
		id := data[0]
		length := int(data[1])
		if len(data) < 2+length {
			break
		}
		value := data[2 : 2+length]

		switch id {
		case 0x01: // Mileage (4 bytes, 0.1 km)
			if length >= 4 {
				mileage := binary.BigEndian.Uint32(value[0:4])
				msg.Extras["mileage"] = float64(mileage) / 10.0
			}
		case 0x02: // Fuel (2 bytes, 0.1 L)
			if length >= 2 {
				fuel := binary.BigEndian.Uint16(value[0:2])
				msg.Extras["fuel"] = float64(fuel) / 10.0
			}
		case 0x03: // Speed from sensor (2 bytes, 0.1 km/h)
			if length >= 2 {
				sensorSpeed := binary.BigEndian.Uint16(value[0:2])
				msg.Extras["sensor_speed"] = float64(sensorSpeed) / 10.0
			}
		case 0x25: // Signal strength
			if length >= 1 {
				msg.Extras["signal_strength"] = value[0]
			}
		case 0x30: // WiFi signal
			if length >= 1 {
				msg.Extras["wifi_signal"] = value[0]
			}
		}

		data = data[2+length:]
	}
}

func (j *JT808Adapter) encodeGeneralAck(params map[string]interface{}) ([]byte, error) {
	body := make([]byte, 5)

	// Original MsgID
	if msgID, ok := params["msg_id"].(uint16); ok {
		binary.BigEndian.PutUint16(body[0:2], msgID)
	}

	// Original Serial
	if serial, ok := params["serial"].(uint16); ok {
		binary.BigEndian.PutUint16(body[2:4], serial)
	}

	// Result: 0 = success, 1 = fail, 2 = msg error, 3 = not supported
	body[4] = 0

	phoneNum := make([]byte, 6)
	if phone, ok := params["phone"].(string); ok && len(phone) == 12 {
		// Convert BCD
		for i := 0; i < 6; i++ {
			high := phone[i*2] - '0'
			low := phone[i*2+1] - '0'
			phoneNum[i] = (high << 4) | low
		}
	}

	return j.buildPacket(MsgIDPlatformGeneralAck, phoneNum, body), nil
}

// bcdToString converts BCD encoded bytes to string
func bcdToString(bcd []byte) string {
	var result []byte
	for _, b := range bcd {
		high := (b >> 4) & 0x0F
		low := b & 0x0F
		if high < 10 {
			result = append(result, '0'+high)
		}
		if low < 10 {
			result = append(result, '0'+low)
		}
	}
	return string(result)
}

// stringToBCD converts string to BCD encoded bytes
func stringToBCD(s string) []byte {
	// Pad with leading zero if odd length
	if len(s)%2 == 1 {
		s = "0" + s
	}

	result := make([]byte, len(s)/2)
	for i := 0; i < len(s); i += 2 {
		high := s[i] - '0'
		low := s[i+1] - '0'
		result[i/2] = (high << 4) | low
	}
	return result
}

// JT808Detector implements protocol detection for JT808
type JT808Detector struct {
	adapter *JT808Adapter
}

// NewJT808Detector creates a new JT808 detector
func NewJT808Detector() *JT808Detector {
	return &JT808Detector{
		adapter: NewJT808Adapter(),
	}
}

// Match detects JT808 protocol from header bytes
func (d *JT808Detector) Match(headerBytes []byte) (protocol.ProtocolAdapter, bool) {
	if len(headerBytes) < 1 {
		return nil, false
	}
	// JT808 packets start with 0x7E
	if headerBytes[0] == JT808Header {
		return d.adapter, true
	}
	return nil, false
}
