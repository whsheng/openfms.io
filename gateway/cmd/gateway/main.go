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

	"openfms/gateway/internal/config"
	"openfms/gateway/internal/server"
)

func main() {
	log.Println("[Gateway] Starting OpenFMS Gateway...")

	// Load configuration
	cfg := config.Load()
	log.Printf("[Gateway] Configuration loaded: ID=%s, Port=%d", cfg.GatewayID, cfg.GatewayPort)

	// Connect to Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisURL,
		DB:   0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("[Gateway] Failed to connect to Redis: %v", err)
	}
	log.Println("[Gateway] Connected to Redis")
	defer redisClient.Close()

	// Connect to NATS
	natsConn, err := nats.Connect(cfg.NATSURL)
	if err != nil {
		log.Fatalf("[Gateway] Failed to connect to NATS: %v", err)
	}
	log.Println("[Gateway] Connected to NATS")
	defer natsConn.Close()

	// Create and start TCP server
	tcpServer := server.NewTCPServer(cfg, redisClient, natsConn)
	if err := tcpServer.Start(); err != nil {
		log.Fatalf("[Gateway] Failed to start TCP server: %v", err)
	}

	log.Println("[Gateway] Server started successfully")
	log.Printf("[Gateway] Listening on TCP port %d", cfg.GatewayPort)
	log.Printf("[Gateway] HTTP API on port %d", cfg.HTTPPort)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("[Gateway] Shutting down...")

	tcpServer.Stop()
	log.Println("[Gateway] Server stopped")
}
