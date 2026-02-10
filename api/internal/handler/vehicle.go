package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"openfms/api/internal/model"
)

// VehicleHandler 车辆处理器
type VehicleHandler struct {
	db *gorm.DB
}

// NewVehicleHandler 创建车辆处理器
func NewVehicleHandler(db *gorm.DB) *VehicleHandler {
	return &VehicleHandler{db: db}
}

// RegisterRoutes 注册路由
func (h *VehicleHandler) RegisterRoutes(r *gin.RouterGroup) {
	vehicles := r.Group("/vehicles")
	{
		vehicles.GET("", h.ListVehicles)
		vehicles.POST("", h.CreateVehicle)
		vehicles.GET("/:id", h.GetVehicle)
		vehicles.PUT("/:id", h.UpdateVehicle)
		vehicles.DELETE("/:id", h.DeleteVehicle)
		
		// 设备绑定
		vehicles.POST("/:id/bind", h.BindDevice)
		vehicles.POST("/:id/unbind", h.UnbindDevice)
		vehicles.GET("/:id/history", h.GetBindHistory)
	}
	
	// 车辆分组
	groups := r.Group("/vehicle-groups")
	{
		groups.GET("", h.ListGroups)
		groups.POST("", h.CreateGroup)
		groups.PUT("/:id", h.UpdateGroup)
		groups.DELETE("/:id", h.DeleteGroup)
		groups.POST("/:id/vehicles", h.AddToGroup)
		groups.DELETE("/:id/vehicles/:vehicle_id", h.RemoveFromGroup)
	}
}

// ListVehicles 获取车辆列表
func (h *VehicleHandler) ListVehicles(c *gin.Context) {
	var query model.VehicleListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := h.db.Model(&model.Vehicle{}).Preload("Device")

	if query.PlateNumber != "" {
		db = db.Where("plate_number LIKE ?", "%"+query.PlateNumber+"%")
	}
	if query.VehicleType != "" {
		db = db.Where("vehicle_type = ?", query.VehicleType)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.DeviceID != "" {
		db = db.Where("device_id = ?", query.DeviceID)
	}
	if query.GroupID > 0 {
		db = db.Joins("JOIN vehicle_group_members ON vehicle_group_members.vehicle_id = vehicles.id").
			Where("vehicle_group_members.group_id = ?", query.GroupID)
	}

	var total int64
	db.Count(&total)

	var vehicles []model.Vehicle
	offset := (query.Page - 1) * query.PageSize
	db.Order("created_at DESC").Offset(offset).Limit(query.PageSize).Find(&vehicles)

	c.JSON(http.StatusOK, gin.H{
		"list":     vehicles,
		"total":    total,
		"page":     query.Page,
		"page_size": query.PageSize,
	})
}

// GetVehicle 获取车辆详情
func (h *VehicleHandler) GetVehicle(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vehicle id"})
		return
	}

	var vehicle model.Vehicle
	if err := h.db.Preload("Device").First(&vehicle, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "vehicle not found"})
		return
	}

	c.JSON(http.StatusOK, vehicle)
}

// CreateVehicle 创建车辆
func (h *VehicleHandler) CreateVehicle(c *gin.Context) {
	var req model.CreateVehicleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查车牌号是否已存在
	var count int64
	h.db.Model(&model.Vehicle{}).Where("plate_number = ?", req.PlateNumber).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plate number already exists"})
		return
	}

	vehicle := model.Vehicle{
		PlateNumber:     req.PlateNumber,
		PlateColor:      req.PlateColor,
		VehicleType:     req.VehicleType,
		Brand:           req.Brand,
		Model:           req.Model,
		Color:           req.Color,
		VIN:             req.VIN,
		EngineNo:        req.EngineNo,
		OwnerName:       req.OwnerName,
		OwnerPhone:      req.OwnerPhone,
		OwnerIDCard:     req.OwnerIDCard,
		RegistrationNo:  req.RegistrationNo,
		TransportNo:     req.TransportNo,
		InsuranceNo:     req.InsuranceNo,
		InsuranceExpire: req.InsuranceExpire,
		DeviceID:        req.DeviceID,
		Remark:          req.Remark,
		Status:          "active",
	}

	if err := h.db.Create(&vehicle).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 如果绑定了设备，记录历史
	if req.DeviceID != nil && *req.DeviceID != "" {
		h.db.Create(&model.DeviceVehicleHistory{
			DeviceID:   *req.DeviceID,
			VehicleID:  vehicle.ID,
			Action:     "bind",
			OperatedAt: time.Now(),
		})
	}

	c.JSON(http.StatusCreated, vehicle)
}

// UpdateVehicle 更新车辆
func (h *VehicleHandler) UpdateVehicle(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vehicle id"})
		return
	}

	var req model.UpdateVehicleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}
	if req.PlateNumber != "" {
		updates["plate_number"] = req.PlateNumber
	}
	if req.PlateColor != "" {
		updates["plate_color"] = req.PlateColor
	}
	if req.VehicleType != "" {
		updates["vehicle_type"] = req.VehicleType
	}
	if req.Brand != "" {
		updates["brand"] = req.Brand
	}
	if req.Model != "" {
		updates["model"] = req.Model
	}
	if req.Color != "" {
		updates["color"] = req.Color
	}
	if req.VIN != "" {
		updates["vin"] = req.VIN
	}
	if req.EngineNo != "" {
		updates["engine_no"] = req.EngineNo
	}
	if req.OwnerName != "" {
		updates["owner_name"] = req.OwnerName
	}
	if req.OwnerPhone != "" {
		updates["owner_phone"] = req.OwnerPhone
	}
	if req.OwnerIDCard != "" {
		updates["owner_idcard"] = req.OwnerIDCard
	}
	if req.RegistrationNo != "" {
		updates["registration_no"] = req.RegistrationNo
	}
	if req.TransportNo != "" {
		updates["transport_no"] = req.TransportNo
	}
	if req.InsuranceNo != "" {
		updates["insurance_no"] = req.InsuranceNo
	}
	if req.InsuranceExpire != nil {
		updates["insurance_expire"] = req.InsuranceExpire
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if req.Remark != "" {
		updates["remark"] = req.Remark
	}

	if err := h.db.Model(&model.Vehicle{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "vehicle updated"})
}

// DeleteVehicle 删除车辆
func (h *VehicleHandler) DeleteVehicle(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vehicle id"})
		return
	}

	if err := h.db.Delete(&model.Vehicle{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "vehicle deleted"})
}

// BindDevice 绑定设备
func (h *VehicleHandler) BindDevice(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vehicle id"})
		return
	}

	var req model.BindDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查设备是否已被其他车辆绑定
	var existing model.Vehicle
	h.db.Where("device_id = ? AND id != ?", req.DeviceID, id).First(&existing)
	if existing.ID > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device already bound to another vehicle"})
		return
	}

	// 更新车辆设备绑定
	if err := h.db.Model(&model.Vehicle{}).Where("id = ?", id).Update("device_id", req.DeviceID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 记录历史
	userID, _ := c.Get("userID")
	h.db.Create(&model.DeviceVehicleHistory{
		DeviceID:   req.DeviceID,
		VehicleID:  id,
		Action:     "bind",
		OperatedBy: userID.(*int),
		OperatedAt: time.Now(),
		Remark:     req.Remark,
	})

	c.JSON(http.StatusOK, gin.H{"message": "device bound"})
}

// UnbindDevice 解绑设备
func (h *VehicleHandler) UnbindDevice(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vehicle id"})
		return
	}

	var vehicle model.Vehicle
	h.db.First(&vehicle, id)

	if vehicle.DeviceID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "vehicle has no bound device"})
		return
	}

	deviceID := *vehicle.DeviceID

	// 更新车辆设备绑定
	if err := h.db.Model(&model.Vehicle{}).Where("id = ?", id).Update("device_id", nil).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 记录历史
	userID, _ := c.Get("userID")
	h.db.Create(&model.DeviceVehicleHistory{
		DeviceID:   deviceID,
		VehicleID:  id,
		Action:     "unbind",
		OperatedBy: userID.(*int),
		OperatedAt: time.Now(),
	})

	c.JSON(http.StatusOK, gin.H{"message": "device unbound"})
}

// GetBindHistory 获取绑定历史
func (h *VehicleHandler) GetBindHistory(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vehicle id"})
		return
	}

	var history []model.DeviceVehicleHistory
	h.db.Where("vehicle_id = ?", id).Order("operated_at DESC").Find(&history)

	c.JSON(http.StatusOK, history)
}

// ========== 车辆分组 ==========

// ListGroups 获取分组列表
func (h *VehicleHandler) ListGroups(c *gin.Context) {
	var groups []model.VehicleGroup
	h.db.Find(&groups)

	// 统计每个分组的车辆数
	for i := range groups {
		var count int64
		h.db.Model(&model.VehicleGroupMember{}).Where("group_id = ?", groups[i].ID).Count(&count)
		groups[i].VehicleCount = count
	}

	c.JSON(http.StatusOK, groups)
}

// CreateGroup 创建分组
func (h *VehicleHandler) CreateGroup(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Color       string `json:"color"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("userID")
	group := model.VehicleGroup{
		Name:        req.Name,
		Color:       req.Color,
		Description: req.Description,
		CreatedBy:   userID.(*int),
	}

	if err := h.db.Create(&group).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, group)
}

// UpdateGroup 更新分组
func (h *VehicleHandler) UpdateGroup(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
		return
	}

	var req struct {
		Name        string `json:"name"`
		Color       string `json:"color"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Color != "" {
		updates["color"] = req.Color
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}

	if err := h.db.Model(&model.VehicleGroup{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "group updated"})
}

// DeleteGroup 删除分组
func (h *VehicleHandler) DeleteGroup(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
		return
	}

	// 先删除分组关联
	h.db.Where("group_id = ?", id).Delete(&model.VehicleGroupMember{})

	// 删除分组
	if err := h.db.Delete(&model.VehicleGroup{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "group deleted"})
}

// AddToGroup 添加车辆到分组
func (h *VehicleHandler) AddToGroup(c *gin.Context) {
	groupID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
		return
	}

	var req struct {
		VehicleIDs []int `json:"vehicle_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for _, vehicleID := range req.VehicleIDs {
		h.db.Create(&model.VehicleGroupMember{
			VehicleID: vehicleID,
			GroupID:   groupID,
		})
	}

	c.JSON(http.StatusOK, gin.H{"message": "vehicles added to group"})
}

// RemoveFromGroup 从分组移除车辆
func (h *VehicleHandler) RemoveFromGroup(c *gin.Context) {
	groupID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
		return
	}

	vehicleID, err := strconv.Atoi(c.Param("vehicle_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vehicle id"})
		return
	}

	h.db.Where("group_id = ? AND vehicle_id = ?", groupID, vehicleID).Delete(&model.VehicleGroupMember{})

	c.JSON(http.StatusOK, gin.H{"message": "vehicle removed from group"})
}
