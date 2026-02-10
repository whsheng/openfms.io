package service

import (
	"math"
	"sort"
	"time"

	"openfms/api/internal/model"
)

// TrackPoint represents a point in a trajectory
type TrackPoint struct {
	Lat       float64   `json:"lat"`
	Lon       float64   `json:"lon"`
	Speed     float64   `json:"speed"`
	Angle     float64   `json:"angle"`
	Timestamp time.Time `json:"timestamp"`
}

// TrackProcessor handles trajectory correction and simplification
type TrackProcessor struct {
	// MaxSpeed is the maximum allowed speed (km/h) for filtering outliers
	MaxSpeed float64
	// MaxAngleChange is the maximum allowed angle change (degrees) between consecutive points
	MaxAngleChange float64
	// MinDistance is the minimum distance (meters) between points
	MinDistance float64
	// SimplificationEpsilon is the epsilon value for Douglas-Peucker algorithm (meters)
	SimplificationEpsilon float64
}

// NewTrackProcessor creates a new track processor with default values
func NewTrackProcessor() *TrackProcessor {
	return &TrackProcessor{
		MaxSpeed:              200.0,  // 200 km/h max speed
		MaxAngleChange:        120.0,  // 120 degrees max angle change
		MinDistance:           5.0,    // 5 meters min distance
		SimplificationEpsilon: 10.0,   // 10 meters default epsilon
	}
}

// PositionsToTrackPoints converts model.Position slice to TrackPoint slice
func PositionsToTrackPoints(positions []model.Position) []TrackPoint {
	points := make([]TrackPoint, len(positions))
	for i, p := range positions {
		points[i] = TrackPoint{
			Lat:       p.Lat,
			Lon:       p.Lon,
			Speed:     float64(p.Speed),
			Angle:     float64(p.Angle),
			Timestamp: p.Time,
		}
	}
	return points
}

// TrackPointsToPositions converts TrackPoint slice back to model.Position slice
// Note: This is a simplified conversion, some fields may be lost
func TrackPointsToPositions(points []TrackPoint, deviceID string) []model.Position {
	positions := make([]model.Position, len(points))
	for i, p := range points {
		positions[i] = model.Position{
			Time:     p.Timestamp,
			DeviceID: deviceID,
			Lat:      p.Lat,
			Lon:      p.Lon,
			Speed:    int16(p.Speed),
			Angle:    int16(p.Angle),
		}
	}
	return positions
}

// CorrectTrack performs trajectory correction including:
// 1. Sort by timestamp
// 2. Remove duplicate points
// 3. Filter speed outliers
// 4. Filter angle outliers (sudden direction changes)
// 5. Filter distance outliers (unrealistic jumps)
func (tp *TrackProcessor) CorrectTrack(points []TrackPoint) []TrackPoint {
	if len(points) < 2 {
		return points
	}

	// Sort by timestamp
	sort.Slice(points, func(i, j int) bool {
		return points[i].Timestamp.Before(points[j].Timestamp)
	})

	// Remove duplicates and apply initial filtering
	filtered := tp.removeDuplicates(points)

	// Filter by speed outliers
	filtered = tp.filterSpeedOutliers(filtered)

	// Filter by angle outliers
	filtered = tp.filterAngleOutliers(filtered)

	// Filter by distance outliers (unrealistic jumps)
	filtered = tp.filterDistanceOutliers(filtered)

	return filtered
}

// removeDuplicates removes points with same timestamp or very close coordinates
func (tp *TrackProcessor) removeDuplicates(points []TrackPoint) []TrackPoint {
	if len(points) < 2 {
		return points
	}

	result := make([]TrackPoint, 0, len(points))
	result = append(result, points[0])

	for i := 1; i < len(points); i++ {
		prev := result[len(result)-1]
		curr := points[i]

		// Skip if same timestamp
		if curr.Timestamp.Equal(prev.Timestamp) {
			continue
		}

		// Skip if too close (less than MinDistance)
		dist := haversineDistance(prev.Lat, prev.Lon, curr.Lat, curr.Lon)
		if dist < tp.MinDistance {
			continue
		}

		result = append(result, curr)
	}

	return result
}

// filterSpeedOutliers removes points with unrealistic speeds
func (tp *TrackProcessor) filterSpeedOutliers(points []TrackPoint) []TrackPoint {
	if len(points) < 2 {
		return points
	}

	result := make([]TrackPoint, 0, len(points))
	result = append(result, points[0])

	for i := 1; i < len(points); i++ {
		prev := result[len(result)-1]
		curr := points[i]

		// Calculate speed between points
		dist := haversineDistance(prev.Lat, prev.Lon, curr.Lat, curr.Lon)
		timeDiff := curr.Timestamp.Sub(prev.Timestamp).Hours()

		if timeDiff > 0 {
			speed := dist / timeDiff // km/h
			if speed > tp.MaxSpeed {
				// This point is an outlier, skip it
				continue
			}
		}

		result = append(result, curr)
	}

	return result
}

// filterAngleOutliers removes points with unrealistic direction changes
func (tp *TrackProcessor) filterAngleOutliers(points []TrackPoint) []TrackPoint {
	if len(points) < 3 {
		return points
	}

	result := make([]TrackPoint, 0, len(points))
	result = append(result, points[0])
	result = append(result, points[1])

	for i := 2; i < len(points); i++ {
		p1 := result[len(result)-2]
		p2 := result[len(result)-1]
		p3 := points[i]

		// Calculate angle change
		angle1 := calculateBearing(p1.Lat, p1.Lon, p2.Lat, p2.Lon)
		angle2 := calculateBearing(p2.Lat, p2.Lon, p3.Lat, p3.Lon)

		angleChange := math.Abs(angle2 - angle1)
		if angleChange > 180 {
			angleChange = 360 - angleChange
		}

		if angleChange > tp.MaxAngleChange {
			// Check if this is a temporary outlier by looking at the overall trend
			if i < len(points)-1 {
				p4 := points[i+1]
				angle3 := calculateBearing(p3.Lat, p3.Lon, p4.Lat, p4.Lon)
				angleChange2 := math.Abs(angle3 - angle1)
				if angleChange2 > 180 {
					angleChange2 = 360 - angleChange2
				}
				// If the point after also shows large angle change, p3 might be valid
				if angleChange2 < tp.MaxAngleChange {
					// p3 is likely an outlier, skip it
					continue
				}
			}
		}

		result = append(result, p3)
	}

	return result
}

// filterDistanceOutliers removes points that are too far from the expected path
func (tp *TrackProcessor) filterDistanceOutliers(points []TrackPoint) []TrackPoint {
	if len(points) < 3 {
		return points
	}

	result := make([]TrackPoint, 0, len(points))
	result = append(result, points[0])

	for i := 1; i < len(points)-1; i++ {
		prev := result[len(result)-1]
		curr := points[i]
		next := points[i+1]

		// Calculate expected position based on linear interpolation
		timeRatio := curr.Timestamp.Sub(prev.Timestamp).Seconds() /
			next.Timestamp.Sub(prev.Timestamp).Seconds()

		if timeRatio > 0 && timeRatio < 1 {
			expectedLat := prev.Lat + (next.Lat-prev.Lat)*timeRatio
			expectedLon := prev.Lon + (next.Lon-prev.Lon)*timeRatio

			// Calculate deviation from expected path
			deviation := haversineDistance(curr.Lat, curr.Lon, expectedLat, expectedLon)

			// If deviation is too large, this point might be an outlier
			// Use 5x the simplification epsilon as threshold
			if deviation > tp.SimplificationEpsilon*5 {
				continue
			}
		}

		result = append(result, curr)
	}

	// Always add the last point
	result = append(result, points[len(points)-1])

	return result
}

// SimplifyTrack uses Douglas-Peucker algorithm to simplify the trajectory
func (tp *TrackProcessor) SimplifyTrack(points []TrackPoint) []TrackPoint {
	if len(points) <= 2 {
		return points
	}

	// Use Douglas-Peucker algorithm
	return tp.douglasPeucker(points, 0, len(points)-1, tp.SimplificationEpsilon)
}

// douglasPeucker implements the Douglas-Peucker algorithm recursively
func (tp *TrackProcessor) douglasPeucker(points []TrackPoint, start, end int, epsilon float64) []TrackPoint {
	if start >= end {
		return []TrackPoint{points[start]}
	}

	// Find the point with maximum distance from line between start and end
	maxDist := 0.0
	maxIndex := start

	startPoint := points[start]
	endPoint := points[end]

	for i := start + 1; i < end; i++ {
		dist := perpendicularDistance(points[i], startPoint, endPoint)
		if dist > maxDist {
			maxDist = dist
			maxIndex = i
		}
	}

	// If max distance is greater than epsilon, recursively simplify
	if maxDist > epsilon {
		// Recursive call
		left := tp.douglasPeucker(points, start, maxIndex, epsilon)
		right := tp.douglasPeucker(points, maxIndex, end, epsilon)

		// Combine results (avoid duplicate point at maxIndex)
		return append(left[:len(left)-1], right...)
	}

	// All points between start and end are within epsilon, return only endpoints
	return []TrackPoint{startPoint, endPoint}
}

// SimplifyTrackWithRate simplifies track to target number of points
func (tp *TrackProcessor) SimplifyTrackWithRate(points []TrackPoint, targetCount int) []TrackPoint {
	if len(points) <= targetCount || targetCount < 2 {
		return points
	}

	// Binary search for the right epsilon value
	low, high := 0.0, 10000.0
	var result []TrackPoint

	for i := 0; i < 20; i++ { // Max 20 iterations
		mid := (low + high) / 2
		result = tp.douglasPeucker(points, 0, len(points)-1, mid)

		if len(result) > targetCount {
			low = mid
		} else if len(result) < targetCount {
			high = mid
		} else {
			break
		}
	}

	return result
}

// haversineDistance calculates the great circle distance between two points in kilometers
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth's radius in kilometers

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

// calculateBearing calculates the initial bearing between two points in degrees
func calculateBearing(lat1, lon1, lat2, lon2 float64) float64 {
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	y := math.Sin(deltaLon) * math.Cos(lat2Rad)
	x := math.Cos(lat1Rad)*math.Sin(lat2Rad) -
		math.Sin(lat1Rad)*math.Cos(lat2Rad)*math.Cos(deltaLon)

	bearing := math.Atan2(y, x) * 180 / math.Pi
	if bearing < 0 {
		bearing += 360
	}
	return bearing
}

// perpendicularDistance calculates the perpendicular distance from point p to line ab
func perpendicularDistance(p, a, b TrackPoint) float64 {
	// Convert to meters for accurate distance calculation
	// Using equirectangular approximation for small distances

	latAvg := (a.Lat + b.Lat) / 2 * math.Pi / 180
	x1 := a.Lon * math.Pi / 180 * math.Cos(latAvg) * 6371000
	y1 := a.Lat * math.Pi / 180 * 6371000
	x2 := b.Lon * math.Pi / 180 * math.Cos(latAvg) * 6371000
	y2 := b.Lat * math.Pi / 180 * 6371000
	x0 := p.Lon * math.Pi / 180 * math.Cos(latAvg) * 6371000
	y0 := p.Lat * math.Pi / 180 * 6371000

	// Calculate perpendicular distance
	numerator := math.Abs((y2-y1)*x0 - (x2-x1)*y0 + x2*y1 - y2*x1)
	denominator := math.Sqrt((y2-y1)*(y2-y1) + (x2-x1)*(x2-x1))

	if denominator == 0 {
		return math.Sqrt((x0-x1)*(x0-x1) + (y0-y1)*(y0-y1))
	}

	return numerator / denominator
}

// TrackStats holds statistics about a track
type TrackStats struct {
	TotalPoints     int       `json:"total_points"`
	TotalDistance   float64   `json:"total_distance_km"`
	Duration        float64   `json:"duration_hours"`
	AverageSpeed    float64   `json:"average_speed_kmh"`
	MaxSpeed        float64   `json:"max_speed_kmh"`
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
	StartLat        float64   `json:"start_lat"`
	StartLon        float64   `json:"start_lon"`
	EndLat          float64   `json:"end_lat"`
	EndLon          float64   `json:"end_lon"`
}

// CalculateStats calculates statistics for a track
func (tp *TrackProcessor) CalculateStats(points []TrackPoint) TrackStats {
	if len(points) == 0 {
		return TrackStats{}
	}

	stats := TrackStats{
		TotalPoints: len(points),
		StartTime:   points[0].Timestamp,
		EndTime:     points[len(points)-1].Timestamp,
		StartLat:    points[0].Lat,
		StartLon:    points[0].Lon,
		EndLat:      points[len(points)-1].Lat,
		EndLon:      points[len(points)-1].Lon,
	}

	stats.Duration = stats.EndTime.Sub(stats.StartTime).Hours()

	var totalSpeed float64
	stats.MaxSpeed = -1

	for i := 0; i < len(points); i++ {
		// Update max speed
		if points[i].Speed > stats.MaxSpeed {
			stats.MaxSpeed = points[i].Speed
		}
		totalSpeed += points[i].Speed

		// Calculate distance
		if i > 0 {
			dist := haversineDistance(points[i-1].Lat, points[i-1].Lon,
				points[i].Lat, points[i].Lon)
			stats.TotalDistance += dist
		}
	}

	if len(points) > 0 {
		stats.AverageSpeed = totalSpeed / float64(len(points))
	}

	return stats
}
