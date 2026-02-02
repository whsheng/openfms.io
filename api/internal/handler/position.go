package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"openfms/api/internal/service"
)

// PositionHandler handles position-related requests
type PositionHandler struct {
	positionService *service.PositionService
}

// NewPositionHandler creates a new position handler
func NewPositionHandler(positionService *service.PositionService) *PositionHandler {
	return &PositionHandler{positionService: positionService}
}

// GetHistory returns position history for a device
func (h *PositionHandler) GetHistory(c *gin.Context) {
	deviceID := c.Param("device_id")

	// Parse time range
	startTime, err := time.Parse(time.RFC3339, c.Query("start"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start time"})
		return
	}

	endTime, err := time.Parse(time.RFC3339, c.Query("end"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end time"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "1000"))

	positions, err := h.positionService.GetHistory(c.Request.Context(), deviceID, startTime, endTime, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": positions,
	})
}

// GetLatest returns latest position for a device
func (h *PositionHandler) GetLatest(c *gin.Context) {
	deviceID := c.Param("device_id")

	position, err := h.positionService.GetLatest(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no position data"})
		return
	}

	c.JSON(http.StatusOK, position)
}

// GetAllLatest returns latest positions for all online devices
func (h *PositionHandler) GetAllLatest(c *gin.Context) {
	positions, err := h.positionService.GetAllLatest(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": positions,
	})
}
