package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"openfms/api/internal/config"
	"openfms/api/internal/model"
	"openfms/api/internal/server"
	"openfms/api/internal/service"

	_ "openfms/api/docs"
)

// @title OpenFMS API
// @version 1.0
// @description OpenFMS - Open Fleet Management System API
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url https://github.com/openfms/openfms/issues
// @contact.email support@openfms.local

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:3000
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

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

	// Create and setup server
	srv := server.NewServer(cfg, db, redisClient, natsConn)
	srv.Setup()

	// Start NATS consumers for non-WS messages
	go startNATSConsumers(natsConn)

	// Start geofence checker
	geofenceChecker := service.NewGeofenceChecker(db, redisClient, natsConn)
	if err := geofenceChecker.Start(); err != nil {
		log.Fatalf("[API] Failed to start geofence checker: %v", err)
	}
	srv.SetGeofenceChecker(geofenceChecker)
	log.Println("[API] Geofence checker started")

	// Start alarm service
	// Note: Get WSHub from server to pass to alarm service
	alarmService := service.NewAlarmService(db, natsConn, srv.GetWSHub())
	if err := alarmService.Start(); err != nil {
		log.Fatalf("[API] Failed to start alarm service: %v", err)
	}
	srv.SetAlarmService(alarmService)
	log.Println("[API] Alarm service started")

	// Start HTTP server
	addr := ":3000"
	go func() {
		if err := srv.Run(addr); err != nil {
			log.Fatalf("[API] Failed to start server: %v", err)
		}
	}()

	log.Printf("[API] Server ready on %s", addr)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("[API] Shutting down...")

	// Graceful shutdown
	srv.Shutdown()
	log.Println("[API] Server stopped")
}

func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{},
		&model.Device{},
		&model.Vehicle{},
		&model.Position{},
		&model.Geofence{},
		&model.GeofenceDevice{},
		&model.GeofenceEvent{},
		&model.DeviceGeofenceState{},
		&model.Alarm{},
		&model.AlarmRule{},
		&model.Role{},
		&model.Permission{},
		&model.RolePermission{},
		&model.UserRole{},
	)
}

func startNATSConsumers(nc *nats.Conn) {
	// Subscribe to all uplink messages for logging/debugging
	nc.Subscribe("fms.uplink.all", func(msg *nats.Msg) {
		log.Printf("[NATS] Received message: %s", string(msg.Data))
	})

	// Note: Location messages are now handled by WebSocket hub
	// Additional consumers can be added here for other message types
	log.Println("[NATS] Consumers started")
}
