package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"openfms/api/internal/model"
)

// GeofenceService handles geofence business logic
type GeofenceService struct {
	db    *gorm.DB
	redis *redis.Client
}

// NewGeofenceService creates a new geofence service
func NewGeofenceService(db *gorm.DB, redisClient *redis.Client) *GeofenceService {
	return &GeofenceService{
		db:    db,
		redis: redisClient,
	}
}

// Create creates a new geofence
func (s *GeofenceService) Create(ctx context.Context, geofence *model.Geofence) error {
	// Validate coordinates based on type
	if err := s.validateCoordinates(geofence); err != nil {
		return err
	}

	if err := s.db.Create(geofence).Error; err != nil {
		return err
	}

	// Cache geofence in Redis for quick lookup
	s.cacheGeofence(ctx, geofence)

	return nil
}

// GetByID returns a geofence by ID
func (s *GeofenceService) GetByID(ctx context.Context, id uint) (*model.Geofence, error) {
	var geofence model.Geofence
	if err := s.db.Preload("Devices").First(&geofence, id).Error; err != nil {
		return nil, err
	}
	return &geofence, nil
}

// List returns list of geofences with pagination
func (s *GeofenceService) List(ctx context.Context, page, pageSize int) ([]model.Geofence, int64, error) {
	var geofences []model.Geofence
	var total int64

	offset := (page - 1) * pageSize

	if err := s.db.Model(&model.Geofence{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := s.db.Offset(offset).Limit(pageSize).Find(&geofences).Error; err != nil {
		return nil, 0, err
	}

	return geofences, total, nil
}

// Update updates a geofence
func (s *GeofenceService) Update(ctx context.Context, geofence *model.Geofence) error {
	// Validate coordinates based on type
	if err := s.validateCoordinates(geofence); err != nil {
		return err
	}

	if err := s.db.Save(geofence).Error; err != nil {
		return err
	}

	// Update cache
	s.cacheGeofence(ctx, geofence)

	return nil
}

// Delete deletes a geofence
func (s *GeofenceService) Delete(ctx context.Context, id uint) error {
	// Delete from database (cascade will handle associations)
	if err := s.db.Delete(&model.Geofence{}, id).Error; err != nil {
		return err
	}

	// Remove from cache
	s.removeGeofenceFromCache(ctx, id)

	return nil
}

// BindDevices binds devices to a geofence
func (s *GeofenceService) BindDevices(ctx context.Context, geofenceID uint, deviceIDs []uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, deviceID := range deviceIDs {
			association := model.GeofenceDevice{
				GeofenceID: geofenceID,
				DeviceID:   deviceID,
			}
			if err := tx.Create(&association).Error; err != nil {
				// Ignore duplicate key errors
				if err.Error() != "ERROR: duplicate key value violates unique constraint \"idx_geofence_devices_geofence_id_device_id\" (SQLSTATE 23505)" {
					return err
				}
			}
		}
		return nil
	})
}

// UnbindDevices unbinds devices from a geofence
func (s *GeofenceService) UnbindDevices(ctx context.Context, geofenceID uint, deviceIDs []uint) error {
	return s.db.Where("geofence_id = ? AND device_id IN ?", geofenceID, deviceIDs).
		Delete(&model.GeofenceDevice{}).Error
}

// GetDevices returns all devices bound to a geofence
func (s *GeofenceService) GetDevices(ctx context.Context, geofenceID uint) ([]model.Device, error) {
	var devices []model.Device
	err := s.db.Joins("JOIN geofence_devices ON geofence_devices.device_id = devices.id").
		Where("geofence_devices.geofence_id = ?", geofenceID).
		Find(&devices).Error
	return devices, err
}

// GetGeofencesByDevice returns all geofences bound to a device
func (s *GeofenceService) GetGeofencesByDevice(ctx context.Context, deviceID uint) ([]model.Geofence, error) {
	var geofences []model.Geofence
	err := s.db.Joins("JOIN geofence_devices ON geofence_devices.geofence_id = geofences.id").
		Where("geofence_devices.device_id = ? AND geofences.status = 1", deviceID).
		Find(&geofences).Error
	return geofences, err
}

// GetEvents returns geofence events with pagination
func (s *GeofenceService) GetEvents(ctx context.Context, geofenceID uint, page, pageSize int) ([]model.GeofenceEvent, int64, error) {
	var events []model.GeofenceEvent
	var total int64

	offset := (page - 1) * pageSize

	if err := s.db.Model(&model.GeofenceEvent{}).Where("geofence_id = ?", geofenceID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := s.db.Where("geofence_id = ?", geofenceID).
		Order("triggered_at DESC").
		Offset(offset).Limit(pageSize).
		Preload("Device").
		Find(&events).Error; err != nil {
		return nil, 0, err
	}

	return events, total, nil
}

// CheckPointInGeofence checks if a point is inside a geofence
func (s *GeofenceService) CheckPointInGeofence(lat, lon float64, geofence *model.Geofence) (bool, error) {
	switch geofence.Type {
	case "circle":
		return s.checkPointInCircle(lat, lon, geofence.Coordinates)
	case "polygon":
		return s.checkPointInPolygon(lat, lon, geofence.Coordinates)
	default:
		return false, fmt.Errorf("unsupported geofence type: %s", geofence.Type)
	}
}

// checkPointInCircle checks if a point is inside a circular geofence
func (s *GeofenceService) checkPointInCircle(lat, lon float64, coordinates model.JSONMap) (bool, error) {
	coordsJSON, err := json.Marshal(coordinates)
	if err != nil {
		return false, err
	}

	var circleCoords model.CircleGeofenceCoordinates
	if err := json.Unmarshal(coordsJSON, &circleCoords); err != nil {
		return false, err
	}

	distance := calculateDistance(lat, lon, circleCoords.Center.Lat, circleCoords.Center.Lon)
	return distance <= circleCoords.Radius, nil
}

// checkPointInPolygon checks if a point is inside a polygon geofence using ray casting algorithm
func (s *GeofenceService) checkPointInPolygon(lat, lon float64, coordinates model.JSONMap) (bool, error) {
	coordsJSON, err := json.Marshal(coordinates)
	if err != nil {
		return false, err
	}

	var polyCoords model.PolygonGeofenceCoordinates
	if err := json.Unmarshal(coordsJSON, &polyCoords); err != nil {
		return false, err
	}

	points := polyCoords.Points
	if len(points) < 3 {
		return false, fmt.Errorf("polygon must have at least 3 points")
	}

	// Ray casting algorithm
	inside := false
	j := len(points) - 1
	for i := 0; i < len(points); i++ {
		pi := points[i]
		pj := points[j]

		if ((pi.Lon > lon) != (pj.Lon > lon)) &&
			(lat < (pj.Lat-pi.Lat)*(lon-pi.Lon)/(pj.Lon-pi.Lon)+pi.Lat) {
			inside = !inside
		}
		j = i
	}

	return inside, nil
}

// GetDeviceGeofenceState gets the current state of a device relative to a geofence
func (s *GeofenceService) GetDeviceGeofenceState(ctx context.Context, deviceID, geofenceID uint) (*model.DeviceGeofenceState, error) {
	var state model.DeviceGeofenceState
	if err := s.db.Where("device_id = ? AND geofence_id = ?", deviceID, geofenceID).First(&state).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create initial state
			state = model.DeviceGeofenceState{
				DeviceID:   deviceID,
				GeofenceID: geofenceID,
				IsInside:   false,
			}
			if err := s.db.Create(&state).Error; err != nil {
				return nil, err
			}
			return &state, nil
		}
		return nil, err
	}
	return &state, nil
}

// UpdateDeviceGeofenceState updates the state of a device relative to a geofence
func (s *GeofenceService) UpdateDeviceGeofenceState(ctx context.Context, deviceID, geofenceID uint, isInside bool, eventType string) error {
	return s.db.Model(&model.DeviceGeofenceState{}).
		Where("device_id = ? AND geofence_id = ?", deviceID, geofenceID).
		Updates(map[string]interface{}{
			"is_inside":         isInside,
			"last_event_type":   eventType,
			"last_triggered_at": gorm.Expr("NOW()"),
		}).Error
}

// CreateEvent creates a geofence event
func (s *GeofenceService) CreateEvent(ctx context.Context, event *model.GeofenceEvent) error {
	return s.db.Create(event).Error
}

// validateCoordinates validates geofence coordinates based on type
func (s *GeofenceService) validateCoordinates(geofence *model.Geofence) error {
	switch geofence.Type {
	case "circle":
		coordsJSON, err := json.Marshal(geofence.Coordinates)
		if err != nil {
			return err
		}
		var circleCoords model.CircleGeofenceCoordinates
		if err := json.Unmarshal(coordsJSON, &circleCoords); err != nil {
			return fmt.Errorf("invalid circle coordinates: %v", err)
		}
		if circleCoords.Center.Lat < -90 || circleCoords.Center.Lat > 90 {
			return fmt.Errorf("invalid latitude")
		}
		if circleCoords.Center.Lon < -180 || circleCoords.Center.Lon > 180 {
			return fmt.Errorf("invalid longitude")
		}
		if circleCoords.Radius <= 0 {
			return fmt.Errorf("radius must be positive")
		}
	case "polygon":
		coordsJSON, err := json.Marshal(geofence.Coordinates)
		if err != nil {
			return err
		}
		var polyCoords model.PolygonGeofenceCoordinates
		if err := json.Unmarshal(coordsJSON, &polyCoords); err != nil {
			return fmt.Errorf("invalid polygon coordinates: %v", err)
		}
		if len(polyCoords.Points) < 3 {
			return fmt.Errorf("polygon must have at least 3 points")
		}
		for _, p := range polyCoords.Points {
			if p.Lat < -90 || p.Lat > 90 {
				return fmt.Errorf("invalid latitude in polygon")
			}
			if p.Lon < -180 || p.Lon > 180 {
				return fmt.Errorf("invalid longitude in polygon")
			}
		}
	default:
		return fmt.Errorf("unsupported geofence type: %s", geofence.Type)
	}
	return nil
}

// cacheGeofence caches a geofence in Redis
func (s *GeofenceService) cacheGeofence(ctx context.Context, geofence *model.Geofence) {
	key := fmt.Sprintf("fms:geofence:%d", geofence.ID)
	data, _ := json.Marshal(geofence)
	s.redis.Set(ctx, key, data, 0)
}

// removeGeofenceFromCache removes a geofence from Redis cache
func (s *GeofenceService) removeGeofenceFromCache(ctx context.Context, id uint) {
	key := fmt.Sprintf("fms:geofence:%d", id)
	s.redis.Del(ctx, key)
}

// calculateDistance calculates the distance between two points using Haversine formula
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371000 // Earth's radius in meters

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}
