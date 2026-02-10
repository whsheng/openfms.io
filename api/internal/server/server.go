package server

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"openfms/api/internal/config"
	"openfms/api/internal/handler"
	"openfms/api/internal/service"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// GeofenceChecker interface for the geofence checker service
type GeofenceCheckerInterface interface {
	Start() error
	Stop()
}

// AlarmServiceInterface interface for the alarm service
type AlarmServiceInterface interface {
	Start() error
	Stop()
}

// Server represents the HTTP server
type Server struct {
	router          *gin.Engine
	config          *config.Config
	db              *gorm.DB
	redis           *redis.Client
	nats            *nats.Conn
	jetstream       *service.JetStreamService
	wsHub           *handler.WSHub
	wsHandler       *handler.WSHandler
	geofenceChecker GeofenceCheckerInterface
	alarmService    AlarmServiceInterface
}

// NewServer creates a new server instance
func NewServer(cfg *config.Config, db *gorm.DB, redisClient *redis.Client, natsConn *nats.Conn, jetstream *service.JetStreamService) *Server {
	return &Server{
		config:    cfg,
		db:        db,
		redis:     redisClient,
		nats:      natsConn,
		jetstream: jetstream,
	}
}

// Setup initializes routes and handlers
func (s *Server) Setup() {
	// Initialize WebSocket hub first (needed by alarm service)
	s.wsHub = handler.NewWSHub(s.nats)
	s.wsHandler = handler.NewWSHandler(s.wsHub)

	// Initialize services
	authService := service.NewAuthService(s.db)
	deviceService := service.NewDeviceService(s.db, s.redis, s.nats)
	deviceImportService := service.NewDeviceImportService(s.db, deviceService)
	positionService := service.NewPositionService(s.db, s.redis, s.jetstream)
	geofenceService := service.NewGeofenceService(s.db, s.redis)
	alarmService := service.NewAlarmService(s.db, s.nats, s.wsHub, s.jetstream)
	webhookService := service.NewWebhookService(s.db)
	s.alarmService = alarmService

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService, s.config)
	deviceHandler := handler.NewDeviceHandler(deviceService)
	deviceHandler.SetDeviceImportService(deviceImportService)
	positionHandler := handler.NewPositionHandler(positionService)
	geofenceHandler := handler.NewGeofenceHandler(geofenceService)
	alarmHandler := handler.NewAlarmHandler(s.db, alarmService)
	webhookHandler := handler.NewWebhookHandler(s.db, webhookService)

	// Start WebSocket hub in background
	go s.wsHub.Run()
	log.Println("[Server] WebSocket hub started")

	// Setup Gin router
	s.router = gin.Default()

	// CORS middleware
	s.router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Swagger UI
	s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Public routes
	s.router.GET("/health", func(c *gin.Context) {
		health := gin.H{"status": "ok"}
		
		// Add JetStream status if enabled
		if s.jetstream != nil && s.jetstream.IsEnabled() {
			health["jetstream"] = "enabled"
			
			// Get stream info
			if locInfo, err := s.jetstream.GetStreamInfo(service.StreamLocations); err == nil {
				health["jetstream_locations"] = gin.H{
					"messages": locInfo.State.Msgs,
					"bytes":    locInfo.State.Bytes,
				}
			}
			if alarmInfo, err := s.jetstream.GetStreamInfo(service.StreamAlarms); err == nil {
				health["jetstream_alarms"] = gin.H{
					"messages": alarmInfo.State.Msgs,
					"bytes":    alarmInfo.State.Bytes,
				}
			}
		} else {
			health["jetstream"] = "disabled"
		}
		
		c.JSON(200, health)
	})
	s.router.POST("/api/v1/auth/login", authHandler.Login)

	// WebSocket routes - public but can add auth middleware if needed
	s.router.GET("/ws/location", s.wsHandler.HandleLocation)
	s.router.GET("/ws/stats", s.wsHandler.GetStats)

	// Protected routes
	api := s.router.Group("/api/v1")
	api.Use(authHandler.AuthMiddleware())
	{
		// Auth
		api.GET("/auth/me", authHandler.GetMe)

		// Devices
		api.GET("/devices", deviceHandler.List)
		api.POST("/devices", deviceHandler.Create)
		api.GET("/devices/:id", deviceHandler.Get)
		api.PUT("/devices/:id", deviceHandler.Update)
		api.DELETE("/devices/:id", deviceHandler.Delete)
		api.GET("/devices/:device_id/shadow", deviceHandler.GetShadow)
		api.POST("/devices/:device_id/commands", deviceHandler.SendCommand)

		// Device Import
		api.GET("/devices/import-template", deviceHandler.DownloadImportTemplate)
		api.POST("/devices/import-preview", deviceHandler.PreviewImport)
		api.POST("/devices/import", deviceHandler.ImportDevices)
		api.GET("/devices/import/:task_id/status", deviceHandler.GetImportStatus)
		api.GET("/devices/import/:task_id/errors", deviceHandler.DownloadImportErrorReport)

		// Positions
		api.GET("/positions/latest", positionHandler.GetAllLatest)
		api.GET("/devices/:device_id/positions", positionHandler.GetHistory)
		api.GET("/devices/:device_id/positions/latest", positionHandler.GetLatest)

		// Track processing
		api.GET("/devices/:id/track/correct", positionHandler.GetCorrectedTrack)
		api.GET("/devices/:id/track/simplify", positionHandler.GetSimplifiedTrack)

		// Geofences
		api.GET("/geofences", geofenceHandler.List)
		api.POST("/geofences", geofenceHandler.Create)
		api.GET("/geofences/:id", geofenceHandler.Get)
		api.PUT("/geofences/:id", geofenceHandler.Update)
		api.DELETE("/geofences/:id", geofenceHandler.Delete)
		api.POST("/geofences/:id/bind", geofenceHandler.BindDevices)
		api.POST("/geofences/:id/unbind", geofenceHandler.UnbindDevices)
		api.GET("/geofences/:id/devices", geofenceHandler.GetDevices)
		api.GET("/geofences/:id/events", geofenceHandler.GetEvents)
		api.POST("/geofences/:id/check", geofenceHandler.CheckLocation)

		// Alarms
		api.GET("/alarms", alarmHandler.ListAlarms)
		api.GET("/alarms/stats", alarmHandler.GetStats)
		api.GET("/alarms/unread-count", alarmHandler.GetUnreadCount)
		api.GET("/alarms/types", alarmHandler.GetAlarmTypes)
		api.GET("/alarms/:id", alarmHandler.GetAlarm)
		api.POST("/alarms/:id/read", alarmHandler.MarkAsRead)
		api.POST("/alarms/:id/resolve", alarmHandler.ResolveAlarm)
		api.POST("/alarms/batch-read", alarmHandler.BatchRead)
		api.POST("/alarms/batch-resolve", alarmHandler.BatchResolve)
		api.GET("/alarms/rules", alarmHandler.ListRules)
		api.GET("/alarms/rules/:id", alarmHandler.GetRule)
		api.PUT("/alarms/rules/:id", alarmHandler.UpdateRule)
		api.POST("/alarms/rules/:id/toggle", alarmHandler.ToggleRule)

		// JetStream Replay API
		s.registerJetStreamRoutes(api)

		// RBAC - Users & Roles
		rbacHandler := handler.NewRBACHandler(s.db)
		rbacHandler.RegisterRoutes(api)

		// Webhooks
		webhookHandler.RegisterRoutes(api)
	}
}

// registerJetStreamRoutes registers JetStream related routes
func (s *Server) registerJetStreamRoutes(api *gin.RouterGroup) {
	// Location replay
	api.POST("/jetstream/locations/replay", func(c *gin.Context) {
		if s.jetstream == nil || !s.jetstream.IsEnabled() {
			c.JSON(503, gin.H{"error": "JetStream is not enabled"})
			return
		}

		var req struct {
			DeviceID  string    `json:"device_id"`
			StartTime time.Time `json:"start_time" binding:"required"`
			EndTime   time.Time `json:"end_time" binding:"required"`
			BatchSize int       `json:"batch_size"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		if req.BatchSize <= 0 {
			req.BatchSize = 100
		}

		locations, hasMore, err := s.jetstream.ReplayLocations(c.Request.Context(), req.DeviceID, req.StartTime, req.EndTime, req.BatchSize)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"locations": locations,
			"count":     len(locations),
			"has_more":  hasMore,
		})
	})

	// Alarm replay
	api.POST("/jetstream/alarms/replay", func(c *gin.Context) {
		if s.jetstream == nil || !s.jetstream.IsEnabled() {
			c.JSON(503, gin.H{"error": "JetStream is not enabled"})
			return
		}

		var req struct {
			DeviceID  string    `json:"device_id"`
			StartTime time.Time `json:"start_time" binding:"required"`
			EndTime   time.Time `json:"end_time" binding:"required"`
			BatchSize int       `json:"batch_size"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		if req.BatchSize <= 0 {
			req.BatchSize = 100
		}

		alarms, hasMore, err := s.jetstream.ReplayAlarms(c.Request.Context(), req.DeviceID, req.StartTime, req.EndTime, req.BatchSize)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"alarms":   alarms,
			"count":    len(alarms),
			"has_more": hasMore,
		})
	})

	// Stream info
	api.GET("/jetstream/streams/:name", func(c *gin.Context) {
		if s.jetstream == nil || !s.jetstream.IsEnabled() {
			c.JSON(503, gin.H{"error": "JetStream is not enabled"})
			return
		}

		streamName := c.Param("name")
		info, err := s.jetstream.GetStreamInfo(streamName)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"name":        info.Config.Name,
			"subjects":    info.Config.Subjects,
			"state":       info.State,
			"created":     info.Created,
			"max_age":     info.Config.MaxAge,
			"max_bytes":   info.Config.MaxBytes,
			"storage":     info.Config.Storage,
			"replicas":    info.Config.Replicas,
		})
	})
}

// SetGeofenceChecker sets the geofence checker
func (s *Server) SetGeofenceChecker(checker GeofenceCheckerInterface) {
	s.geofenceChecker = checker
}

// SetAlarmService sets the alarm service
func (s *Server) SetAlarmService(service AlarmServiceInterface) {
	s.alarmService = service
}

// Run starts the HTTP server
func (s *Server) Run(addr string) error {
	log.Printf("[Server] HTTP server listening on %s", addr)
	return s.router.Run(addr)
}

// GetWSHub returns the WebSocket hub for external use
func (s *Server) GetWSHub() *handler.WSHub {
	return s.wsHub
}

// GetRouter returns the gin router for testing
func (s *Server) GetRouter() *gin.Engine {
	return s.router
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() {
	if s.wsHub != nil {
		s.wsHub.Stop()
		log.Println("[Server] WebSocket hub stopped")
	}
	if s.geofenceChecker != nil {
		s.geofenceChecker.Stop()
		log.Println("[Server] Geofence checker stopped")
	}
	if s.alarmService != nil {
		s.alarmService.Stop()
		log.Println("[Server] Alarm service stopped")
	}
	if s.jetstream != nil {
		s.jetstream.Close()
		log.Println("[Server] JetStream service stopped")
	}
}
