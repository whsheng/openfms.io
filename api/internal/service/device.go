package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"openfms/api/internal/model"
)

// DeviceService handles device business logic
type DeviceService struct {
	db    *gorm.DB
	redis *redis.Client
	nats  *nats.Conn
}

// NewDeviceService creates a new device service
func NewDeviceService(db *gorm.DB, redisClient *redis.Client, natsConn *nats.Conn) *DeviceService {
	return &DeviceService{
		db:    db,
		redis: redisClient,
		nats:  natsConn,
	}
}

// List returns list of devices
func (s *DeviceService) List(ctx context.Context, page, pageSize int) ([]model.Device, int64, error) {
	var devices []model.Device
	var total int64

	offset := (page - 1) * pageSize

	if err := s.db.Model(&model.Device{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := s.db.Offset(offset).Limit(pageSize).Find(&devices).Error; err != nil {
		return nil, 0, err
	}

	return devices, total, nil
}

// GetByID returns a device by ID
func (s *DeviceService) GetByID(ctx context.Context, id uint) (*model.Device, error) {
	var device model.Device
	if err := s.db.First(&device, id).Error; err != nil {
		return nil, err
	}
	return &device, nil
}

// Create creates a new device
func (s *DeviceService) Create(ctx context.Context, device *model.Device) error {
	return s.db.Create(device).Error
}

// Update updates a device
func (s *DeviceService) Update(ctx context.Context, device *model.Device) error {
	return s.db.Save(device).Error
}

// Delete deletes a device
func (s *DeviceService) Delete(ctx context.Context, id uint) error {
	return s.db.Delete(&model.Device{}, id).Error
}

// GetShadow returns device shadow from Redis
func (s *DeviceService) GetShadow(ctx context.Context, deviceID string) (*model.DeviceShadow, error) {
	key := fmt.Sprintf("fms:shadow:%s", deviceID)
	data, err := s.redis.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("shadow not found")
	}

	shadow := &model.DeviceShadow{DeviceID: deviceID}
	// Parse fields from Redis hash
	// This is simplified - in production, handle type conversions properly
	return shadow, nil
}

// SendCommand sends a command to a device
func (s *DeviceService) SendCommand(ctx context.Context, deviceID, cmdType string, params map[string]interface{}) error {
	// Check if device is online
	sessionKey := fmt.Sprintf("fms:sess:%s", deviceID)
	gatewayInfo, err := s.redis.Get(ctx, sessionKey).Result()
	if err != nil {
		return fmt.Errorf("device not online")
	}

	// Parse gateway ID from session info (format: gateway_id:conn_id:client_ip)
	var gatewayID string
	fmt.Sscanf(gatewayInfo, "%s:", &gatewayID)

	// Publish command to NATS
	cmd := map[string]interface{}{
		"device_id": deviceID,
		"type":      cmdType,
		"params":    params,
	}

	cmdData, _ := json.Marshal(cmd)
	subject := fmt.Sprintf("gateway.downlink.%s", gatewayID)
	return s.nats.Publish(subject, cmdData)
}
