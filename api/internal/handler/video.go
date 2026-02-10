package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"openfms/api/internal/model"
	"openfms/api/internal/service"
)

// VideoHandler 视频处理器
type VideoHandler struct {
	db          *gorm.DB
	videoService *service.VideoService
}

// NewVideoHandler 创建视频处理器
func NewVideoHandler(db *gorm.DB, videoService *service.VideoService) *VideoHandler {
	return &VideoHandler{
		db:          db,
		videoService: videoService,
	}
}

// RegisterRoutes 注册路由
func (h *VideoHandler) RegisterRoutes(r *gin.RouterGroup) {
	videos := r.Group("/videos")
	{
		// 实时视频
		videos.POST("/start", h.StartRealtimeVideo)
		videos.POST("/stop", h.StopVideo)
		videos.GET("/streams", h.GetActiveStreams)
		videos.GET("/streams/:id", h.GetStreamStatus)
		
		// 回放
		videos.POST("/playback/start", h.StartPlayback)
		videos.POST("/playback/control", h.ControlPlayback)
		videos.GET("/records", h.QueryRecords)
		
		// 截图
		videos.POST("/snapshot", h.TakeSnapshot)
		
		// 设备配置
		videos.GET("/devices/:device_id/config", h.GetDeviceConfig)
		videos.PUT("/devices/:device_id/config", h.UpdateDeviceConfig)
	}
}

// StartRealtimeVideo 开始实时视频
func (h *VideoHandler) StartRealtimeVideo(c *gin.Context) {
	var req model.StartVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("userID")

	resp, err := h.videoService.StartRealtimeVideo(req.DeviceID, req.Channel, userID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// StopVideo 停止视频
func (h *VideoHandler) StopVideo(c *gin.Context) {
	var req model.StopVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.videoService.StopVideo(req.StreamID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "video stopped"})
}

// GetStreamStatus 获取流状态
func (h *VideoHandler) GetStreamStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid stream id"})
		return
	}

	resp, err := h.videoService.GetStreamStatus(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetActiveStreams 获取活跃流列表
func (h *VideoHandler) GetActiveStreams(c *gin.Context) {
	deviceID := c.Query("device_id")
	
	streams, err := h.videoService.GetActiveStreams(deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, streams)
}

// StartPlayback 开始回放
func (h *VideoHandler) StartPlayback(c *gin.Context) {
	var req struct {
		DeviceID  string `json:"device_id" binding:"required"`
		Channel   int    `json:"channel" binding:"required,min=1,max=8"`
		StartTime int64  `json:"start_time" binding:"required"`
		EndTime   int64  `json:"end_time" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("userID")

	resp, err := h.videoService.StartPlayback(req.DeviceID, req.Channel, req.StartTime, req.EndTime, userID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ControlPlayback 控制回放
func (h *VideoHandler) ControlPlayback(c *gin.Context) {
	var req struct {
		StreamID int                    `json:"stream_id" binding:"required"`
		Action   string                 `json:"action" binding:"required"` // pause, resume, speed, seek
		Params   map[string]interface{} `json:"params"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.videoService.ControlPlayback(req.StreamID, req.Action, req.Params); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "control sent"})
}

// QueryRecords 查询录像
func (h *VideoHandler) QueryRecords(c *gin.Context) {
	var query model.VideoRecordQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	records, err := h.videoService.QueryRecords(query.DeviceID, query.Channel, query.StartTime, query.EndTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, records)
}

// TakeSnapshot 截图
func (h *VideoHandler) TakeSnapshot(c *gin.Context) {
	var req model.SnapshotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.videoService.TakeSnapshot(req.DeviceID, req.Channel)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetDeviceConfig 获取设备视频配置
func (h *VideoHandler) GetDeviceConfig(c *gin.Context) {
	deviceID := c.Param("device_id")
	
	config, err := h.videoService.GetDeviceConfig(deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// UpdateDeviceConfig 更新设备视频配置
func (h *VideoHandler) UpdateDeviceConfig(c *gin.Context) {
	deviceID := c.Param("device_id")
	
	var config model.VideoDeviceConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	config.DeviceID = deviceID

	if err := h.videoService.UpdateDeviceConfig(&config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "config updated"})
}
