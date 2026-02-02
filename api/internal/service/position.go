package service

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"openfms/api/internal/model"
)

// PositionService handles position business logic
type PositionService struct {
	db    *gorm.DB
	redis *redis.Client
}

// NewPositionService creates a new position service
func NewPositionService(db *gorm.DB, redisClient *redis.Client) *PositionService {
	return &PositionService{
		db:    db,
		redis: redisClient,
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
	return s.db.Create(position).Error
}
