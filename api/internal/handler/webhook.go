package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"openfms/api/internal/model"
	"openfms/api/internal/service"
)

// WebhookHandler Webhook 处理器
type WebhookHandler struct {
	db             *gorm.DB
	webhookService *service.WebhookService
}

// NewWebhookHandler 创建 Webhook 处理器
func NewWebhookHandler(db *gorm.DB, webhookService *service.WebhookService) *WebhookHandler {
	return &WebhookHandler{
		db:             db,
		webhookService: webhookService,
	}
}

// RegisterRoutes 注册路由
func (h *WebhookHandler) RegisterRoutes(r *gin.RouterGroup) {
	webhooks := r.Group("/webhooks")
	{
		webhooks.GET("", h.ListWebhooks)
		webhooks.POST("", h.CreateWebhook)
		webhooks.GET("/stats", h.GetStats)
		webhooks.GET("/events", h.GetEventTypes)
		webhooks.GET("/:id", h.GetWebhook)
		webhooks.PUT("/:id", h.UpdateWebhook)
		webhooks.DELETE("/:id", h.DeleteWebhook)
		webhooks.POST("/:id/toggle", h.ToggleWebhook)
		webhooks.POST("/:id/test", h.TestWebhook)
		webhooks.GET("/:id/deliveries", h.GetDeliveries)
	}
}

// ListWebhooks 获取 Webhook 列表
func (h *WebhookHandler) ListWebhooks(c *gin.Context) {
	var query model.WebhookListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 设置默认值
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 || query.PageSize > 100 {
		query.PageSize = 20
	}

	resp, err := h.webhookService.List(c.Request.Context(), &query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetWebhook 获取 Webhook 详情
func (h *WebhookHandler) GetWebhook(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook id"})
		return
	}

	webhook, err := h.webhookService.Get(c.Request.Context(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "webhook not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, webhook)
}

// CreateWebhook 创建 Webhook
func (h *WebhookHandler) CreateWebhook(c *gin.Context) {
	var req model.CreateWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证事件类型
	if len(req.Events) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "events cannot be empty"})
		return
	}

	userID, _ := c.Get("userID")
	userIDInt, _ := userID.(int)

	webhook, err := h.webhookService.Create(c.Request.Context(), &req, userIDInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, webhook)
}

// UpdateWebhook 更新 Webhook
func (h *WebhookHandler) UpdateWebhook(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook id"})
		return
	}

	var req model.UpdateWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.webhookService.Update(c.Request.Context(), id, &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "webhook updated"})
}

// DeleteWebhook 删除 Webhook
func (h *WebhookHandler) DeleteWebhook(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook id"})
		return
	}

	if err := h.webhookService.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "webhook deleted"})
}

// ToggleWebhook 切换 Webhook 状态
func (h *WebhookHandler) ToggleWebhook(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook id"})
		return
	}

	newStatus, err := h.webhookService.ToggleStatus(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "webhook status toggled",
		"status":  newStatus,
	})
}

// TestWebhook 测试 Webhook
func (h *WebhookHandler) TestWebhook(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook id"})
		return
	}

	var req model.TestWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.webhookService.TestWebhook(c.Request.Context(), id, req.EventType, req.Payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetDeliveries 获取投递日志
func (h *WebhookHandler) GetDeliveries(c *gin.Context) {
	var query model.WebhookDeliveryQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 从 URL 参数获取 webhook_id
	if webhookID, err := strconv.Atoi(c.Param("id")); err == nil {
		query.WebhookID = webhookID
	}

	// 设置默认值
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 || query.PageSize > 100 {
		query.PageSize = 20
	}

	resp, err := h.webhookService.GetDeliveries(c.Request.Context(), &query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetStats 获取 Webhook 统计
func (h *WebhookHandler) GetStats(c *gin.Context) {
	stats, err := h.webhookService.GetStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetEventTypes 获取支持的事件类型列表
func (h *WebhookHandler) GetEventTypes(c *gin.Context) {
	eventTypes := []gin.H{
		{
			"value":       model.WebhookEventAlarmCreated,
			"label":       "报警创建",
			"description": "当新的报警产生时触发",
			"category":    "alarm",
		},
		{
			"value":       model.WebhookEventAlarmResolved,
			"label":       "报警已处理",
			"description": "当报警被标记为已处理时触发",
			"category":    "alarm",
		},
		{
			"value":       model.WebhookEventGeofenceEnter,
			"label":       "进入围栏",
			"description": "当车辆进入电子围栏时触发",
			"category":    "geofence",
		},
		{
			"value":       model.WebhookEventGeofenceExit,
			"label":       "离开围栏",
			"description": "当车辆离开电子围栏时触发",
			"category":    "geofence",
		},
		{
			"value":       model.WebhookEventDeviceOnline,
			"label":       "设备上线",
			"description": "当设备上线时触发",
			"category":    "device",
		},
		{
			"value":       model.WebhookEventDeviceOffline,
			"label":       "设备离线",
			"description": "当设备离线时触发",
			"category":    "device",
		},
		{
			"value":       model.WebhookEventDevicePosition,
			"label":       "位置更新",
			"description": "当设备上报位置时触发（注意：频率较高）",
			"category":    "device",
		},
		{
			"value":       model.WebhookEventCommandResult,
			"label":       "指令执行结果",
			"description": "当设备指令执行完成时触发",
			"category":    "device",
		},
		{
			"value":       model.WebhookEventVehicleCreated,
			"label":       "车辆创建",
			"description": "当新车辆被创建时触发",
			"category":    "vehicle",
		},
		{
			"value":       model.WebhookEventVehicleUpdated,
			"label":       "车辆更新",
			"description": "当车辆信息被更新时触发",
			"category":    "vehicle",
		},
		{
			"value":       model.WebhookEventAll,
			"label":       "所有事件",
			"description": "订阅所有类型的事件",
			"category":    "all",
		},
	}

	c.JSON(http.StatusOK, eventTypes)
}

// VerifySignature 验证 Webhook 签名（用于接收方验证）
func (h *WebhookHandler) VerifySignature(c *gin.Context) {
	var req struct {
		Payload   json.RawMessage `json:"payload" binding:"required"`
		Timestamp string          `json:"timestamp" binding:"required"`
		Signature string          `json:"signature" binding:"required"`
		Secret    string          `json:"secret" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	isValid := h.webhookService.VerifySignature(req.Payload, req.Timestamp, req.Signature, req.Secret)

	c.JSON(http.StatusOK, gin.H{
		"valid": isValid,
	})
}

// WebhookEventTrigger Webhook 事件触发器接口
type WebhookEventTrigger interface {
	TriggerAlarmEvent(ctx interface{}, alarm *model.Alarm, eventType string) error
	TriggerGeofenceEvent(ctx interface{}, alert *model.GeofenceAlert) error
	TriggerDeviceEvent(ctx interface{}, deviceID, deviceName, status, reason string) error
}

// TriggerAlarmEvent 触发报警事件（供其他服务调用）
func TriggerAlarmEvent(webhookService *service.WebhookService, alarm *model.Alarm, eventType string) {
	if webhookService == nil {
		return
	}

	ctx := context.Background()
	
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

	webhookService.TriggerEvent(ctx, eventType, data)
}

// TriggerGeofenceEvent 触发围栏事件（供其他服务调用）
func TriggerGeofenceEvent(webhookService *service.WebhookService, alert *model.GeofenceAlert) {
	if webhookService == nil {
		return
	}

	ctx := context.Background()

	var eventType string
	if alert.EventType == "enter" {
		eventType = string(model.WebhookEventGeofenceEnter)
	} else {
		eventType = string(model.WebhookEventGeofenceExit)
	}

	data := model.GeofenceEventData{
		GeofenceID:   alert.GeofenceID,
		GeofenceName: alert.GeofenceName,
		DeviceID:     strconv.FormatUint(uint64(alert.DeviceID), 10),
		DeviceName:   alert.DeviceName,
		EventType:    alert.EventType,
		Lat:          alert.Location.Lat,
		Lon:          alert.Location.Lon,
		Speed:        alert.Speed,
		Timestamp:    alert.Timestamp,
	}

	webhookService.TriggerEvent(ctx, eventType, data)
}

// TriggerDeviceEvent 触发设备事件（供其他服务调用）
func TriggerDeviceEvent(webhookService *service.WebhookService, deviceID, deviceName, status, reason string) {
	if webhookService == nil {
		return
	}

	ctx := context.Background()

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

	webhookService.TriggerEvent(ctx, eventType, data)
}
