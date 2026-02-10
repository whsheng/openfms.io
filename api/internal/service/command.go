// 指令服务 - 支持超时处理、批量下发、历史记录

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"gorm.io/gorm"

	"openfms/api/internal/model"
)

// CommandService 指令服务
type CommandService struct {
	db       *gorm.DB
	natsConn *nats.Conn
	
	// 待响应指令池
	pendingCommands map[string]*PendingCommand
	mu              sync.RWMutex
}

// PendingCommand 待响应指令
type PendingCommand struct {
	ID         string
	DeviceID   string
	Command    string
	Params     map[string]interface{}
	SentAt     time.Time
	Timeout    time.Duration
	Response   chan *CommandResponse
	Status     string // pending, success, timeout, error
}

// CommandResponse 指令响应
type CommandResponse struct {
	DeviceID  string
	CommandID string
	Success   bool
	Data      map[string]interface{}
	Error     string
}

// NewCommandService 创建指令服务
func NewCommandService(db *gorm.DB, natsConn *nats.Conn) *CommandService {
	s := &CommandService{
		db:              db,
		natsConn:        natsConn,
		pendingCommands: make(map[string]*PendingCommand),
	}
	
	// 启动响应监听
	go s.startResponseListener()
	
	return s
}

// SendCommand 发送指令（带超时）
func (s *CommandService) SendCommand(ctx context.Context, deviceID, command string, params map[string]interface{}, timeout time.Duration) (*CommandResponse, error) {
	// 生成指令ID
	cmdID := fmt.Sprintf("%s_%d", deviceID, time.Now().UnixNano())
	
	// 创建待响应指令
	pending := &PendingCommand{
		ID:       cmdID,
		DeviceID: deviceID,
		Command:  command,
		Params:   params,
		SentAt:   time.Now(),
		Timeout:  timeout,
		Response: make(chan *CommandResponse, 1),
		Status:   "pending",
	}
	
	s.mu.Lock()
	s.pendingCommands[cmdID] = pending
	s.mu.Unlock()
	
	// 保存到数据库
	cmdRecord := model.DeviceCommand{
		DeviceID: deviceID,
		Command:  command,
		Params:   mustJSON(params),
		Status:   "pending",
	}
	s.db.Create(&cmdRecord)
	
	// 发送指令
	msg := map[string]interface{}{
		"command_id": cmdID,
		"device_id":  deviceID,
		"command":    command,
		"params":     params,
		"timestamp":  time.Now().Unix(),
	}
	msgData, _ := json.Marshal(msg)
	
	subject := fmt.Sprintf("device.%s.command", deviceID)
	if err := s.natsConn.Publish(subject, msgData); err != nil {
		pending.Status = "error"
		s.db.Model(&cmdRecord).Updates(map[string]interface{}{
			"status":     "failed",
			"error_msg":  err.Error(),
			"updated_at": time.Now(),
		})
		return nil, err
	}
	
	// 等待响应或超时
	select {
	case resp := <-pending.Response:
		pending.Status = "success"
		s.db.Model(&cmdRecord).Updates(map[string]interface{}{
			"status":     "success",
			"response":   mustJSON(resp.Data),
			"updated_at": time.Now(),
		})
		return resp, nil
		
	case <-time.After(timeout):
		pending.Status = "timeout"
		s.db.Model(&cmdRecord).Updates(map[string]interface{}{
			"status":     "timeout",
			"updated_at": time.Now(),
		})
		return nil, fmt.Errorf("command timeout after %v", timeout)
		
	case <-ctx.Done():
		pending.Status = "cancelled"
		return nil, ctx.Err()
	}
}

// SendCommandAsync 异步发送指令（不等待响应）
func (s *CommandService) SendCommandAsync(deviceID, command string, params map[string]interface{}) (string, error) {
	cmdID := fmt.Sprintf("%s_%d", deviceID, time.Now().UnixNano())
	
	// 保存到数据库
	cmdRecord := model.DeviceCommand{
		DeviceID: deviceID,
		Command:  command,
		Params:   mustJSON(params),
		Status:   "sent",
	}
	s.db.Create(&cmdRecord)
	
	// 发送指令
	msg := map[string]interface{}{
		"command_id": cmdID,
		"device_id":  deviceID,
		"command":    command,
		"params":     params,
		"timestamp":  time.Now().Unix(),
	}
	msgData, _ := json.Marshal(msg)
	
	subject := fmt.Sprintf("device.%s.command", deviceID)
	if err := s.natsConn.Publish(subject, msgData); err != nil {
		s.db.Model(&cmdRecord).Updates(map[string]interface{}{
			"status":     "failed",
			"error_msg":  err.Error(),
			"updated_at": time.Now(),
		})
		return "", err
	}
	
	return cmdID, nil
}

// BatchSendCommand 批量发送指令
func (s *CommandService) BatchSendCommand(deviceIDs []string, command string, params map[string]interface{}) map[string]string {
	results := make(map[string]string)
	
	var wg sync.WaitGroup
	var mu sync.Mutex
	
	for _, deviceID := range deviceIDs {
		wg.Add(1)
		go func(did string) {
			defer wg.Done()
			
			cmdID, err := s.SendCommandAsync(did, command, params)
			
			mu.Lock()
			if err != nil {
				results[did] = "error: " + err.Error()
			} else {
				results[did] = cmdID
			}
			mu.Unlock()
		}(deviceID)
	}
	
	wg.Wait()
	return results
}

// GetCommandHistory 获取指令历史
func (s *CommandService) GetCommandHistory(deviceID string, limit int) ([]model.DeviceCommand, error) {
	var commands []model.DeviceCommand
	query := s.db.Where("device_id = ?", deviceID).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&commands).Error
	return commands, err
}

// startResponseListener 启动响应监听
func (s *CommandService) startResponseListener() {
	s.natsConn.Subscribe("device.>.command.response", func(msg *nats.Msg) {
		var resp CommandResponse
		if err := json.Unmarshal(msg.Data, &resp); err != nil {
			return
		}
		
		s.mu.RLock()
		pending, exists := s.pendingCommands[resp.CommandID]
		s.mu.RUnlock()
		
		if exists {
			select {
			case pending.Response <- &resp:
			default:
			}
		}
	})
}

// mustJSON 转换为JSON
func mustJSON(v interface{}) string {
	data, _ := json.Marshal(v)
	return string(data)
}
