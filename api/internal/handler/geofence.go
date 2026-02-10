package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"openfms/api/internal/model"
	"openfms/api/internal/service"
)

// GeofenceHandler handles geofence-related requests
type GeofenceHandler struct {
	geofenceService *service.GeofenceService
}

// NewGeofenceHandler creates a new geofence handler
func NewGeofenceHandler(geofenceService *service.GeofenceService) *GeofenceHandler {
	return &GeofenceHandler{geofenceService: geofenceService}
}

// Create creates a new geofence
// @Summary Create geofence
// @Description Create a new geofence
// @Tags Geofences
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param geofence body model.Geofence true "Geofence data"
// @Success 201 {object} model.Geofence
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /geofences [post]
func (h *GeofenceHandler) Create(c *gin.Context) {
	var geofence model.Geofence
	if err := c.ShouldBindJSON(&geofence); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context (set by auth middleware)
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(uint); ok {
			geofence.UserID = &uid
		}
	}

	if err := h.geofenceService.Create(c.Request.Context(), &geofence); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, geofence)
}

// List returns list of geofences
// @Summary List geofences
// @Description Get a paginated list of geofences
// @Tags Geofences
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /geofences [get]
func (h *GeofenceHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	geofences, total, err := h.geofenceService.List(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  geofences,
		"total": total,
		"page":  page,
	})
}

// Get returns a single geofence
// @Summary Get geofence
// @Description Get a single geofence by ID
// @Tags Geofences
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Geofence ID"
// @Success 200 {object} model.Geofence
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /geofences/{id} [get]
func (h *GeofenceHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	geofence, err := h.geofenceService.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "geofence not found"})
		return
	}

	c.JSON(http.StatusOK, geofence)
}

// Update updates a geofence
// @Summary Update geofence
// @Description Update an existing geofence
// @Tags Geofences
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Geofence ID"
// @Param geofence body model.Geofence true "Geofence data"
// @Success 200 {object} model.Geofence
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /geofences/{id} [put]
func (h *GeofenceHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var geofence model.Geofence
	if err := c.ShouldBindJSON(&geofence); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	geofence.ID = uint(id)
	if err := h.geofenceService.Update(c.Request.Context(), &geofence); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, geofence)
}

// Delete deletes a geofence
// @Summary Delete geofence
// @Description Delete a geofence by ID
// @Tags Geofences
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Geofence ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /geofences/{id} [delete]
func (h *GeofenceHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.geofenceService.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// BindDevices binds devices to a geofence
// @Summary Bind devices to geofence
// @Description Bind devices to a geofence
// @Tags Geofences
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Geofence ID"
// @Param request body object true "Device IDs"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /geofences/{id}/bind [post]
func (h *GeofenceHandler) BindDevices(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		DeviceIDs []uint `json:"device_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.geofenceService.BindDevices(c.Request.Context(), uint(id), req.DeviceIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "devices bound successfully"})
}

// UnbindDevices unbinds devices from a geofence
// @Summary Unbind devices from geofence
// @Description Unbind devices from a geofence
// @Tags Geofences
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Geofence ID"
// @Param request body object true "Device IDs"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /geofences/{id}/unbind [post]
func (h *GeofenceHandler) UnbindDevices(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		DeviceIDs []uint `json:"device_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.geofenceService.UnbindDevices(c.Request.Context(), uint(id), req.DeviceIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "devices unbound successfully"})
}

// GetDevices returns all devices bound to a geofence
// @Summary Get devices bound to geofence
// @Description Get all devices bound to a geofence
// @Tags Geofences
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Geofence ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /geofences/{id}/devices [get]
func (h *GeofenceHandler) GetDevices(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	devices, err := h.geofenceService.GetDevices(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": devices})
}

// GetEvents returns geofence events
// @Summary Get geofence events
// @Description Get events for a geofence
// @Tags Geofences
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Geofence ID"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /geofences/{id}/events [get]
func (h *GeofenceHandler) GetEvents(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	events, total, err := h.geofenceService.GetEvents(c.Request.Context(), uint(id), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  events,
		"total": total,
		"page":  page,
	})
}

// CheckLocation checks if a location is inside a geofence
// @Summary Check location in geofence
// @Description Check if a location is inside a geofence
// @Tags Geofences
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Geofence ID"
// @Param location body object true "Location coordinates"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /geofences/{id}/check [post]
func (h *GeofenceHandler) CheckLocation(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Lat float64 `json:"lat" binding:"required"`
		Lon float64 `json:"lon" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	geofence, err := h.geofenceService.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "geofence not found"})
		return
	}

	isInside, err := h.geofenceService.CheckPointInGeofence(req.Lat, req.Lon, geofence)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"is_inside": isInside,
		"geofence":  geofence,
	})
}
