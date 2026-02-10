package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"openfms/api/internal/model"
	"openfms/api/internal/service"
)

// DeviceHandler handles device-related requests
type DeviceHandler struct {
	deviceService       *service.DeviceService
	deviceImportService *service.DeviceImportService
}

// NewDeviceHandler creates a new device handler
func NewDeviceHandler(deviceService *service.DeviceService) *DeviceHandler {
	return &DeviceHandler{deviceService: deviceService}
}

// SetDeviceImportService sets the device import service
func (h *DeviceHandler) SetDeviceImportService(importService *service.DeviceImportService) {
	h.deviceImportService = importService
}

// getUserIDFromContext 从上下文中获取用户ID
func getUserIDFromContext(c *gin.Context) uint {
	if claims, exists := c.Get("claims"); exists {
		if jwtClaims, ok := claims.(jwt.MapClaims); ok {
			if userID, ok := jwtClaims["user_id"].(float64); ok {
				return uint(userID)
			}
		}
	}
	return 0
}

// List returns list of devices
// @Summary List devices
// @Description Get a paginated list of devices
// @Tags Devices
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /devices [get]
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
// @Summary Get device
// @Description Get a single device by ID
// @Tags Devices
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Device ID"
// @Success 200 {object} model.Device
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /devices/{id} [get]
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
// @Summary Create device
// @Description Create a new device
// @Tags Devices
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param device body model.Device true "Device data"
// @Success 201 {object} model.Device
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /devices [post]
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
// @Summary Update device
// @Description Update an existing device
// @Tags Devices
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Device ID"
// @Param device body model.Device true "Device data"
// @Success 200 {object} model.Device
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /devices/{id} [put]
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
// @Summary Delete device
// @Description Delete a device by ID
// @Tags Devices
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Device ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /devices/{id} [delete]
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
// @Summary Get device shadow
// @Description Get device shadow (real-time state)
// @Tags Devices
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param device_id path string true "Device ID"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /devices/{device_id}/shadow [get]
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
// @Summary Send command to device
// @Description Send a command to a device
// @Tags Devices
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param device_id path string true "Device ID"
// @Param command body object true "Command data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /devices/{device_id}/commands [post]
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

// DownloadImportTemplate 下载设备导入模板
// @Summary Download device import template
// @Description Download Excel template for importing devices
// @Tags Devices
// @Accept json
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Security BearerAuth
// @Success 200 {file} binary
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /devices/import-template [get]
func (h *DeviceHandler) DownloadImportTemplate(c *gin.Context) {
	if h.deviceImportService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "import service not initialized"})
		return
	}

	buf, err := h.deviceImportService.GenerateImportTemplate()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=device_import_template.xlsx")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

// PreviewImport 预览设备导入数据
// @Summary Preview device import
// @Description Preview and validate device import data without actually importing
// @Tags Devices
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "Excel file"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /devices/import-preview [post]
func (h *DeviceHandler) PreviewImport(c *gin.Context) {
	if h.deviceImportService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "import service not initialized"})
		return
	}

	// 获取上传的文件
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传文件"})
		return
	}
	defer file.Close()

	// 解析Excel
	rows, err := h.deviceImportService.ParseExcel(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(rows) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件中没有有效数据"})
		return
	}

	// 验证并返回预览
	preview := h.deviceImportService.PreviewImport(rows)
	c.JSON(http.StatusOK, preview)
}

// ImportDevices 批量导入设备
// @Summary Import devices from Excel
// @Description Import devices from Excel file (async)
// @Tags Devices
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "Excel file"
// @Success 202 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /devices/import [post]
func (h *DeviceHandler) ImportDevices(c *gin.Context) {
	if h.deviceImportService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "import service not initialized"})
		return
	}

	// 获取上传的文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传文件"})
		return
	}
	defer file.Close()

	// 验证文件类型
	if header.Header.Get("Content-Type") != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" &&
		header.Header.Get("Content-Type") != "application/vnd.ms-excel" {
		// 检查文件扩展名
		filename := header.Filename
		if len(filename) < 5 || (filename[len(filename)-5:] != ".xlsx" && filename[len(filename)-4:] != ".xls") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请上传Excel文件(.xlsx或.xls)"})
			return
		}
	}

	// 解析Excel
	rows, err := h.deviceImportService.ParseExcel(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(rows) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件中没有有效数据"})
		return
	}

	// 验证数据
	validatedRows := h.deviceImportService.ValidateRows(rows)

	// 生成任务ID
	taskID := fmt.Sprintf("import_%d", time.Now().UnixNano())

	// 获取当前用户ID
	userID := getUserIDFromContext(c)

	// 异步导入
	h.deviceImportService.ImportDevices(c.Request.Context(), taskID, validatedRows, userID)

	c.JSON(http.StatusAccepted, gin.H{
		"task_id":     taskID,
		"total":       len(rows),
		"message":     "导入任务已启动",
		"check_url":   fmt.Sprintf("/api/v1/devices/import/%s/status", taskID),
	})
}

// GetImportStatus 获取导入任务状态
// @Summary Get import task status
// @Description Get the status of a device import task
// @Tags Devices
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param task_id path string true "Task ID"
// @Success 200 {object} model.DeviceImportResult
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /devices/import/{task_id}/status [get]
func (h *DeviceHandler) GetImportStatus(c *gin.Context) {
	if h.deviceImportService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "import service not initialized"})
		return
	}

	taskID := c.Param("task_id")

	result, ok := h.deviceImportService.GetImportResult(taskID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// DownloadImportErrorReport 下载导入错误报告
// @Summary Download import error report
// @Description Download error report for failed imports
// @Tags Devices
// @Accept json
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Security BearerAuth
// @Param task_id path string true "Task ID"
// @Success 200 {file} binary
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /devices/import/{task_id}/errors [get]
func (h *DeviceHandler) DownloadImportErrorReport(c *gin.Context) {
	if h.deviceImportService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "import service not initialized"})
		return
	}

	taskID := c.Param("task_id")

	result, ok := h.deviceImportService.GetImportResult(taskID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	if result.ErrorCount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有错误记录"})
		return
	}

	// 获取原始数据并生成错误报告
	// 这里简化处理，实际应该从存储中重新获取
	c.JSON(http.StatusNotImplemented, gin.H{"error": "功能开发中"})
}
