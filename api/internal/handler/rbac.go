package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"openfms/api/internal/model"
)

// RBACHandler RBAC处理器
type RBACHandler struct {
	db *gorm.DB
}

// NewRBACHandler 创建RBAC处理器
func NewRBACHandler(db *gorm.DB) *RBACHandler {
	return &RBACHandler{db: db}
}

// RegisterRoutes 注册路由
func (h *RBACHandler) RegisterRoutes(r *gin.RouterGroup) {
	// 角色管理
	roles := r.Group("/roles")
	{
		roles.GET("", h.ListRoles)
		roles.POST("", h.CreateRole)
		roles.GET("/:id", h.GetRole)
		roles.PUT("/:id", h.UpdateRole)
		roles.DELETE("/:id", h.DeleteRole)
		roles.GET("/:id/permissions", h.GetRolePermissions)
	}

	// 权限管理
	permissions := r.Group("/permissions")
	{
		permissions.GET("", h.ListPermissions)
		permissions.GET("/groups", h.GetPermissionGroups)
	}

	// 用户角色管理
	users := r.Group("/users")
	{
		users.GET("", h.ListUsers)
		users.POST("", h.CreateUser)
		users.GET("/:id", h.GetUser)
		users.PUT("/:id", h.UpdateUser)
		users.DELETE("/:id", h.DeleteUser)
		users.POST("/:id/role", h.AssignRole)
		users.GET("/:id/permissions", h.GetUserPermissions)
	}

	// 当前用户权限
	r.GET("/me/permissions", h.GetMyPermissions)
	r.POST("/me/check-permission", h.CheckPermission)
}

// ========== 角色管理 ==========

// ListRoles 获取角色列表
func (h *RBACHandler) ListRoles(c *gin.Context) {
	var roles []model.Role
	if err := h.db.Find(&roles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, roles)
}

// GetRole 获取角色详情
func (h *RBACHandler) GetRole(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role id"})
		return
	}

	var role model.Role
	if err := h.db.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}

	// 获取权限
	var permissions []model.Permission
	h.db.Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", id).
		Find(&permissions)

	c.JSON(http.StatusOK, model.RoleWithPermissions{
		Role:        role,
		Permissions: permissions,
	})
}

// CreateRole 创建角色
func (h *RBACHandler) CreateRole(c *gin.Context) {
	var req model.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	role := model.Role{
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		IsSystem:    false,
	}

	if err := h.db.Create(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 分配权限
	if len(req.PermissionIDs) > 0 {
		for _, permID := range req.PermissionIDs {
			h.db.Create(&model.RolePermission{
				RoleID:       role.ID,
				PermissionID: permID,
			})
		}
	}

	c.JSON(http.StatusCreated, role)
}

// UpdateRole 更新角色
func (h *RBACHandler) UpdateRole(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role id"})
		return
	}

	var role model.Role
	if err := h.db.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}

	// 系统角色不可修改
	if role.IsSystem {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot modify system role"})
		return
	}

	var req model.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}

	if err := h.db.Model(&role).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 更新权限
	if req.PermissionIDs != nil {
		// 删除旧权限
		h.db.Where("role_id = ?", id).Delete(&model.RolePermission{})
		// 添加新权限
		for _, permID := range req.PermissionIDs {
			h.db.Create(&model.RolePermission{
				RoleID:       id,
				PermissionID: permID,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "role updated"})
}

// DeleteRole 删除角色
func (h *RBACHandler) DeleteRole(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role id"})
		return
	}

	var role model.Role
	if err := h.db.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}

	// 系统角色不可删除
	if role.IsSystem {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete system role"})
		return
	}

	// 检查是否有用户使用此角色
	var count int64
	h.db.Model(&model.User{}).Where("role_id = ?", id).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role is in use by users"})
		return
	}

	if err := h.db.Delete(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "role deleted"})
}

// GetRolePermissions 获取角色权限
func (h *RBACHandler) GetRolePermissions(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role id"})
		return
	}

	var permissions []model.Permission
	h.db.Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", id).
		Find(&permissions)

	c.JSON(http.StatusOK, permissions)
}

// ========== 权限管理 ==========

// ListPermissions 获取权限列表
func (h *RBACHandler) ListPermissions(c *gin.Context) {
	var permissions []model.Permission
	if err := h.db.Find(&permissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, permissions)
}

// GetPermissionGroups 获取权限分组
func (h *RBACHandler) GetPermissionGroups(c *gin.Context) {
	var permissions []model.Permission
	h.db.Find(&permissions)

	// 按分组组织
	groups := map[string]*model.PermissionGroup{
		"device":    {Name: "device", Label: "设备管理", Permissions: []model.Permission{}},
		"position":  {Name: "position", Label: "位置轨迹", Permissions: []model.Permission{}},
		"geofence":  {Name: "geofence", Label: "电子围栏", Permissions: []model.Permission{}},
		"alarm":     {Name: "alarm", Label: "报警中心", Permissions: []model.Permission{}},
		"user":      {Name: "user", Label: "用户管理", Permissions: []model.Permission{}},
		"setting":   {Name: "setting", Label: "系统设置", Permissions: []model.Permission{}},
		"other":     {Name: "other", Label: "其他", Permissions: []model.Permission{}},
	}

	for _, perm := range permissions {
		if group, ok := groups[perm.GroupName]; ok {
			group.Permissions = append(group.Permissions, perm)
		}
	}

	// 转换为列表
	result := make([]model.PermissionGroup, 0, len(groups))
	for _, group := range groups {
		if len(group.Permissions) > 0 {
			result = append(result, *group)
		}
	}

	c.JSON(http.StatusOK, result)
}

// ========== 用户管理 ==========

// ListUsers 获取用户列表
func (h *RBACHandler) ListUsers(c *gin.Context) {
	var users []model.UserWithRole
	h.db.Table("users").
		Select("users.*, roles.id as role_id, roles.name as role_name, roles.code as role_code").
		Joins("LEFT JOIN roles ON roles.id = users.role_id").
		Find(&users)

	c.JSON(http.StatusOK, users)
}

// GetUser 获取用户详情
func (h *RBACHandler) GetUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var user model.UserWithRole
	result := h.db.Table("users").
		Select("users.*, roles.id as role_id, roles.name as role_name, roles.code as role_code").
		Joins("LEFT JOIN roles ON roles.id = users.role_id").
		Where("users.id = ?", id).
		First(&user)

	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// CreateUser 创建用户
func (h *RBACHandler) CreateUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required,min=6"`
		Nickname string `json:"nickname"`
		RoleID   int    `json:"role_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查角色是否存在
	var role model.Role
	if err := h.db.First(&role, req.RoleID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role not found"})
		return
	}

	// 检查用户名是否已存在
	var count int64
	h.db.Model(&model.User{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username already exists"})
		return
	}

	user := model.User{
		Username: req.Username,
		Nickname: req.Nickname,
		RoleID:   &req.RoleID,
	}
	user.SetPassword(req.Password)

	if err := h.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// UpdateUser 更新用户
func (h *RBACHandler) UpdateUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var req struct {
		Nickname string `json:"nickname"`
		Password string `json:"password"`
		RoleID   *int   `json:"role_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}
	if req.Nickname != "" {
		updates["nickname"] = req.Nickname
	}
	if req.RoleID != nil {
		updates["role_id"] = *req.RoleID
	}

	if err := h.db.Model(&model.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 更新密码
	if req.Password != "" {
		var user model.User
		h.db.First(&user, id)
		user.SetPassword(req.Password)
		h.db.Save(&user)
	}

	c.JSON(http.StatusOK, gin.H{"message": "user updated"})
}

// DeleteUser 删除用户
func (h *RBACHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	// 不能删除自己
	currentUserID, _ := c.Get("userID")
	if currentUserID.(int) == id {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete yourself"})
		return
	}

	if err := h.db.Delete(&model.User{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user deleted"})
}

// AssignRole 分配角色
func (h *RBACHandler) AssignRole(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var req model.AssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查角色是否存在
	var role model.Role
	if err := h.db.First(&role, req.RoleID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role not found"})
		return
	}

	if err := h.db.Model(&model.User{}).Where("id = ?", userID).Update("role_id", req.RoleID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "role assigned"})
}

// GetUserPermissions 获取用户权限
func (h *RBACHandler) GetUserPermissions(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	permissions := h.getUserPermissions(userID)
	c.JSON(http.StatusOK, gin.H{"permissions": permissions})
}

// GetMyPermissions 获取当前用户权限
func (h *RBACHandler) GetMyPermissions(c *gin.Context) {
	userID, _ := c.Get("userID")
	permissions := h.getUserPermissions(userID.(int))
	
	// 获取角色信息
	var user model.User
	h.db.First(&user, userID)
	
	var roleInfo model.UserRoleInfo
	if user.RoleID != nil {
		var role model.Role
		h.db.First(&role, *user.RoleID)
		roleInfo = model.UserRoleInfo{
			RoleID:   role.ID,
			RoleName: role.Name,
			RoleCode: role.Code,
		}
	}

	c.JSON(http.StatusOK, model.UserPermissionsResponse{
		UserID:      userID.(int),
		Role:        roleInfo,
		Permissions: permissions,
	})
}

// CheckPermission 检查权限
func (h *RBACHandler) CheckPermission(c *gin.Context) {
	userID, _ := c.Get("userID")

	var req model.CheckPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	permissions := h.getUserPermissions(userID.(int))
	hasPermission := false
	for _, perm := range permissions {
		if perm == req.PermissionCode {
			hasPermission = true
			break
		}
	}

	c.JSON(http.StatusOK, model.PermissionCheckResult{
		HasPermission: hasPermission,
		Permission:    req.PermissionCode,
	})
}

// getUserPermissions 获取用户权限列表（内部方法）
func (h *RBACHandler) getUserPermissions(userID int) []string {
	var user model.User
	h.db.First(&user, userID)

	if user.RoleID == nil {
		return []string{}
	}

	var permissions []string
	h.db.Table("permissions").
		Select("permissions.code").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", *user.RoleID).
		Pluck("permissions.code", &permissions)

	return permissions
}
