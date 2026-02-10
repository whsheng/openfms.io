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
	trackProcessor  *service.TrackProcessor
}

// NewPositionHandler creates a new position handler
func NewPositionHandler(positionService *service.PositionService) *PositionHandler {
	return &PositionHandler{
		positionService: positionService,
		trackProcessor:  service.NewTrackProcessor(),
	}
}

// GetHistory returns position history for a device
// @Summary Get position history
// @Description Get position history for a specific device within a time range
// @Tags Positions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param device_id path string true "Device ID"
// @Param start query string true "Start time (RFC3339 format)"
// @Param end query string true "End time (RFC3339 format)"
// @Param limit query int false "Limit" default(1000)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /devices/{device_id}/positions [get]
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
// @Summary Get latest position
// @Description Get the latest position for a specific device
// @Tags Positions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param device_id path string true "Device ID"
// @Success 200 {object} model.Position
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /devices/{device_id}/positions/latest [get]
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
// @Summary Get all latest positions
// @Description Get the latest positions for all online devices
// @Tags Positions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /positions/latest [get]
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

// GetCorrectedTrack returns the corrected track for a device
// Query params:
//   - start: start time in RFC3339 format
//   - end: end time in RFC3339 format
//   - max_speed: maximum allowed speed in km/h (default: 200)
//   - max_angle: maximum allowed angle change in degrees (default: 120)
//   - min_distance: minimum distance between points in meters (default: 5)
//   - epsilon: simplification epsilon in meters (default: 10)
func (h *PositionHandler) GetCorrectedTrack(c *gin.Context) {
	deviceID := c.Param("id")

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

	// Parse correction parameters
	maxSpeed, _ := strconv.ParseFloat(c.DefaultQuery("max_speed", "200"), 64)
	maxAngle, _ := strconv.ParseFloat(c.DefaultQuery("max_angle", "120"), 64)
	minDistance, _ := strconv.ParseFloat(c.DefaultQuery("min_distance", "5"), 64)
	epsilon, _ := strconv.ParseFloat(c.DefaultQuery("epsilon", "10"), 64)

	// Configure track processor
	h.trackProcessor.MaxSpeed = maxSpeed
	h.trackProcessor.MaxAngleChange = maxAngle
	h.trackProcessor.MinDistance = minDistance
	h.trackProcessor.SimplificationEpsilon = epsilon

	// Get raw positions
	limit := 10000
	positions, err := h.positionService.GetHistory(c.Request.Context(), deviceID, startTime, endTime, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to track points
	points := service.PositionsToTrackPoints(positions)

	// Apply correction
	correctedPoints := h.trackProcessor.CorrectTrack(points)

	// Calculate statistics
	originalStats := h.trackProcessor.CalculateStats(points)
	correctedStats := h.trackProcessor.CalculateStats(correctedPoints)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"device_id":       deviceID,
			"original_count":  len(points),
			"corrected_count": len(correctedPoints),
			"removed_count":   len(points) - len(correctedPoints),
			"original_stats":  originalStats,
			"corrected_stats": correctedStats,
			"track":           correctedPoints,
		},
	})
}

// GetSimplifiedTrack returns the simplified track for a device using Douglas-Peucker algorithm
// Query params:
//   - start: start time in RFC3339 format
//   - end: end time in RFC3339 format
//   - epsilon: simplification epsilon in meters (default: 10)
//   - rate: target simplification rate (0-1), if provided will override epsilon
//   - target_count: target number of points, if provided will override epsilon and rate
//   - apply_correction: whether to apply correction before simplification (default: true)
func (h *PositionHandler) GetSimplifiedTrack(c *gin.Context) {
	deviceID := c.Param("id")

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

	// Parse simplification parameters
	epsilon, _ := strconv.ParseFloat(c.DefaultQuery("epsilon", "10"), 64)
	rate, _ := strconv.ParseFloat(c.DefaultQuery("rate", "0"), 64)
	targetCount, _ := strconv.Atoi(c.DefaultQuery("target_count", "0"))
	applyCorrection := c.DefaultQuery("apply_correction", "true") == "true"

	// Configure track processor
	h.trackProcessor.SimplificationEpsilon = epsilon

	// Get raw positions
	limit := 10000
	positions, err := h.positionService.GetHistory(c.Request.Context(), deviceID, startTime, endTime, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to track points
	points := service.PositionsToTrackPoints(positions)

	// Apply correction if requested
	if applyCorrection {
		points = h.trackProcessor.CorrectTrack(points)
	}

	originalCount := len(points)

	// Apply simplification
	var simplifiedPoints []service.TrackPoint
	if targetCount > 0 && targetCount < originalCount {
		simplifiedPoints = h.trackProcessor.SimplifyTrackWithRate(points, targetCount)
	} else if rate > 0 && rate < 1 {
		target := int(float64(originalCount) * rate)
		if target < 2 {
			target = 2
		}
		simplifiedPoints = h.trackProcessor.SimplifyTrackWithRate(points, target)
	} else {
		simplifiedPoints = h.trackProcessor.SimplifyTrack(points)
	}

	// Calculate statistics
	originalStats := h.trackProcessor.CalculateStats(points)
	simplifiedStats := h.trackProcessor.CalculateStats(simplifiedPoints)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"device_id":         deviceID,
			"original_count":    originalCount,
			"simplified_count":  len(simplifiedPoints),
			"reduction_percent": float64(originalCount-len(simplifiedPoints)) / float64(originalCount) * 100,
			"original_stats":    originalStats,
			"simplified_stats":  simplifiedStats,
			"track":             simplifiedPoints,
		},
	})
}
