// 登录日志审计

package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"openfms/api/internal/model"
)

// AuditHandler 审计处理器
type AuditHandler struct {
	db *gorm.DB
}

// NewAuditHandler 创建审计处理器
func NewAuditHandler(db *gorm.DB) *AuditHandler {
	return &AuditHandler{db: db}
}

// RegisterRoutes 注册路由
func (h *AuditHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/audit-logs", h.ListLogs)
	r.GET("/audit-logs/stats", h.GetStats)
}

// ListLogs 获取审计日志列表
func (h *AuditHandler) ListLogs(c *gin.Context) {
	var logs []model.LoginLog
	
	query := h.db.Order("created_at DESC")
	
	// 筛选条件
	if username := c.Query("username"); username != "" {
		query = query.Where("username = ?", username)
	}
	if action := c.Query("action"); action != "" {
		query = query.Where("action = ?", action)
	}
	if startTime := c.Query("start_time"); startTime != "" {
		query = query.Where("created_at >= ?", startTime)
	}
	if endTime := c.Query("end_time"); endTime != "" {
		query = query.Where("created_at <= ?", endTime)
	}
	
	// 分页
	page := 1
	pageSize := 20
	if p := c.Query("page"); p != "" {
		// 解析page
	}
	
	var total int64
	query.Model(&model.LoginLog{}).Count(&total)
	
	offset := (page - 1) * pageSize
	query.Offset(offset).Limit(pageSize).Find(&logs)
	
	c.JSON(http.StatusOK, gin.H{
		"list":      logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetStats 获取统计
func (h *AuditHandler) GetStats(c *gin.Context) {
	// 今日登录次数
	var todayLogins int64
	h.db.Model(&model.LoginLog{}).Where("action = ? AND DATE(created_at) = CURRENT_DATE", "login").Count(&todayLogins)
	
	// 失败次数
	var failedLogins int64
	h.db.Model(&model.LoginLog{}).Where("action = ? AND success = ? AND DATE(created_at) = CURRENT_DATE", "login", false).Count(&failedLogins)
	
	// 活跃用户数
	var activeUsers int64
	h.db.Model(&model.LoginLog{}).Where("action = ? AND DATE(created_at) = CURRENT_DATE", "login").Distinct("user_id").Count(&activeUsers)
	
	c.JSON(http.StatusOK, gin.H{
		"today_logins":  todayLogins,
		"failed_logins": failedLogins,
		"active_users":  activeUsers,
	})
}

// RecordLogin 记录登录（在登录时调用）
func (h *AuditHandler) RecordLogin(userID int, username, ip, userAgent string, success bool, errorMsg string) {
	log := model.LoginLog{
		UserID:    userID,
		Username:  username,
		Action:    "login",
		IP:        ip,
		UserAgent: userAgent,
		Success:   success,
		ErrorMsg:  errorMsg,
		CreatedAt: time.Now(),
	}
	h.db.Create(&log)
}
