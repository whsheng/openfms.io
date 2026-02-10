package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"openfms/api/internal/model"
)

// PositionService handles position business logic
type PositionService struct {
	db         *gorm.DB
	redis      *redis.Client
	jetstream  *JetStreamService
}

// NewPositionService creates a new position service
func NewPositionService(db *gorm.DB, redisClient *redis.Client, jetstream *JetStreamService) *PositionService {
	return &PositionService{
		db:        db,
		redis:     redisClient,
		jetstream: jetstream,
	}
}

// GetHistory returns position history for a device
func (s *PositionService) GetHistory(ctx context.Context, deviceID string, start, end time.Time, limit int) ([]model.Position, error) {
	var positions []model.Position

	query := s.db.Where("device_id = ? AND time >= ? AND time <= ?", deviceID, start, end).
		Order("time DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&positions).Error; err != nil {
		return nil, err
	}

	return positions, nil
}

// GetLatest returns latest position for a device
func (s *PositionService) GetLatest(ctx context.Context, deviceID string) (*model.Position, error) {
	var position model.Position

	if err := s.db.Where("device_id = ?", deviceID).
		Order("time DESC").
		First(&position).Error; err != nil {
		return nil, err
	}

	return &position, nil
}

// GetAllLatest returns latest positions for all online devices
func (s *PositionService) GetAllLatest(ctx context.Context) ([]model.Position, error) {
	// Get all online devices from Redis
	deviceKeys, err := s.redis.Keys(ctx, "fms:shadow:*").Result()
	if err != nil {
		return nil, err
	}

	var positions []model.Position

	for _, key := range deviceKeys {
		// Extract device ID from key (fms:shadow:{device_id})
		var deviceID string
		fmt.Sscanf(key, "fms:shadow:%s", &deviceID)

		if deviceID != "" {
			if pos, err := s.GetLatest(ctx, deviceID); err == nil {
				positions = append(positions, *pos)
			}
		}
	}

	return positions, nil
}

// SavePosition saves a position record
func (s *PositionService) SavePosition(ctx context.Context, position *model.Position) error {
	// Save to database
	if err := s.db.Create(position).Error; err != nil {
		return err
	}

	// Publish to JetStream for persistence and replay
	if s.jetstream != nil && s.jetstream.IsEnabled() {
		extras, _ := json.Marshal(position.Extras)
		locMsg := &LocationMessage{
			ID:        fmt.Sprintf("%d", position.ID),
			DeviceID:  position.DeviceID,
			Lat:       position.Lat,
			Lon:       position.Lon,
			Speed:     position.Speed,
			Direction: position.Direction,
			Altitude:  position.Altitude,
			Timestamp: position.Time,
			Extras:    extras,
		}

		if err := s.jetstream.PublishLocation(ctx, locMsg); err != nil {
			// Log error but don't fail the save operation
			fmt.Printf("[PositionService] Failed to publish location to JetStream: %v\n", err)
		}
	}

	return nil
}

// ReplayLocations replays location messages from JetStream within a time range
func (s *PositionService) ReplayLocations(ctx context.Context, deviceID string, start, end time.Time, batchSize int) ([]*LocationMessage, bool, error) {
	if s.jetstream == nil || !s.jetstream.IsEnabled() {
		return nil, false, fmt.Errorf("JetStream is not enabled")
	}

	return s.jetstream.ReplayLocations(ctx, deviceID, start, end, batchSize)
}

// GetLocationStats returns statistics about the location stream
func (s *PositionService) GetLocationStats() (map[string]interface{}, error) {
	if s.jetstream == nil || !s.jetstream.IsEnabled() {
		return map[string]interface{}{
			"enabled": false,
		}, nil
	}

	info, err := s.jetstream.GetStreamInfo(StreamLocations)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"enabled":     true,
		"stream":      info.Config.Name,
		"subjects":    info.Config.Subjects,
		"state":       info.State,
		"created":     info.Created,
		"max_age":     info.Config.MaxAge,
		"max_bytes":   info.Config.MaxBytes,
		"storage":     info.Config.Storage,
		"replicas":    info.Config.Replicas,
	}, nil
}
