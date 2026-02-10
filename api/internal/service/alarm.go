package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"gorm.io/gorm"

	"openfms/api/internal/model"
)

// AlarmService 报警服务
type AlarmService struct {
	db             *gorm.DB
	natsConn       *nats.Conn
	wsHub          WSHubInterface
	webhookService *WebhookService
	jetstream      *JetStreamService
}

// NewAlarmService 创建报警服务
func NewAlarmService(db *gorm.DB, natsConn *nats.Conn, wsHub WSHubInterface, jetstream *JetStreamService) *AlarmService {
	return &AlarmService{
		db:             db,
		natsConn:       natsConn,
		wsHub:          wsHub,
		webhookService: NewWebhookService(db),
		jetstream:      jetstream,
	}
}

// Start 启动报警服务
func (s *AlarmService) Start() error {
	// 订阅各类报警消息 (普通 NATS 订阅用于实时处理)
	subjects := []string{
		"fms.alarm.GEOFENCE_ENTER",
		"fms.alarm.GEOFENCE_EXIT",
		"fms.alarm.OVERSPEED",
		"fms.alarm.LOW_BATTERY",
		"fms.alarm.OFFLINE",
		"fms.alarm.SOS",
	}

	for _, subject := range subjects {
		_, err := s.natsConn.Subscribe(subject, s.handleAlarmMessage)
		if err != nil {
			return fmt.Errorf("subscribe %s failed: %w", subject, err)
		}
	}

	// 启动 JetStream 消费者（如果启用）
	if s.jetstream != nil && s.jetstream.IsEnabled() {
		ctx := context.Background()
		
		// 订阅 JetStream 报警消息用于持久化
		go func() {
			if err := s.jetstream.SubscribeAlarms(ctx, s.handleJetStreamAlarm); err != nil {
				fmt.Printf("[AlarmService] Failed to subscribe to JetStream alarms: %v\n", err)
			}
		}()
	}

	// 启动离线检测任务
	go s.offlineChecker()

	return nil
}

// handleAlarmMessage 处理报警消息 (NATS Core)
func (s *AlarmService) handleAlarmMessage(msg *nats.Msg) {
	var alarmMsg struct {
		Type         model.AlarmType `json:"type"`
		DeviceID     string          `json:"device_id"`
		DeviceName   string          `json:"device_name"`
		Lat          float64         `json:"lat"`
		Lon          float64         `json:"lon"`
		Speed        int16           `json:"speed"`
		GeofenceID   int             `json:"geofence_id"`
		GeofenceName string          `json:"geofence_name"`
		Extras       json.RawMessage `json:"extras"`
	}

	if err := json.Unmarshal(msg.Data, &alarmMsg); err != nil {
		fmt.Printf("parse alarm message error: %v\n", err)
		return
	}

	// 检查是否需要静默
	if s.isSilenced(alarmMsg.DeviceID, alarmMsg.Type) {
		return
	}

	// 检查规则是否启用
	if !s.isRuleEnabled(alarmMsg.Type, alarmMsg.DeviceID) {
		return
	}

	// 创建报警记录
	alarm := model.Alarm{
		Type:         alarmMsg.Type,
		Level:        s.getAlarmLevel(alarmMsg.Type),
		DeviceID:     alarmMsg.DeviceID,
		DeviceName:   alarmMsg.DeviceName,
		Title:        s.getAlarmTitle(alarmMsg.Type),
		Content:      s.getAlarmContent(alarmMsg),
		Lat:          &alarmMsg.Lat,
		Lon:          &alarmMsg.Lon,
		Speed:        &alarmMsg.Speed,
		GeofenceID:   &alarmMsg.GeofenceID,
		GeofenceName: alarmMsg.GeofenceName,
		Extras:       alarmMsg.Extras,
		Status:       model.AlarmStatusUnread,
	}

	if err := s.db.Create(&alarm).Error; err != nil {
		fmt.Printf("create alarm error: %v\n", err)
		return
	}

	// 发布到 JetStream 进行持久化
	if s.jetstream != nil && s.jetstream.IsEnabled() {
		ctx := context.Background()
		s.publishAlarmToJetStream(ctx, &alarm)
	}

	// WebSocket 推送
	if s.wsHub != nil {
		msg := map[string]interface{}{
			"type": "ALARM",
			"data": alarm,
		}
		data, _ := json.Marshal(msg)
		s.wsHub.Broadcast(data)
	}

	// Webhook 通知（使用新的 webhook 服务）
	if s.webhookService != nil {
		s.webhookService.TriggerAlarmEvent(context.Background(), &alarm, string(model.WebhookEventAlarmCreated))
	}
}

// handleJetStreamAlarm 处理 JetStream 报警消息
func (s *AlarmService) handleJetStreamAlarm(msg *AlarmMessage) error {
	// 这里可以进行额外的处理，如：
	// 1. 数据同步到其他系统
	// 2. 触发数据分析
	// 3. 更新缓存等
	
	// 目前仅记录日志，因为实际存储已在 handleAlarmMessage 中完成
	fmt.Printf("[AlarmService] JetStream alarm processed: %s - %s\n", msg.Type, msg.Title)
	return nil
}

// publishAlarmToJetStream 发布报警到 JetStream
func (s *AlarmService) publishAlarmToJetStream(ctx context.Context, alarm *model.Alarm) {
	alarmMsg := &AlarmMessage{
		ID:           alarm.ID,
		Type:         string(alarm.Type),
		Level:        string(alarm.Level),
		DeviceID:     alarm.DeviceID,
		DeviceName:   alarm.DeviceName,
		Title:        alarm.Title,
		Content:      alarm.Content,
		Lat:          alarm.Lat,
		Lon:          alarm.Lon,
		Speed:        alarm.Speed,
		GeofenceID:   alarm.GeofenceID,
		GeofenceName: alarm.GeofenceName,
		Extras:       alarm.Extras,
		Timestamp:    alarm.CreatedAt,
	}

	if err := s.jetstream.PublishAlarm(ctx, alarmMsg); err != nil {
		fmt.Printf("[AlarmService] Failed to publish alarm to JetStream: %v\n", err)
	}
}

// CheckOverspeed 检查超速
func (s *AlarmService) CheckOverspeed(deviceID string, speed int16, lat, lon float64) {
	// 获取设备绑定的报警规则
	var rules []model.AlarmRule
	s.db.Where("type = ? AND enabled = ?", model.AlarmTypeOverspeed, true).Find(&rules)

	for _, rule := range rules {
		var conditions struct {
			SpeedLimit int16 `json:"speed_limit"`
		}
		json.Unmarshal(rule.Conditions, &conditions)

		if conditions.SpeedLimit > 0 && speed > conditions.SpeedLimit {
			// 触发超速报警
			msg := map[string]interface{}{
				"type":        model.AlarmTypeOverspeed,
				"device_id":   deviceID,
				"speed":       speed,
				"speed_limit": conditions.SpeedLimit,
				"lat":         lat,
				"lon":         lon,
			}
			data, _ := json.Marshal(msg)
			s.natsConn.Publish("fms.alarm.OVERSPEED", data)
		}
	}
}

// offlineChecker 离线检测任务
func (s *AlarmService) offlineChecker() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.checkOfflineDevices()
	}
}

// checkOfflineDevices 检查离线设备
func (s *AlarmService) checkOfflineDevices() {
	// 获取离线规则
	var rule model.AlarmRule
	if err := s.db.Where("type = ? AND enabled = ?", model.AlarmTypeOffline, true).First(&rule).Error; err != nil {
		return
	}

	var conditions struct {
		OfflineMinutes int `json:"offline_minutes"`
	}
	json.Unmarshal(rule.Conditions, &conditions)
	if conditions.OfflineMinutes <= 0 {
		conditions.OfflineMinutes = 10
	}

	// 查询超过指定时间未上报的设备
	threshold := time.Now().Add(-time.Duration(conditions.OfflineMinutes) * time.Minute)
	
	var devices []model.Device
	s.db.Where("status = ? AND last_report_at < ?", model.DeviceStatusOnline, threshold).Find(&devices)

	for _, device := range devices {
		// 检查是否已经存在未处理的离线报警
		var count int64
		s.db.Model(&model.Alarm{}).
			Where("device_id = ? AND type = ? AND status = ?", 
				device.SimNo, model.AlarmTypeOffline, model.AlarmStatusUnread).
			Count(&count)
		
		if count > 0 {
			continue // 已存在未处理的离线报警
		}

		// 发送离线报警
		msg := map[string]interface{}{
			"type":      model.AlarmTypeOffline,
			"device_id": device.SimNo,
			"device_name": device.Name,
		}
		data, _ := json.Marshal(msg)
		s.natsConn.Publish("fms.alarm.OFFLINE", data)
	}
}

// isSilenced 检查报警是否被静默
func (s *AlarmService) isSilenced(deviceID string, alarmType model.AlarmType) bool {
	var count int64
	s.db.Model(&model.AlarmSilence{}).
		Where("device_id = ? AND alarm_type = ? AND silence_until > ?",
			deviceID, alarmType, time.Now()).
		Count(&count)
	return count > 0
}

// isRuleEnabled 检查规则是否对设备启用
func (s *AlarmService) isRuleEnabled(alarmType model.AlarmType, deviceID string) bool {
	var rule model.AlarmRule
	if err := s.db.Where("type = ? AND enabled = ?", alarmType, true).First(&rule).Error; err != nil {
		return false
	}

	if rule.AllDevices {
		return true
	}

	// 检查设备是否在指定列表中
	// 这里简化处理，实际应该查询 device_ids 数组
	return true
}

// getAlarmLevel 获取报警级别
func (s *AlarmService) getAlarmLevel(alarmType model.AlarmType) model.AlarmLevel {
	switch alarmType {
	case model.AlarmTypeSOS, model.AlarmTypeOverspeed, model.AlarmTypeIllegalMove:
		return model.AlarmLevelCritical
	case model.AlarmTypeGeofenceEnter, model.AlarmTypeGeofenceExit, 
		 model.AlarmTypeLowBattery, model.AlarmTypePowerCut:
		return model.AlarmLevelWarning
	default:
		return model.AlarmLevelInfo
	}
}

// getAlarmTitle 获取报警标题
func (s *AlarmService) getAlarmTitle(alarmType model.AlarmType) string {
	titles := map[model.AlarmType]string{
		model.AlarmTypeGeofenceEnter: "进入围栏",
		model.AlarmTypeGeofenceExit:  "离开围栏",
		model.AlarmTypeOverspeed:     "超速报警",
		model.AlarmTypeLowBattery:    "低电量报警",
		model.AlarmTypeOffline:       "设备离线",
		model.AlarmTypeSOS:           "紧急求救",
		model.AlarmTypePowerCut:      "断电报警",
		model.AlarmTypeVibration:     "震动报警",
		model.AlarmTypeIllegalMove:   "非法移动",
	}
	if title, ok := titles[alarmType]; ok {
		return title
	}
	return "未知报警"
}

// getAlarmContent 获取报警内容
func (s *AlarmService) getAlarmContent(msg struct {
	Type         model.AlarmType `json:"type"`
	DeviceID     string          `json:"device_id"`
	DeviceName   string          `json:"device_name"`
	Lat          float64         `json:"lat"`
	Lon          float64         `json:"lon"`
	Speed        int16           `json:"speed"`
	GeofenceID   int             `json:"geofence_id"`
	GeofenceName string          `json:"geofence_name"`
	Extras       json.RawMessage `json:"extras"`
}) string {
	switch msg.Type {
	case model.AlarmTypeGeofenceEnter:
		return fmt.Sprintf("车辆 %s 进入围栏 %s", msg.DeviceName, msg.GeofenceName)
	case model.AlarmTypeGeofenceExit:
		return fmt.Sprintf("车辆 %s 离开围栏 %s", msg.DeviceName, msg.GeofenceName)
	case model.AlarmTypeOverspeed:
		return fmt.Sprintf("车辆 %s 当前速度 %d km/h，超过限速", msg.DeviceName, msg.Speed)
	case model.AlarmTypeOffline:
		return fmt.Sprintf("车辆 %s 已离线", msg.DeviceName)
	case model.AlarmTypeSOS:
		return fmt.Sprintf("车辆 %s 触发紧急求救按钮", msg.DeviceName)
	default:
		return fmt.Sprintf("车辆 %s 触发 %s 报警", msg.DeviceName, s.getAlarmTitle(msg.Type))
	}
}

// sendWebhookNotification 发送 Webhook 通知
func (s *AlarmService) sendWebhookNotification(alarm model.Alarm) {
	// 获取规则配置
	var rule model.AlarmRule
	if err := s.db.Where("type = ? AND notify_webhook = ?", alarm.Type, true).First(&rule).Error; err != nil {
		return
	}

	if rule.WebhookURL == "" {
		return
	}

	// 异步发送 webhook
	go func() {
		payload, _ := json.Marshal(alarm)
		// 这里使用 http client 发送 webhook
		// 简化处理，实际应该实现重试机制
		fmt.Printf("send webhook to %s: %s\n", rule.WebhookURL, string(payload))
	}()
}

// UpdateStatus 更新报警状态
func (s *AlarmService) UpdateStatus(ctx context.Context, alarmID int, status model.AlarmStatus, note string, userID int) error {
	// 获取报警信息（用于 webhook 通知）
	var alarm model.Alarm
	if err := s.db.First(&alarm, alarmID).Error; err != nil {
		return err
	}

	updates := map[string]interface{}{
		"status":       status,
		"updated_at":   time.Now(),
	}

	if status == model.AlarmStatusResolved {
		updates["resolved_at"] = time.Now()
		updates["resolved_by"] = userID
		updates["resolve_note"] = note
	}

	if err := s.db.Model(&model.Alarm{}).Where("id = ?", alarmID).Updates(updates).Error; err != nil {
		return err
	}

	// 触发 webhook 事件
	if s.webhookService != nil && status == model.AlarmStatusResolved {
		alarm.Status = status
		alarm.ResolveNote = note
		s.webhookService.TriggerAlarmEvent(ctx, &alarm, string(model.WebhookEventAlarmResolved))
	}

	return nil
}

// BatchUpdateStatus 批量更新报警状态
func (s *AlarmService) BatchUpdateStatus(ctx context.Context, ids []int, status model.AlarmStatus, note string, userID int) error {
	updates := map[string]interface{}{
		"status":       status,
		"updated_at":   time.Now(),
	}

	if status == model.AlarmStatusResolved {
		updates["resolved_at"] = time.Now()
		updates["resolved_by"] = userID
		updates["resolve_note"] = note
	}

	return s.db.Model(&model.Alarm{}).Where("id IN ?", ids).Updates(updates).Error
}

// GetStats 获取报警统计
func (s *AlarmService) GetStats(ctx context.Context) (*model.AlarmStats, error) {
	var stats model.AlarmStats
	stats.ByType = make(map[string]int64)
	stats.ByLevel = make(map[string]int64)

	// 总数
	s.db.Model(&model.Alarm{}).Count(&stats.Total)

	// 按状态统计
	s.db.Model(&model.Alarm{}).Where("status = ?", model.AlarmStatusUnread).Count(&stats.Unread)
	s.db.Model(&model.Alarm{}).Where("status = ?", model.AlarmStatusRead).Count(&stats.Read)
	s.db.Model(&model.Alarm{}).Where("status = ?", model.AlarmStatusResolved).Count(&stats.Resolved)

	// 今日统计
	today := time.Now().Format("2006-01-02")
	s.db.Model(&model.Alarm{}).Where("DATE(created_at) = ?", today).Count(&stats.Today)
	stats.TotalToday = stats.Today

	// 按级别统计
	s.db.Model(&model.Alarm{}).Where("level = ?", model.AlarmLevelCritical).Count(&stats.Critical)
	s.db.Model(&model.Alarm{}).Where("level = ?", model.AlarmLevelWarning).Count(&stats.Warning)
	s.db.Model(&model.Alarm{}).Where("level = ?", model.AlarmLevelInfo).Count(&stats.Info)

	// 填充 ByLevel
	stats.ByLevel["critical"] = stats.Critical
	stats.ByLevel["warning"] = stats.Warning
	stats.ByLevel["info"] = stats.Info
	stats.ByLevel["resolved"] = stats.Resolved

	// 按类型统计
	var typeStats []struct {
		Type  string
		Count int64
	}
	s.db.Model(&model.Alarm{}).Select("type, COUNT(*) as count").Group("type").Scan(&typeStats)
	for _, ts := range typeStats {
		stats.ByType[ts.Type] = ts.Count
	}

	return &stats, nil
}

// GetUnreadCount 获取未读报警数量
func (s *AlarmService) GetUnreadCount(ctx context.Context) (int64, error) {
	var count int64
	err := s.db.Model(&model.Alarm{}).Where("status = ?", model.AlarmStatusUnread).Count(&count).Error
	return count, err
}

// ReplayAlarms replays alarm messages from JetStream within a time range
func (s *AlarmService) ReplayAlarms(ctx context.Context, deviceID string, start, end time.Time, batchSize int) ([]*AlarmMessage, bool, error) {
	if s.jetstream == nil || !s.jetstream.IsEnabled() {
		return nil, false, fmt.Errorf("JetStream is not enabled")
	}

	return s.jetstream.ReplayAlarms(ctx, deviceID, start, end, batchSize)
}

// GetAlarmStats returns statistics about the alarm stream
func (s *AlarmService) GetAlarmStats() (map[string]interface{}, error) {
	if s.jetstream == nil || !s.jetstream.IsEnabled() {
		return map[string]interface{}{
			"enabled": false,
		}, nil
	}

	info, err := s.jetstream.GetStreamInfo(StreamAlarms)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"enabled":     true,
		"stream":      info.Config.Name,
		"subjects":    info.Config.Subjects,
		"state":       info.State,
		"created":     info.Created,
		"max_age":     info.Config.MaxAge,
		"max_bytes":   info.Config.MaxBytes,
		"storage":     info.Config.Storage,
		"replicas":    info.Config.Replicas,
	}, nil
}

// Stop 停止报警服务
func (s *AlarmService) Stop() {
	// 清理资源
}
