package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"openfms/api/internal/model"
	"openfms/api/internal/service"
)

// AlarmHandler 报警处理器
type AlarmHandler struct {
	db          *gorm.DB
	alarmService *service.AlarmService
}

// NewAlarmHandler 创建报警处理器
func NewAlarmHandler(db *gorm.DB, alarmService *service.AlarmService) *AlarmHandler {
	return &AlarmHandler{
		db:          db,
		alarmService: alarmService,
	}
}

// RegisterRoutes 注册路由
func (h *AlarmHandler) RegisterRoutes(r *gin.RouterGroup) {
	alarms := r.Group("/alarms")
	{
		alarms.GET("", h.ListAlarms)
		alarms.GET("/stats", h.GetStats)
		alarms.GET("/types", h.GetAlarmTypes)
		alarms.GET("/:id", h.GetAlarm)
		alarms.POST("/:id/read", h.MarkAsRead)
		alarms.POST("/:id/resolve", h.ResolveAlarm)
		alarms.POST("/batch-resolve", h.BatchResolve)
		alarms.POST("/batch-read", h.BatchRead)
		
		// 报警规则
		alarms.GET("/rules", h.ListRules)
		alarms.GET("/rules/:id", h.GetRule)
		alarms.PUT("/rules/:id", h.UpdateRule)
		alarms.POST("/rules/:id/toggle", h.ToggleRule)
	}
}

// ListAlarms 获取报警列表
// @Summary List alarms
// @Description Get a paginated list of alarms with optional filters
// @Tags Alarms
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param device_id query string false "Filter by device ID"
// @Param type query string false "Filter by alarm type"
// @Param level query string false "Filter by alarm level"
// @Param status query string false "Filter by status"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Success 200 {object} model.AlarmListResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /alarms [get]
func (h *AlarmHandler) ListAlarms(c *gin.Context) {
	var query model.AlarmListQuery
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

	db := h.db.Model(&model.Alarm{})

	// 应用筛选条件
	if query.DeviceID != "" {
		db = db.Where("device_id = ?", query.DeviceID)
	}
	if query.Type != "" {
		db = db.Where("type = ?", query.Type)
	}
	if query.Level != "" {
		db = db.Where("level = ?", query.Level)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.StartTime != nil {
		db = db.Where("created_at >= ?", query.StartTime)
	}
	if query.EndTime != nil {
		db = db.Where("created_at <= ?", query.EndTime)
	}

	// 获取总数
	var total int64
	if err := db.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取列表
	var alarms []model.Alarm
	offset := (query.Page - 1) * query.PageSize
	if err := db.Order("created_at DESC").Offset(offset).Limit(query.PageSize).Find(&alarms).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.AlarmListResponse{
		List:     alarms,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	})
}

// GetAlarm 获取报警详情
// @Summary Get alarm
// @Description Get a single alarm by ID
// @Tags Alarms
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Alarm ID"
// @Success 200 {object} model.Alarm
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /alarms/{id} [get]
func (h *AlarmHandler) GetAlarm(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alarm id"})
		return
	}

	var alarm model.Alarm
	if err := h.db.First(&alarm, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "alarm not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, alarm)
}

// GetStats 获取报警统计
// @Summary Get alarm statistics
// @Description Get alarm statistics
// @Tags Alarms
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /alarms/stats [get]
func (h *AlarmHandler) GetStats(c *gin.Context) {
	stats, err := h.alarmService.GetStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetAlarmTypes 获取报警类型列表
// @Summary Get alarm types
// @Description Get list of available alarm types
// @Tags Alarms
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Router /alarms/types [get]
func (h *AlarmHandler) GetAlarmTypes(c *gin.Context) {
	types := []gin.H{
		{"value": model.AlarmTypeGeofenceEnter, "label": "进入围栏", "level": model.AlarmLevelWarning},
		{"value": model.AlarmTypeGeofenceExit, "label": "离开围栏", "level": model.AlarmLevelWarning},
		{"value": model.AlarmTypeOverspeed, "label": "超速", "level": model.AlarmLevelCritical},
		{"value": model.AlarmTypeLowBattery, "label": "低电量", "level": model.AlarmLevelWarning},
		{"value": model.AlarmTypeOffline, "label": "设备离线", "level": model.AlarmLevelInfo},
		{"value": model.AlarmTypeSOS, "label": "紧急求救", "level": model.AlarmLevelCritical},
		{"value": model.AlarmTypePowerCut, "label": "断电报警", "level": model.AlarmLevelWarning},
		{"value": model.AlarmTypeVibration, "label": "震动报警", "level": model.AlarmLevelInfo},
		{"value": model.AlarmTypeIllegalMove, "label": "非法移动", "level": model.AlarmLevelCritical},
	}
	c.JSON(http.StatusOK, types)
}

// GetUnreadCount 获取未读报警数量
// @Summary Get unread alarm count
// @Description Get count of unread alarms
// @Tags Alarms
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]int64
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /alarms/unread-count [get]
func (h *AlarmHandler) GetUnreadCount(c *gin.Context) {
	count, err := h.alarmService.GetUnreadCount(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"unread_count": count})
}

// MarkAsRead 标记报警为已读
// @Summary Mark alarm as read
// @Description Mark a specific alarm as read
// @Tags Alarms
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Alarm ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /alarms/{id}/read [post]
func (h *AlarmHandler) MarkAsRead(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alarm id"})
		return
	}

	userID, _ := c.Get("userID")

	if err := h.alarmService.UpdateStatus(c.Request.Context(), id, model.AlarmStatusRead, "", userID.(int)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "marked as read"})
}

// ResolveAlarm 处理报警
// @Summary Resolve alarm
// @Description Resolve a specific alarm
// @Tags Alarms
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Alarm ID"
// @Param request body object true "Resolve note"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /alarms/{id}/resolve [post]
func (h *AlarmHandler) ResolveAlarm(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alarm id"})
		return
	}

	var req struct {
		ResolveNote string `json:"resolve_note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("userID")

	if err := h.alarmService.UpdateStatus(c.Request.Context(), id, model.AlarmStatusResolved, req.ResolveNote, userID.(int)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "resolved"})
}

// BatchResolve 批量处理报警
// @Summary Batch resolve alarms
// @Description Resolve multiple alarms at once
// @Tags Alarms
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body model.BatchResolveRequest true "Alarm IDs and resolve note"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /alarms/batch-resolve [post]
func (h *AlarmHandler) BatchResolve(c *gin.Context) {
	var req model.BatchResolveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("userID")

	if err := h.alarmService.BatchUpdateStatus(c.Request.Context(), req.IDs, model.AlarmStatusResolved, req.ResolveNote, userID.(int)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "batch resolved",
		"count":   len(req.IDs),
	})
}

// BatchRead 批量标记已读
// @Summary Batch mark alarms as read
// @Description Mark multiple alarms as read
// @Tags Alarms
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body object true "Alarm IDs"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /alarms/batch-read [post]
func (h *AlarmHandler) BatchRead(c *gin.Context) {
	var req struct {
		IDs []int `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("userID")

	if err := h.alarmService.BatchUpdateStatus(c.Request.Context(), req.IDs, model.AlarmStatusRead, "", userID.(int)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "batch marked as read",
		"count":   len(req.IDs),
	})
}

// ListRules 获取报警规则列表
// @Summary List alarm rules
// @Description Get list of alarm rules
// @Tags Alarms
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} model.AlarmRule
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /alarms/rules [get]
func (h *AlarmHandler) ListRules(c *gin.Context) {
	var rules []model.AlarmRule
	if err := h.db.Find(&rules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rules)
}

// GetRule 获取规则详情
// @Summary Get alarm rule
// @Description Get a single alarm rule by ID
// @Tags Alarms
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Rule ID"
// @Success 200 {object} model.AlarmRule
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /alarms/rules/{id} [get]
func (h *AlarmHandler) GetRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule id"})
		return
	}

	var rule model.AlarmRule
	if err := h.db.First(&rule, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// UpdateRule 更新报警规则
// @Summary Update alarm rule
// @Description Update an existing alarm rule
// @Tags Alarms
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Rule ID"
// @Param rule body object true "Rule data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /alarms/rules/{id} [put]
func (h *AlarmHandler) UpdateRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule id"})
		return
	}

	var req struct {
		Name          string          `json:"name"`
		Description   string          `json:"description"`
		Conditions    map[string]interface{} `json:"conditions"`
		AllDevices    bool            `json:"all_devices"`
		DeviceIDs     []int           `json:"device_ids"`
		NotifyWebhook bool            `json:"notify_webhook"`
		WebhookURL    string          `json:"webhook_url"`
		NotifyWS      bool            `json:"notify_ws"`
		NotifySound   bool            `json:"notify_sound"`
		Enabled       bool            `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{
		"name":           req.Name,
		"description":    req.Description,
		"conditions":     req.Conditions,
		"all_devices":    req.AllDevices,
		"device_ids":     req.DeviceIDs,
		"notify_webhook": req.NotifyWebhook,
		"webhook_url":    req.WebhookURL,
		"notify_ws":      req.NotifyWS,
		"notify_sound":   req.NotifySound,
		"enabled":        req.Enabled,
		"updated_at":     time.Now(),
	}

	if err := h.db.Model(&model.AlarmRule{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "rule updated"})
}

// ToggleRule 切换规则启用状态
// @Summary Toggle alarm rule
// @Description Toggle the enabled status of an alarm rule
// @Tags Alarms
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Rule ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /alarms/rules/{id}/toggle [post]
func (h *AlarmHandler) ToggleRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule id"})
		return
	}

	var rule model.AlarmRule
	if err := h.db.First(&rule, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.Model(&rule).Update("enabled", !rule.Enabled).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "rule toggled",
		"enabled": !rule.Enabled,
	})
}
