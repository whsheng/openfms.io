// NATS JetStream 消息持久化服务

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// JetStreamService JetStream服务
type JetStreamService struct {
	nc *nats.Conn
	js nats.JetStreamContext
}

// Stream 配置
const (
	StreamLocations = "FMS_LOCATIONS"
	StreamAlarms    = "FMS_ALARMS"
	StreamCommands  = "FMS_COMMANDS"
	StreamEvents    = "FMS_EVENTS"
)

// NewJetStreamService 创建JetStream服务
func NewJetStreamService(nc *nats.Conn) (*JetStreamService, error) {
	js, err := nc.JetStream()
	if err != nil {
		return nil, fmt.Errorf("failed to create jetstream context: %w", err)
	}

	s := &JetStreamService{
		nc: nc,
		js: js,
	}

	// 初始化Streams
	if err := s.initStreams(); err != nil {
		return nil, err
	}

	return s, nil
}

// initStreams 初始化Streams
func (s *JetStreamService) initStreams() error {
	streams := []nats.StreamConfig{
		{
			Name:     StreamLocations,
			Subjects: []string{"fms.locations.*"},
			Retention: nats.LimitsPolicy,
			MaxMsgs:  -1,
			MaxBytes: 10 * 1024 * 1024 * 1024, // 10GB
			MaxAge:   7 * 24 * time.Hour,      // 7天
			Storage:  nats.FileStorage,
			Replicas: 1,
		},
		{
			Name:     StreamAlarms,
			Subjects: []string{"fms.alarms.*"},
			Retention: nats.LimitsPolicy,
			MaxMsgs:  -1,
			MaxBytes: 5 * 1024 * 1024 * 1024, // 5GB
			MaxAge:   30 * 24 * time.Hour,    // 30天
			Storage:  nats.FileStorage,
			Replicas: 1,
		},
		{
			Name:     StreamCommands,
			Subjects: []string{"fms.commands.*"},
			Retention: nats.WorkQueuePolicy,
			MaxMsgs:  100000,
			MaxAge:   24 * time.Hour, // 1天
			Storage:  nats.FileStorage,
			Replicas: 1,
		},
		{
			Name:     StreamEvents,
			Subjects: []string{"fms.events.*"},
			Retention: nats.LimitsPolicy,
			MaxMsgs:  -1,
			MaxBytes: 2 * 1024 * 1024 * 1024, // 2GB
			MaxAge:   7 * 24 * time.Hour,     // 7天
			Storage:  nats.FileStorage,
			Replicas: 1,
		},
	}

	for _, cfg := range streams {
		_, err := s.js.AddStream(&cfg)
		if err != nil {
			if err == nats.ErrStreamNameAlreadyInUse {
				// Stream已存在，更新配置
				_, err = s.js.UpdateStream(&cfg)
				if err != nil {
					return fmt.Errorf("failed to update stream %s: %w", cfg.Name, err)
				}
			} else {
				return fmt.Errorf("failed to create stream %s: %w", cfg.Name, err)
			}
		}
	}

	return nil
}

// PublishLocation 发布位置消息（持久化）
func (s *JetStreamService) PublishLocation(deviceID string, data interface{}) error {
	subject := fmt.Sprintf("fms.locations.%s", deviceID)
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = s.js.Publish(subject, payload)
	return err
}

// PublishAlarm 发布报警消息（持久化）
func (s *JetStreamService) PublishAlarm(alarmType string, data interface{}) error {
	subject := fmt.Sprintf("fms.alarms.%s", alarmType)
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = s.js.Publish(subject, payload)
	return err
}

// PublishEvent 发布事件消息（持久化）
func (s *JetStreamService) PublishEvent(eventType string, data interface{}) error {
	subject := fmt.Sprintf("fms.events.%s", eventType)
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = s.js.Publish(subject, payload)
	return err
}

// SubscribeLocations 订阅位置消息
func (s *JetStreamService) SubscribeLocations(consumer string, handler func(msg *nats.Msg)) error {
	_, err := s.js.Subscribe("fms.locations.*", handler,
		nats.Durable(consumer),
		nats.ManualAck(),
	)
	return err
}

// SubscribeAlarms 订阅报警消息
func (s *JetStreamService) SubscribeAlarms(alarmType string, consumer string, handler func(msg *nats.Msg)) error {
	subject := fmt.Sprintf("fms.alarms.%s", alarmType)
	_, err := s.js.Subscribe(subject, handler,
		nats.Durable(consumer),
		nats.ManualAck(),
	)
	return err
}

// ReplayMessages 重放历史消息
func (s *JetStreamService) ReplayMessages(stream string, startTime time.Time, handler func(msg *nats.Msg)) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	sub, err := s.js.SubscribeSync(stream,
		nats.StartTime(startTime),
	)
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := sub.NextMsg(1 * time.Second)
			if err != nil {
				if err == nats.ErrTimeout {
					return nil // 没有更多消息
				}
				return err
			}
			handler(msg)
			msg.Ack()
		}
	}
}

// GetStreamInfo 获取Stream信息
func (s *JetStreamService) GetStreamInfo(stream string) (*nats.StreamInfo, error) {
	return s.js.StreamInfo(stream)
}

// PurgeStream 清空Stream
func (s *JetStreamService) PurgeStream(stream string) error {
	return s.js.PurgeStream(stream)
}

// DeleteMessage 删除单条消息
func (s *JetStreamService) DeleteMessage(stream string, seq uint64) error {
	return s.js.DeleteMsg(stream, seq)
}
