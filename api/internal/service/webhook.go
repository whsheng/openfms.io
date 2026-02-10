package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"gorm.io/gorm"

	"openfms/api/internal/model"
)

const (
	// WebhookSignatureHeader 签名请求头
	WebhookSignatureHeader = "X-Webhook-Signature"
	// WebhookTimestampHeader 时间戳请求头
	WebhookTimestampHeader = "X-Webhook-Timestamp"
	// WebhookEventHeader 事件类型请求头
	WebhookEventHeader = "X-Webhook-Event"
	// WebhookIDHeader 事件ID请求头
	WebhookIDHeader = "X-Webhook-ID"

	// MaxFailCount 最大失败次数，超过则禁用 webhook
	MaxFailCount = 100
)

// WebhookService Webhook 服务
type WebhookService struct {
	db         *gorm.DB
	httpClient *http.Client
}

// NewWebhookService 创建 Webhook 服务
func NewWebhookService(db *gorm.DB) *WebhookService {
	return &WebhookService{
		db: db,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Create 创建 Webhook
func (s *WebhookService) Create(ctx context.Context, req *model.CreateWebhookRequest, userID int) (*model.Webhook, error) {
	webhook := &model.Webhook{
		Name:          req.Name,
		Description:   req.Description,
		URL:           req.URL,
		Secret:        req.Secret,
		Events:        req.Events,
		Status:        model.WebhookStatusActive,
		RetryCount:    req.RetryCount,
		RetryInterval: req.RetryInterval,
		Timeout:       req.Timeout,
		CreatedBy:     &userID,
	}

	// 设置默认值
	if webhook.RetryCount == 0 {
		webhook.RetryCount = 3
	}
	if webhook.RetryInterval == 0 {
		webhook.RetryInterval = 5
	}
	if webhook.Timeout == 0 {
		webhook.Timeout = 30
	}

	if err := s.db.Create(webhook).Error; err != nil {
		return nil, fmt.Errorf("create webhook failed: %w", err)
	}

	return webhook, nil
}

// Get 获取 Webhook
func (s *WebhookService) Get(ctx context.Context, id int) (*model.Webhook, error) {
	var webhook model.Webhook
	if err := s.db.Where("deleted_at IS NULL").First(&webhook, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("webhook not found")
		}
		return nil, err
	}
	return &webhook, nil
}

// List 获取 Webhook 列表
func (s *WebhookService) List(ctx context.Context, query *model.WebhookListQuery) (*model.WebhookListResponse, error) {
	db := s.db.Model(&model.Webhook{}).Where("deleted_at IS NULL")

	// 应用筛选条件
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.Event != "" {
		db = db.Where("? = ANY(events) OR 'all' = ANY(events)", query.Event)
	}

	// 获取总数
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	// 获取列表
	var webhooks []model.Webhook
	offset := (query.Page - 1) * query.PageSize
	if err := db.Order("created_at DESC").Offset(offset).Limit(query.PageSize).Find(&webhooks).Error; err != nil {
		return nil, err
	}

	return &model.WebhookListResponse{
		List:     webhooks,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}

// Update 更新 Webhook
func (s *WebhookService) Update(ctx context.Context, id int, req *model.UpdateWebhookRequest) error {
	updates := make(map[string]interface{})

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.URL != "" {
		updates["url"] = req.URL
	}
	if req.Secret != "" {
		updates["secret"] = req.Secret
	}
	if len(req.Events) > 0 {
		updates["events"] = req.Events
	}
	if req.RetryCount > 0 {
		updates["retry_count"] = req.RetryCount
	}
	if req.RetryInterval > 0 {
		updates["retry_interval"] = req.RetryInterval
	}
	if req.Timeout > 0 {
		updates["timeout"] = req.Timeout
	}
	updates["updated_at"] = time.Now()

	if err := s.db.Model(&model.Webhook{}).Where("id = ? AND deleted_at IS NULL", id).Updates(updates).Error; err != nil {
		return err
	}
	return nil
}

// Delete 删除 Webhook（软删除）
func (s *WebhookService) Delete(ctx context.Context, id int) error {
	now := time.Now()
	return s.db.Model(&model.Webhook{}).Where("id = ?", id).Update("deleted_at", &now).Error
}

// ToggleStatus 切换 Webhook 状态
func (s *WebhookService) ToggleStatus(ctx context.Context, id int) (*model.WebhookStatus, error) {
	var webhook model.Webhook
	if err := s.db.First(&webhook, id).Error; err != nil {
		return nil, err
	}

	var newStatus model.WebhookStatus
	if webhook.Status == model.WebhookStatusActive {
		newStatus = model.WebhookStatusInactive
	} else {
		newStatus = model.WebhookStatusActive
	}

	if err := s.db.Model(&webhook).Updates(map[string]interface{}{
		"status":     newStatus,
		"updated_at": time.Now(),
	}).Error; err != nil {
		return nil, err
	}

	return &newStatus, nil
}

// TriggerEvent 触发事件，发送给所有订阅的 Webhook
func (s *WebhookService) TriggerEvent(ctx context.Context, eventType string, data interface{}) error {
	// 查找订阅了该事件的 webhook
	var webhooks []model.Webhook
	err := s.db.Where("status = ? AND deleted_at IS NULL", model.WebhookStatusActive).
		Where("? = ANY(events) OR 'all' = ANY(events)", eventType).
		Find(&webhooks).Error
	if err != nil {
		return fmt.Errorf("query webhooks failed: %w", err)
	}

	if len(webhooks) == 0 {
		return nil
	}

	// 构建事件数据
	eventID := generateEventID()
	timestamp := time.Now().UnixMilli()

	payload := model.WebhookPayload{
		EventID:   eventID,
		EventType: eventType,
		Timestamp: timestamp,
		Data:      data,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload failed: %w", err)
	}

	// 异步发送给所有 webhook
	for _, webhook := range webhooks {
		go s.sendWebhookWithRetry(&webhook, eventType, eventID, payloadBytes)
	}

	return nil
}

// sendWebhookWithRetry 发送 Webhook 并支持重试
func (s *WebhookService) sendWebhookWithRetry(webhook *model.Webhook, eventType, eventID string, payload []byte) {
	ctx := context.Background()

	// 创建投递记录
	delivery := &model.WebhookDelivery{
		WebhookID: webhook.ID,
		EventType: eventType,
		Payload:   payload,
	}
	if err := s.db.Create(delivery).Error; err != nil {
		fmt.Printf("[Webhook] Failed to create delivery record: %v\n", err)
	}

	var lastErr error
	var lastStatusCode int
	var lastResponseBody string
	var totalDuration int

	// 重试循环
	for attempt := 1; attempt <= webhook.RetryCount+1; attempt++ {
		if attempt > 1 {
			// 非第一次尝试，等待重试间隔
			time.Sleep(time.Duration(webhook.RetryInterval) * time.Second)
		}

		start := time.Now()
		success, statusCode, responseBody, err := s.sendWebhook(webhook, eventType, eventID, payload)
		duration := int(time.Since(start).Milliseconds())
		totalDuration += duration

		lastStatusCode = statusCode
		lastResponseBody = responseBody

		if success {
			// 更新投递记录
			now := time.Now()
			s.db.Model(delivery).Updates(map[string]interface{}{
				"response_status": statusCode,
				"response_body":   responseBody,
				"attempt_count":   attempt,
				"duration_ms":     totalDuration,
				"completed_at":    &now,
			})

			// 更新 webhook 统计
			s.db.Model(webhook).Updates(map[string]interface{}{
				"success_count":     gorm.Expr("success_count + 1"),
				"last_triggered_at": time.Now(),
				"last_error":        "",
			})
			return
		}

		lastErr = err
	}

	// 所有重试都失败了
	now := time.Now()
	errorMsg := ""
	if lastErr != nil {
		errorMsg = lastErr.Error()
	}

	s.db.Model(delivery).Updates(map[string]interface{}{
		"response_status": lastStatusCode,
		"response_body":   lastResponseBody,
		"attempt_count":   webhook.RetryCount + 1,
		"duration_ms":     totalDuration,
		"error_message":   errorMsg,
		"completed_at":    &now,
	})

	// 更新 webhook 统计和失败次数
	updates := map[string]interface{}{
		"fail_count":        gorm.Expr("fail_count + 1"),
		"last_triggered_at": time.Now(),
		"last_error":        errorMsg,
	}

	// 如果失败次数过多，自动禁用
	if webhook.FailCount+1 >= MaxFailCount {
		updates["status"] = model.WebhookStatusFailed
	}

	s.db.Model(webhook).Updates(updates)

	fmt.Printf("[Webhook] Failed to send webhook %d after %d attempts: %v\n",
		webhook.ID, webhook.RetryCount+1, lastErr)
}

// sendWebhook 发送单个 Webhook 请求
func (s *WebhookService) sendWebhook(webhook *model.Webhook, eventType, eventID string, payload []byte) (bool, int, string, error) {
	// 创建请求
	req, err := http.NewRequest(http.MethodPost, webhook.URL, bytes.NewBuffer(payload))
	if err != nil {
		return false, 0, "", fmt.Errorf("create request failed: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "OpenFMS-Webhook/1.0")
	req.Header.Set(WebhookEventHeader, eventType)
	req.Header.Set(WebhookIDHeader, eventID)

	// 计算签名
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	req.Header.Set(WebhookTimestampHeader, timestamp)

	if webhook.Secret != "" {
		signature := s.GenerateSignature(payload, timestamp, webhook.Secret)
		req.Header.Set(WebhookSignatureHeader, signature)
	}

	// 设置超时
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(webhook.Timeout)*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	// 发送请求
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false, 0, "", fmt.Errorf("send request failed: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, _ := io.ReadAll(resp.Body)

	// 2xx 状态码视为成功
	success := resp.StatusCode >= 200 && resp.StatusCode < 300

	return success, resp.StatusCode, string(body), nil
}

// GenerateSignature 生成 HMAC-SHA256 签名
func (s *WebhookService) GenerateSignature(payload []byte, timestamp, secret string) string {
	// 签名格式: timestamp + "." + base64(hmac-sha256(timestamp + "." + payload))
	message := timestamp + "." + string(payload)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifySignature 验证签名
func (s *WebhookService) VerifySignature(payload []byte, timestamp, signature, secret string) bool {
	expected := s.GenerateSignature(payload, timestamp, secret)
	return hmac.Equal([]byte(signature), []byte(expected))
}

// TestWebhook 测试 Webhook
func (s *WebhookService) TestWebhook(ctx context.Context, webhookID int, eventType string, payload json.RawMessage) (*model.TestWebhookResponse, error) {
	webhook, err := s.Get(ctx, webhookID)
	if err != nil {
		return nil, err
	}

	// 构建测试事件
	eventID := "test-" + generateEventID()
	timestamp := time.Now().UnixMilli()

	testPayload := model.WebhookPayload{
		EventID:   eventID,
		EventType: eventType,
		Timestamp: timestamp,
		Data:      json.RawMessage(payload),
	}

	payloadBytes, err := json.Marshal(testPayload)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	success, statusCode, responseBody, err := s.sendWebhook(webhook, eventType, eventID, payloadBytes)
	duration := int(time.Since(start).Milliseconds())

	resp := &model.TestWebhookResponse{
		Success:    success,
		StatusCode: statusCode,
		DurationMs: duration,
	}

	if success {
		resp.ResponseBody = truncateString(responseBody, 1000) // 限制响应体长度
	} else if err != nil {
		resp.ErrorMessage = err.Error()
	} else {
		resp.ErrorMessage = fmt.Sprintf("HTTP %d: %s", statusCode, truncateString(responseBody, 500))
	}

	return resp, nil
}

// GetDeliveries 获取投递日志
func (s *WebhookService) GetDeliveries(ctx context.Context, query *model.WebhookDeliveryQuery) (*model.WebhookDeliveryResponse, error) {
	db := s.db.Model(&model.WebhookDelivery{})

	// 应用筛选条件
	if query.WebhookID > 0 {
		db = db.Where("webhook_id = ?", query.WebhookID)
	}
	if query.EventType != "" {
		db = db.Where("event_type = ?", query.EventType)
	}
	if query.Status == "success" {
		db = db.Where("response_status >= 200 AND response_status < 300")
	} else if query.Status == "failed" {
		db = db.Where("(response_status < 200 OR response_status >= 300) OR error_message != ''")
	}

	// 获取总数
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	// 获取列表
	var deliveries []model.WebhookDelivery
	offset := (query.Page - 1) * query.PageSize
	if err := db.Order("created_at DESC").Offset(offset).Limit(query.PageSize).Find(&deliveries).Error; err != nil {
		return nil, err
	}

	return &model.WebhookDeliveryResponse{
		List:     deliveries,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}

// GetStats 获取 Webhook 统计
func (s *WebhookService) GetStats(ctx context.Context) (*model.WebhookStats, error) {
	var stats model.WebhookStats

	// Webhook 统计
	s.db.Model(&model.Webhook{}).Where("deleted_at IS NULL").Count(&stats.TotalWebhooks)
	s.db.Model(&model.Webhook{}).Where("status = ? AND deleted_at IS NULL", model.WebhookStatusActive).Count(&stats.ActiveWebhooks)
	s.db.Model(&model.Webhook{}).Where("status = ? AND deleted_at IS NULL", model.WebhookStatusInactive).Count(&stats.InactiveWebhooks)
	s.db.Model(&model.Webhook{}).Where("status = ? AND deleted_at IS NULL", model.WebhookStatusFailed).Count(&stats.FailedWebhooks)

	// 投递统计
	s.db.Model(&model.WebhookDelivery{}).Count(&stats.TotalDeliveries)
	s.db.Model(&model.WebhookDelivery{}).Where("response_status >= 200 AND response_status < 300").Count(&stats.SuccessDeliveries)
	s.db.Model(&model.WebhookDelivery{}).Where("response_status < 200 OR response_status >= 300 OR error_message != ''").Count(&stats.FailedDeliveries)

	// 今日投递
	today := time.Now().Format("2006-01-02")
	s.db.Model(&model.WebhookDelivery{}).Where("DATE(created_at) = ?", today).Count(&stats.TodayDeliveries)

	return &stats, nil
}

// generateEventID 生成事件ID
func generateEventID() string {
	return fmt.Sprintf("evt_%d_%d", time.Now().UnixMilli(), time.Now().Nanosecond())
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// TriggerAlarmEvent 触发报警事件
func (s *WebhookService) TriggerAlarmEvent(ctx context.Context, alarm *model.Alarm, eventType string) error {
	var data model.AlarmEventData

	data.AlarmID = alarm.ID
	data.Type = alarm.Type
	data.Level = alarm.Level
	data.DeviceID = alarm.DeviceID
	data.DeviceName = alarm.DeviceName
	data.Title = alarm.Title
	data.Content = alarm.Content
	data.Lat = alarm.Lat
	data.Lon = alarm.Lon
	data.Speed = alarm.Speed
	data.GeofenceID = alarm.GeofenceID
	data.GeofenceName = alarm.GeofenceName
	data.Extras = alarm.Extras
	data.CreatedAt = alarm.CreatedAt

	return s.TriggerEvent(ctx, eventType, data)
}

// TriggerGeofenceEvent 触发围栏事件
func (s *WebhookService) TriggerGeofenceEvent(ctx context.Context, alert *model.GeofenceAlert) error {
	var eventType string
	if alert.EventType == "enter" {
		eventType = string(model.WebhookEventGeofenceEnter)
	} else {
		eventType = string(model.WebhookEventGeofenceExit)
	}

	data := model.GeofenceEventData{
		GeofenceID:   alert.GeofenceID,
		GeofenceName: alert.GeofenceName,
		DeviceID:     fmt.Sprintf("%d", alert.DeviceID),
		DeviceName:   alert.DeviceName,
		EventType:    alert.EventType,
		Lat:          alert.Location.Lat,
		Lon:          alert.Location.Lon,
		Speed:        alert.Speed,
		Timestamp:    alert.Timestamp,
	}

	return s.TriggerEvent(ctx, eventType, data)
}

// TriggerDeviceEvent 触发设备事件
func (s *WebhookService) TriggerDeviceEvent(ctx context.Context, deviceID, deviceName, status, reason string) error {
	var eventType string
	if status == "online" {
		eventType = string(model.WebhookEventDeviceOnline)
	} else {
		eventType = string(model.WebhookEventDeviceOffline)
	}

	data := model.DeviceEventData{
		DeviceID:   deviceID,
		DeviceName: deviceName,
		Status:     status,
		Timestamp:  time.Now().UnixMilli(),
		Reason:     reason,
	}

	return s.TriggerEvent(ctx, eventType, data)
}
