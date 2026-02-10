package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"gorm.io/gorm"

	"openfms/api/internal/model"
)

// VideoService 视频服务
type VideoService struct {
	db       *gorm.DB
	natsConn *nats.Conn
	zlmURL   string // ZLMediaKit 地址
}

// NewVideoService 创建视频服务
func NewVideoService(db *gorm.DB, natsConn *nats.Conn, zlmURL string) *VideoService {
	return &VideoService{
		db:       db,
		natsConn: natsConn,
		zlmURL:   zlmURL,
	}
}

// StartRealtimeVideo 开始实时视频
func (s *VideoService) StartRealtimeVideo(deviceID string, channel int, userID int) (*model.VideoStreamResponse, error) {
	// 检查设备是否存在
	var device model.Device
	if err := s.db.Where("sim_no = ?", deviceID).First(&device).Error; err != nil {
		return nil, fmt.Errorf("device not found")
	}

	// 检查是否已有活跃流
	var existingStream model.VideoStream
	s.db.Where("device_id = ? AND channel = ? AND status = ?", deviceID, channel, "streaming").
		First(&existingStream)
	
	if existingStream.ID > 0 {
		// 返回已有流
		return s.buildStreamResponse(&existingStream), nil
	}

	// 创建流记录
	stream := model.VideoStream{
		DeviceID:   deviceID,
		Channel:    channel,
		StreamType: "realtime",
		Status:     "pending",
		CreatedBy:  userID,
	}
	if err := s.db.Create(&stream).Error; err != nil {
		return nil, err
	}

	// 发送指令到设备 (通过 NATS)
	cmd := model.JT1078Command{
		DeviceID: deviceID,
		Channel:  channel,
		Command:  "play",
		Params: map[string]interface{}{
			"stream_id": stream.ID,
			"media_ip":  s.getMediaServerIP(),
			"media_port": 554,
		},
	}
	cmdData, _ := json.Marshal(cmd)
	
	subject := fmt.Sprintf("device.%s.command.video", deviceID)
	if err := s.natsConn.Publish(subject, cmdData); err != nil {
		stream.Status = "error"
		stream.ErrorMsg = err.Error()
		s.db.Save(&stream)
		return nil, err
	}

	// 更新状态
	stream.Status = "streaming"
	stream.StartTime = time.Now()
	stream.StreamURL = s.buildStreamURL(deviceID, channel, stream.ID)
	s.db.Save(&stream)

	return s.buildStreamResponse(&stream), nil
}

// StopVideo 停止视频
func (s *VideoService) StopVideo(streamID int) error {
	var stream model.VideoStream
	if err := s.db.First(&stream, streamID).Error; err != nil {
		return err
	}

	if stream.Status != "streaming" {
		return fmt.Errorf("stream is not active")
	}

	// 发送停止指令
	cmd := model.JT1078Command{
		DeviceID: stream.DeviceID,
		Channel:  stream.Channel,
		Command:  "stop",
	}
	cmdData, _ := json.Marshal(cmd)
	
	subject := fmt.Sprintf("device.%s.command.video", stream.DeviceID)
	s.natsConn.Publish(subject, cmdData)

	// 更新状态
	now := time.Now()
	stream.Status = "stopped"
	stream.EndTime = &now
	s.db.Save(&stream)

	return nil
}

// GetStreamStatus 获取流状态
func (s *VideoService) GetStreamStatus(streamID int) (*model.VideoStreamResponse, error) {
	var stream model.VideoStream
	if err := s.db.First(&stream, streamID).Error; err != nil {
		return nil, err
	}
	return s.buildStreamResponse(&stream), nil
}

// GetActiveStreams 获取活跃流列表
func (s *VideoService) GetActiveStreams(deviceID string) ([]model.VideoStreamResponse, error) {
	var streams []model.VideoStream
	query := s.db.Where("status = ?", "streaming")
	if deviceID != "" {
		query = query.Where("device_id = ?", deviceID)
	}
	query.Find(&streams)

	var responses []model.VideoStreamResponse
	for _, stream := range streams {
		responses = append(responses, *s.buildStreamResponse(&stream))
	}
	return responses, nil
}

// QueryRecords 查询录像
func (s *VideoService) QueryRecords(deviceID string, channel int, startTime, endTime int64) ([]model.VideoRecord, error) {
	var records []model.VideoRecord
	err := s.db.Where("device_id = ? AND channel = ? AND start_time >= ? AND end_time <= ?",
		deviceID, channel, time.Unix(startTime, 0), time.Unix(endTime, 0)).
		Order("start_time DESC").
		Find(&records).Error
	return records, err
}

// StartPlayback 开始回放
func (s *VideoService) StartPlayback(deviceID string, channel int, startTime, endTime int64, userID int) (*model.VideoStreamResponse, error) {
	// 创建回放流记录
	stream := model.VideoStream{
		DeviceID:   deviceID,
		Channel:    channel,
		StreamType: "playback",
		Status:     "pending",
		CreatedBy:  userID,
	}
	if err := s.db.Create(&stream).Error; err != nil {
		return nil, err
	}

	// 发送回放指令
	cmd := model.JT1078Command{
		DeviceID: deviceID,
		Channel:  channel,
		Command:  "playback",
		Params: map[string]interface{}{
			"stream_id":  stream.ID,
			"media_ip":   s.getMediaServerIP(),
			"media_port": 554,
			"start_time": startTime,
			"end_time":   endTime,
		},
	}
	cmdData, _ := json.Marshal(cmd)
	
	subject := fmt.Sprintf("device.%s.command.video", deviceID)
	s.natsConn.Publish(subject, cmdData)

	// 更新状态
	stream.Status = "streaming"
	stream.StartTime = time.Now()
	stream.StreamURL = s.buildStreamURL(deviceID, channel, stream.ID)
	s.db.Save(&stream)

	return s.buildStreamResponse(&stream), nil
}

// ControlPlayback 控制回放
func (s *VideoService) ControlPlayback(streamID int, action string, params map[string]interface{}) error {
	var stream model.VideoStream
	if err := s.db.First(&stream, streamID).Error; err != nil {
		return err
	}

	cmd := model.JT1078Command{
		DeviceID: stream.DeviceID,
		Channel:  stream.Channel,
		Command:  action, // pause, resume, speed, seek
		Params:   params,
	}
	cmdData, _ := json.Marshal(cmd)
	
	subject := fmt.Sprintf("device.%s.command.video", stream.DeviceID)
	return s.natsConn.Publish(subject, cmdData)
}

// TakeSnapshot 截图
func (s *VideoService) TakeSnapshot(deviceID string, channel int) (*model.SnapshotResponse, error) {
	// 调用 ZLMediaKit API 截图
	// 这里简化处理，实际应该调用 ZLMediaKit 的 HTTP API
	snapshotURL := fmt.Sprintf("%s/index/api/getSnap?url=%s&timeout_sec=10&expire_sec=300",
		s.zlmURL,
		s.buildStreamURL(deviceID, channel, 0))

	return &model.SnapshotResponse{
		DeviceID:  deviceID,
		Channel:   channel,
		ImageURL:  snapshotURL,
		Timestamp: time.Now().Unix(),
	}, nil
}

// GetDeviceConfig 获取设备视频配置
func (s *VideoService) GetDeviceConfig(deviceID string) (*model.VideoDeviceConfig, error) {
	var config model.VideoDeviceConfig
	if err := s.db.Where("device_id = ?", deviceID).First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// 返回默认配置
			return &model.VideoDeviceConfig{
				DeviceID:     deviceID,
				ChannelCount: 1,
				VideoCodec:   "H264",
				AudioCodec:   "G711A",
				Resolution:   "D1",
				FrameRate:    25,
				BitRate:      512,
			}, nil
		}
		return nil, err
	}
	return &config, nil
}

// UpdateDeviceConfig 更新设备视频配置
func (s *VideoService) UpdateDeviceConfig(config *model.VideoDeviceConfig) error {
	var existing model.VideoDeviceConfig
	if err := s.db.Where("device_id = ?", config.DeviceID).First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return s.db.Create(config).Error
		}
		return err
	}
	return s.db.Model(&existing).Updates(config).Error
}

// 辅助方法

func (s *VideoService) buildStreamURL(deviceID string, channel, streamID int) string {
	// 生成流地址
	streamKey := fmt.Sprintf("stream_%s_%d_%d", deviceID, channel, streamID)
	return fmt.Sprintf("%s/live/%s", s.zlmURL, streamKey)
}

func (s *VideoService) buildStreamResponse(stream *model.VideoStream) *model.VideoStreamResponse {
	streamKey := fmt.Sprintf("stream_%s_%d_%d", stream.DeviceID, stream.Channel, stream.ID)
	
	return &model.VideoStreamResponse{
		StreamID:  stream.ID,
		DeviceID:  stream.DeviceID,
		Channel:   stream.Channel,
		Status:    stream.Status,
		StreamURL: stream.StreamURL,
		WSFLVURL:  fmt.Sprintf("ws://%s/live/%s.flv", s.getMediaServerHost(), streamKey),
		WebRTCURL: fmt.Sprintf("webrtc://%s/live/%s", s.getMediaServerHost(), streamKey),
		HLSURL:    fmt.Sprintf("%s/live/%s/hls.m3u8", s.zlmURL, streamKey),
	}
}

func (s *VideoService) getMediaServerIP() string {
	// 从配置中获取，这里简化处理
	return "zlmediakit"
}

func (s *VideoService) getMediaServerHost() string {
	// 返回外部可访问的地址
	return "localhost:8088"
}
