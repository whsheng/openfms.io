package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"openfms/api/internal/model"
	"openfms/api/internal/service"
)

// DeviceHandler handles device-related requests
type DeviceHandler struct {
	deviceService *service.DeviceService
}

// NewDeviceHandler creates a new device handler
func NewDeviceHandler(deviceService *service.DeviceService) *DeviceHandler {
	return &DeviceHandler{deviceService: deviceService}
}

// List returns list of devices
func (h *DeviceHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	devices, total, err := h.deviceService.List(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  devices,
		"total": total,
		"page":  page,
	})
}

// Get returns a single device
func (h *DeviceHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	device, err := h.deviceService.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}

	c.JSON(http.StatusOK, device)
}

// Create creates a new device
func (h *DeviceHandler) Create(c *gin.Context) {
	var device model.Device
	if err := c.ShouldBindJSON(&device); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.deviceService.Create(c.Request.Context(), &device); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, device)
}

// Update updates a device
func (h *DeviceHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var device model.Device
	if err := c.ShouldBindJSON(&device); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	device.ID = uint(id)
	if err := h.deviceService.Update(c.Request.Context(), &device); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, device)
}

// Delete deletes a device
func (h *DeviceHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.deviceService.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// GetShadow returns device shadow (real-time state)
func (h *DeviceHandler) GetShadow(c *gin.Context) {
	deviceID := c.Param("device_id")

	shadow, err := h.deviceService.GetShadow(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device shadow not found"})
		return
	}

	c.JSON(http.StatusOK, shadow)
}

// SendCommand sends a command to a device
func (h *DeviceHandler) SendCommand(c *gin.Context) {
	deviceID := c.Param("device_id")

	var cmd struct {
		Type   string                 `json:"type" binding:"required"`
		Params map[string]interface{} `json:"params"`
	}

	if err := c.ShouldBindJSON(&cmd); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.deviceService.SendCommand(c.Request.Context(), deviceID, cmd.Type, cmd.Params); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "command sent"})
}
