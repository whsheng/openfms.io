// JT808 扩展指令 - 参数查询/设置、终端控制、位置查询

package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"openfms/api/internal/model"
	"openfms/api/internal/service"
)

// JT808ExtendedHandler JT808扩展指令处理器
type JT808ExtendedHandler struct {
	db            *gorm.DB
	commandService *service.CommandService
}

// NewJT808ExtendedHandler 创建处理器
func NewJT808ExtendedHandler(db *gorm.DB, cmdService *service.CommandService) *JT808ExtendedHandler {
	return &JT808ExtendedHandler{
		db:            db,
		commandService: cmdService,
	}
}

// RegisterRoutes 注册路由
func (h *JT808ExtendedHandler) RegisterRoutes(r *gin.RouterGroup) {
	jt808 := r.Group("/devices/:id/jt808")
	{
		// 参数相关
		jt808.POST("/params/query", h.QueryParams)      // 0x8104
		jt808.POST("/params/set", h.SetParams)          // 0x8103
		
		// 终端控制
		jt808.POST("/control", h.ControlTerminal)       // 0x8105
		
		// 位置查询
		jt808.POST("/location/query", h.QueryLocation)  // 0x8201
		
		// 车辆控制
		jt808.POST("/vehicle/control", h.ControlVehicle) // 0x8500
	}
}

// QueryParams 查询终端参数 (0x8104)
func (h *JT808ExtendedHandler) QueryParams(c *gin.Context) {
	deviceID := c.Param("id")
	
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	
	resp, err := h.commandService.SendCommand(ctx, deviceID, model.CmdGetParams, nil, 30*time.Second)
	if err != nil {
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, resp)
}

// SetParams 设置终端参数 (0x8103)
func (h *JT808ExtendedHandler) SetParams(c *gin.Context) {
	deviceID := c.Param("id")
	
	var req struct {
		Params map[string]interface{} `json:"params" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	
	resp, err := h.commandService.SendCommand(ctx, deviceID, model.CmdSetParams, req.Params, 30*time.Second)
	if err != nil {
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, resp)
}

// ControlTerminal 终端控制 (0x8105)
func (h *JT808ExtendedHandler) ControlTerminal(c *gin.Context) {
	deviceID := c.Param("id")
	
	var req struct {
		Command string `json:"command" binding:"required,oneof=reboot reset factory"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	commandMap := map[string]string{
		"reboot":  "1", // 终端复位
		"reset":   "2", // 终端复位
		"factory": "3", // 恢复出厂设置
	}
	
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	
	resp, err := h.commandService.SendCommand(ctx, deviceID, model.CmdTerminalControl, map[string]interface{}{
		"command": commandMap[req.Command],
	}, 30*time.Second)
	if err != nil {
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, resp)
}

// QueryLocation 位置查询 (0x8201)
func (h *JT808ExtendedHandler) QueryLocation(c *gin.Context) {
	deviceID := c.Param("id")
	
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	
	resp, err := h.commandService.SendCommand(ctx, deviceID, model.CmdLocationQuery, nil, 30*time.Second)
	if err != nil {
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, resp)
}

// ControlVehicle 车辆控制 (0x8500)
func (h *JT808ExtendedHandler) ControlVehicle(c *gin.Context) {
	deviceID := c.Param("id")
	
	var req struct {
		Command string `json:"command" binding:"required,oneof=lock unlock cut-oil restore-oil"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	commandMap := map[string]map[string]interface{}{
		"lock":        {"type": "door", "action": "lock"},
		"unlock":      {"type": "door", "action": "unlock"},
		"cut-oil":     {"type": "oil", "action": "cut"},
		"restore-oil": {"type": "oil", "action": "restore"},
	}
	
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	
	resp, err := h.commandService.SendCommand(ctx, deviceID, model.CmdVehicleControl, commandMap[req.Command], 30*time.Second)
	if err != nil {
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, resp)
}
