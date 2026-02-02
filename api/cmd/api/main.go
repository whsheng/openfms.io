package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"openfms/api/internal/config"
	"openfms/api/internal/handler"
	"openfms/api/internal/model"
	"openfms/api/internal/service"
)

func main() {
	log.Println("[API] Starting OpenFMS API Server...")

	// Load configuration
	cfg := config.Load()

	// Connect to database
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("[API] Failed to connect to database: %v", err)
	}
	log.Println("[API] Connected to database")

	// Auto migrate
	if err := autoMigrate(db); err != nil {
		log.Fatalf("[API] Failed to migrate database: %v", err)
	}
	log.Println("[API] Database migrated")

	// Connect to Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisURL,
		DB:   0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("[API] Failed to connect to Redis: %v", err)
	}
	log.Println("[API] Connected to Redis")
	defer redisClient.Close()

	// Connect to NATS
	natsConn, err := nats.Connect(cfg.NATSURL)
	if err != nil {
		log.Fatalf("[API] Failed to connect to NATS: %v", err)
	}
	log.Println("[API] Connected to NATS")
	defer natsConn.Close()

	// Initialize services
	authService := service.NewAuthService(db)
	deviceService := service.NewDeviceService(db, redisClient, natsConn)
	positionService := service.NewPositionService(db, redisClient)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService, cfg)
	deviceHandler := handler.NewDeviceHandler(deviceService)
	positionHandler := handler.NewPositionHandler(positionService)

	// Setup Gin router
	router := gin.Default()

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Public routes
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	router.POST("/api/v1/auth/login", authHandler.Login)

	// Protected routes
	api := router.Group("/api/v1")
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

		// Positions
		api.GET("/positions/latest", positionHandler.GetAllLatest)
		api.GET("/devices/:device_id/positions", positionHandler.GetHistory)
		api.GET("/devices/:device_id/positions/latest", positionHandler.GetLatest)
	}

	// Start NATS consumers
	go startNATSConsumers(natsConn, positionService, deviceService)

	// Start HTTP server
	addr := ":3000"
	log.Printf("[API] HTTP server listening on %s", addr)

	go func() {
		if err := router.Run(addr); err != nil {
			log.Fatalf("[API] Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("[API] Shutting down...")
}

func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{},
		&model.Device{},
		&model.Vehicle{},
		&model.Position{},
		&model.Geofence{},
	)
}

func startNATSConsumers(nc *nats.Conn, positionService *service.PositionService, deviceService *service.DeviceService) {
	// Subscribe to location updates
	nc.Subscribe("fms.uplink.LOCATION", func(msg *nats.Msg) {
		// Parse and save position
		log.Printf("[API] Received location update")
		// TODO: Parse message and save to database
	})

	nc.Subscribe("fms.uplink.all", func(msg *nats.Msg) {
		// Log all messages for debugging
		log.Printf("[API] Received message: %s", string(msg.Data))
	})
}
