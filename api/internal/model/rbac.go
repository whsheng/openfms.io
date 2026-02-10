package model

import (
	"time"
)

// Role 角色
type Role struct {
	ID          int       `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"type:varchar(50);not null;unique"`
	Code        string    `json:"code" gorm:"type:varchar(50);not null;unique"`
	Description string    `json:"description,omitempty" gorm:"type:varchar(200)"`
	IsSystem    bool      `json:"is_system" gorm:"column:is_system;not null;default:false"`
	CreatedAt   time.Time `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"not null;default:now()"`
	
	// 关联
	Permissions []Permission `json:"permissions,omitempty" gorm:"many2many:role_permissions;"`
}

func (Role) TableName() string {
	return "roles"
}

// Permission 权限
type Permission struct {
	ID          int       `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"type:varchar(100);not null"`
	Code        string    `json:"code" gorm:"type:varchar(100);not null;unique"`
	Description string    `json:"description,omitempty" gorm:"type:varchar(200)"`
	GroupName   string    `json:"group_name" gorm:"column:group_name;type:varchar(50);not null;default:'other'"`
	CreatedAt   time.Time `json:"created_at" gorm:"not null;default:now()"`
}

func (Permission) TableName() string {
	return "permissions"
}

// RolePermission 角色权限关联
type RolePermission struct {
	ID           int       `json:"id" gorm:"primaryKey"`
	RoleID       int       `json:"role_id" gorm:"column:role_id;not null;index"`
	PermissionID int       `json:"permission_id" gorm:"column:permission_id;not null;index"`
	CreatedAt    time.Time `json:"created_at" gorm:"not null;default:now()"`
}

func (RolePermission) TableName() string {
	return "role_permissions"
}

// UserRole 用户角色关联
type UserRole struct {
	ID        int       `json:"id" gorm:"primaryKey"`
	UserID    int       `json:"user_id" gorm:"column:user_id;not null;index"`
	RoleID    int       `json:"role_id" gorm:"column:role_id;not null;index"`
	CreatedAt time.Time `json:"created_at" gorm:"not null;default:now()"`
}

func (UserRole) TableName() string {
	return "user_roles"
}

// UserWithRole 带角色信息的用户
type UserWithRole struct {
	User
	RoleID   *int    `json:"role_id,omitempty" gorm:"column:role_id"`
	RoleName *string `json:"role_name,omitempty" gorm:"column:role_name"`
	RoleCode *string `json:"role_code,omitempty" gorm:"column:role_code"`
}

// RoleWithPermissions 带权限的角色
type RoleWithPermissions struct {
	Role
	PermissionIDs []int        `json:"permission_ids,omitempty"`
	Permissions   []Permission `json:"permissions,omitempty"`
}

// PermissionGroup 权限分组
type PermissionGroup struct {
	Name        string       `json:"name"`
	Label       string       `json:"label"`
	Permissions []Permission `json:"permissions"`
}

// CreateRoleRequest 创建角色请求
type CreateRoleRequest struct {
	Name          string `json:"name" binding:"required"`
	Code          string `json:"code" binding:"required"`
	Description   string `json:"description"`
	PermissionIDs []int  `json:"permission_ids"`
}

// UpdateRoleRequest 更新角色请求
type UpdateRoleRequest struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	PermissionIDs []int  `json:"permission_ids"`
}

// AssignRoleRequest 分配角色请求
type AssignRoleRequest struct {
	RoleID int `json:"role_id" binding:"required"`
}

// UserRoleInfo 用户角色信息
type UserRoleInfo struct {
	RoleID   int    `json:"role_id"`
	RoleName string `json:"role_name"`
	RoleCode string `json:"role_code"`
}

// CheckPermissionRequest 检查权限请求
type CheckPermissionRequest struct {
	PermissionCode string `json:"permission_code" binding:"required"`
}

// PermissionCheckResult 权限检查结果
type PermissionCheckResult struct {
	HasPermission bool   `json:"has_permission"`
	Permission    string `json:"permission"`
}

// UserPermissionsResponse 用户权限响应
type UserPermissionsResponse struct {
	UserID      int          `json:"user_id"`
	Role        UserRoleInfo `json:"role"`
	Permissions []string     `json:"permissions"` // 权限编码列表
}
