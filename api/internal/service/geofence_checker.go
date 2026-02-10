package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"openfms/api/internal/model"
)

// GeofenceChecker handles geofence detection and alerting
type GeofenceChecker struct {
	db              *gorm.DB
	redis           *redis.Client
	nats            *nats.Conn
	geofenceService *GeofenceService
	webhookService  *WebhookService
	ctx             context.Context
	cancel          context.CancelFunc
}

// LocationMessage represents a location update from NATS
type LocationMessage struct {
	DeviceID  string  `json:"device_id"`
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	Speed     float64 `json:"speed"`
	Direction float64 `json:"direction"`
	Timestamp int64   `json:"timestamp"`
	Status    int     `json:"status,omitempty"`
}

// NewGeofenceChecker creates a new geofence checker
func NewGeofenceChecker(db *gorm.DB, redisClient *redis.Client, natsConn *nats.Conn) *GeofenceChecker {
	ctx, cancel := context.WithCancel(context.Background())
	return &GeofenceChecker{
		db:              db,
		redis:           redisClient,
		nats:            natsConn,
		geofenceService: NewGeofenceService(db, redisClient),
		webhookService:  NewWebhookService(db),
		ctx:             ctx,
		cancel:          cancel,
	}
}

// Start starts the geofence checker
func (c *GeofenceChecker) Start() error {
	log.Println("[GeofenceChecker] Starting...")

	// Subscribe to location updates
	sub, err := c.nats.Subscribe("fms.uplink.LOCATION", func(msg *nats.Msg) {
		var locMsg LocationMessage
		if err := json.Unmarshal(msg.Data, &locMsg); err != nil {
			log.Printf("[GeofenceChecker] Failed to unmarshal location message: %v", err)
			return
		}

		if err := c.processLocationUpdate(&locMsg); err != nil {
			log.Printf("[GeofenceChecker] Failed to process location update: %v", err)
		}
	})

	if err != nil {
		return fmt.Errorf("failed to subscribe to NATS: %v", err)
	}

	log.Println("[GeofenceChecker] Subscribed to location updates")

	// Store subscription for cleanup
	_ = sub

	return nil
}

// Stop stops the geofence checker
func (c *GeofenceChecker) Stop() {
	c.cancel()
	log.Println("[GeofenceChecker] Stopped")
}

// processLocationUpdate processes a location update and checks geofences
func (c *GeofenceChecker) processLocationUpdate(locMsg *LocationMessage) error {
	// Get device ID from string
	deviceIDUint, err := c.getDeviceIDByDeviceID(locMsg.DeviceID)
	if err != nil {
		// Device not found in database, skip
		return nil
	}

	// Get all geofences bound to this device
	geofences, err := c.geofenceService.GetGeofencesByDevice(c.ctx, deviceIDUint)
	if err != nil {
		return fmt.Errorf("failed to get geofences: %v", err)
	}

	if len(geofences) == 0 {
		return nil
	}

	// Check each geofence
	for _, geofence := range geofences {
		if err := c.checkGeofence(deviceIDUint, locMsg, &geofence); err != nil {
			log.Printf("[GeofenceChecker] Error checking geofence %d: %v", geofence.ID, err)
		}
	}

	return nil
}

// checkGeofence checks if a location update triggers a geofence event
func (c *GeofenceChecker) checkGeofence(deviceID uint, locMsg *LocationMessage, geofence *model.Geofence) error {
	// Check if point is inside geofence
	isInside, err := c.geofenceService.CheckPointInGeofence(locMsg.Lat, locMsg.Lon, geofence)
	if err != nil {
		return err
	}

	// Get current state
	state, err := c.geofenceService.GetDeviceGeofenceState(c.ctx, deviceID, geofence.ID)
	if err != nil {
		return err
	}

	// Determine if we need to trigger an event
	var eventType string
	var shouldTrigger bool

	if isInside && !state.IsInside {
		// Device entered the geofence
		eventType = "enter"
		shouldTrigger = geofence.AlertType == "enter" || geofence.AlertType == "both"
	} else if !isInside && state.IsInside {
		// Device exited the geofence
		eventType = "exit"
		shouldTrigger = geofence.AlertType == "exit" || geofence.AlertType == "both"
	}

	// Update state
	if eventType != "" {
		if err := c.geofenceService.UpdateDeviceGeofenceState(c.ctx, deviceID, geofence.ID, isInside, eventType); err != nil {
			return err
		}
	}

	// Trigger event if needed
	if shouldTrigger {
		if err := c.triggerEvent(deviceID, locMsg, geofence, eventType); err != nil {
			return err
		}
	}

	return nil
}

// triggerEvent creates a geofence event and sends alert
func (c *GeofenceChecker) triggerEvent(deviceID uint, locMsg *LocationMessage, geofence *model.Geofence, eventType string) error {
	// Create event record
	event := &model.GeofenceEvent{
		GeofenceID:  geofence.ID,
		DeviceID:    deviceID,
		EventType:   eventType,
		Location:    model.JSONMap{"lat": locMsg.Lat, "lon": locMsg.Lon},
		Speed:       locMsg.Speed,
		TriggeredAt: time.Unix(locMsg.Timestamp, 0),
	}

	if err := c.geofenceService.CreateEvent(c.ctx, event); err != nil {
		log.Printf("[GeofenceChecker] Failed to create event: %v", err)
		// Continue to send alert even if event creation fails
	}

	// Get device info for alert
	var device model.Device
	if err := c.db.First(&device, deviceID).Error; err != nil {
		device.Name = locMsg.DeviceID // Fallback to device ID
	}

	// Create and send alert
	alert := model.GeofenceAlert{
		GeofenceID:   geofence.ID,
		GeofenceName: geofence.Name,
		DeviceID:     deviceID,
		DeviceName:   device.Name,
		EventType:    eventType,
		Location:     model.Location{Lat: locMsg.Lat, Lon: locMsg.Lon},
		Speed:        locMsg.Speed,
		Timestamp:    locMsg.Timestamp,
	}

	alertData, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("failed to marshal alert: %v", err)
	}

	// Publish to NATS
	subject := "fms.alarm.GEOFENCE"
	if err := c.nats.Publish(subject, alertData); err != nil {
		return fmt.Errorf("failed to publish alert: %v", err)
	}

	// Also publish to device-specific subject
	deviceSubject := fmt.Sprintf("fms.alarm.GEOFENCE.%s", locMsg.DeviceID)
	c.nats.Publish(deviceSubject, alertData)

	// Cache alert in Redis for quick lookup
	c.cacheAlert(locMsg.DeviceID, alert)

	// Trigger webhook event
	if c.webhookService != nil {
		c.webhookService.TriggerGeofenceEvent(c.ctx, &alert)
	}

	log.Printf("[GeofenceChecker] Geofence alert: %s %s geofence '%s' (device: %s)",
		eventType, eventType, geofence.Name, device.Name)

	return nil
}

// cacheAlert caches the alert in Redis
func (c *GeofenceChecker) cacheAlert(deviceID string, alert model.GeofenceAlert) {
	key := fmt.Sprintf("fms:geofence:alert:%s", deviceID)
	data, _ := json.Marshal(alert)
	c.redis.Set(c.ctx, key, data, 24*time.Hour) // Keep for 24 hours

	// Also add to recent alerts list
	listKey := "fms:geofence:alerts:recent"
	c.redis.LPush(c.ctx, listKey, data)
	c.redis.LTrim(c.ctx, listKey, 0, 99) // Keep last 100 alerts
}

// getDeviceIDByDeviceID gets the database ID from device_id string
func (c *GeofenceChecker) getDeviceIDByDeviceID(deviceID string) (uint, error) {
	// Try to get from cache first
	cacheKey := fmt.Sprintf("fms:device:id:%s", deviceID)
	cachedID, err := c.redis.Get(c.ctx, cacheKey).Result()
	if err == nil {
		id, _ := strconv.ParseUint(cachedID, 10, 32)
		return uint(id), nil
	}

	// Query database
	var device model.Device
	if err := c.db.Where("device_id = ?", deviceID).First(&device).Error; err != nil {
		return 0, err
	}

	// Cache the result
	c.redis.Set(c.ctx, cacheKey, strconv.FormatUint(uint64(device.ID), 10), 1*time.Hour)

	return device.ID, nil
}

// CheckAllGeofences checks all geofences for a device (useful for initial check)
func (c *GeofenceChecker) CheckAllGeofences(deviceID string, lat, lon float64) error {
	deviceIDUint, err := c.getDeviceIDByDeviceID(deviceID)
	if err != nil {
		return err
	}

	locMsg := &LocationMessage{
		DeviceID:  deviceID,
		Lat:       lat,
		Lon:       lon,
		Timestamp: time.Now().Unix(),
	}

	return c.processLocationUpdate(locMsg)
}
