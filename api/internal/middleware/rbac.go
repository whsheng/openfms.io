package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"openfms/api/internal/model"
)

// RBACMiddleware RBAC权限中间件
type RBACMiddleware struct {
	db *gorm.DB
}

// NewRBACMiddleware 创建RBAC中间件
func NewRBACMiddleware(db *gorm.DB) *RBACMiddleware {
	return &RBACMiddleware{db: db}
}

// RequirePermission 检查指定权限
func (m *RBACMiddleware) RequirePermission(permissionCode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		// 获取用户角色
		var user model.User
		if err := m.db.First(&user, userID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			c.Abort()
			return
		}

		// 超级管理员拥有所有权限
		if user.RoleID != nil {
			var role model.Role
			if err := m.db.First(&role, *user.RoleID).Error; err == nil {
				if role.Code == "super_admin" {
					c.Next()
					return
				}
			}
		}

		// 检查权限
		if !m.hasPermission(user.ID, permissionCode) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "permission denied",
				"permission": permissionCode,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyPermission 检查任意权限（满足一个即可）
func (m *RBACMiddleware) RequireAnyPermission(permissionCodes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		// 获取用户角色
		var user model.User
		if err := m.db.First(&user, userID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			c.Abort()
			return
		}

		// 超级管理员拥有所有权限
		if user.RoleID != nil {
			var role model.Role
			if err := m.db.First(&role, *user.RoleID).Error; err == nil {
				if role.Code == "super_admin" {
					c.Next()
					return
				}
			}
		}

		// 检查任意权限
		for _, code := range permissionCodes {
			if m.hasPermission(user.ID, code) {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"error": "permission denied",
			"permissions": permissionCodes,
		})
		c.Abort()
	}
}

// RequireAllPermissions 检查所有权限（必须全部满足）
func (m *RBACMiddleware) RequireAllPermissions(permissionCodes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		// 获取用户角色
		var user model.User
		if err := m.db.First(&user, userID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			c.Abort()
			return
		}

		// 超级管理员拥有所有权限
		if user.RoleID != nil {
			var role model.Role
			if err := m.db.First(&role, *user.RoleID).Error; err == nil {
				if role.Code == "super_admin" {
					c.Next()
					return
				}
			}
		}

		// 检查所有权限
		missingPermissions := []string{}
		for _, code := range permissionCodes {
			if !m.hasPermission(user.ID, code) {
				missingPermissions = append(missingPermissions, code)
			}
		}

		if len(missingPermissions) > 0 {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "permission denied",
				"missing_permissions": missingPermissions,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// hasPermission 检查用户是否有指定权限
func (m *RBACMiddleware) hasPermission(userID int, permissionCode string) bool {
	var user model.User
	if err := m.db.First(&user, userID).Error; err != nil {
		return false
	}

	if user.RoleID == nil {
		return false
	}

	var count int64
	m.db.Table("permissions").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ? AND permissions.code = ?", *user.RoleID, permissionCode).
		Count(&count)

	return count > 0
}

// GetUserPermissions 获取用户所有权限
func (m *RBACMiddleware) GetUserPermissions(userID int) []string {
	var user model.User
	if err := m.db.First(&user, userID).Error; err != nil {
		return []string{}
	}

	if user.RoleID == nil {
		return []string{}
	}

	var permissions []string
	m.db.Table("permissions").
		Select("permissions.code").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", *user.RoleID).
		Pluck("permissions.code", &permissions)

	return permissions
}

// PermissionChecker 权限检查器（用于在handler中检查权限）
type PermissionChecker struct {
	db     *gorm.DB
	userID int
}

// NewPermissionChecker 创建权限检查器
func NewPermissionChecker(db *gorm.DB, userID int) *PermissionChecker {
	return &PermissionChecker{db: db, userID: userID}
}

// HasPermission 检查权限
func (c *PermissionChecker) HasPermission(permissionCode string) bool {
	var user model.User
	if err := c.db.First(&user, c.userID).Error; err != nil {
		return false
	}

	// 超级管理员拥有所有权限
	if user.RoleID != nil {
		var role model.Role
		if err := c.db.First(&role, *user.RoleID).Error; err == nil {
			if role.Code == "super_admin" {
				return true
			}
		}
	}

	if user.RoleID == nil {
		return false
	}

	var count int64
	c.db.Table("permissions").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ? AND permissions.code = ?", *user.RoleID, permissionCode).
		Count(&count)

	return count > 0
}

// HasAnyPermission 检查任意权限
func (c *PermissionChecker) HasAnyPermission(permissionCodes ...string) bool {
	for _, code := range permissionCodes {
		if c.HasPermission(code) {
			return true
		}
	}
	return false
}

// HasAllPermissions 检查所有权限
func (c *PermissionChecker) HasAllPermissions(permissionCodes ...string) bool {
	for _, code := range permissionCodes {
		if !c.HasPermission(code) {
			return false
		}
	}
	return true
}

// GetRoleCode 获取用户角色编码
func (c *PermissionChecker) GetRoleCode() string {
	var user model.User
	if err := c.db.First(&user, c.userID).Error; err != nil {
		return ""
	}

	if user.RoleID == nil {
		return ""
	}

	var role model.Role
	if err := c.db.First(&role, *user.RoleID).Error; err != nil {
		return ""
	}

	return role.Code
}

// IsSuperAdmin 是否是超级管理员
func (c *PermissionChecker) IsSuperAdmin() bool {
	return c.GetRoleCode() == "super_admin"
}

// IsAdmin 是否是管理员
func (c *PermissionChecker) IsAdmin() bool {
	roleCode := c.GetRoleCode()
	return roleCode == "super_admin" || roleCode == "admin"
}
