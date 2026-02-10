import React, { useEffect, useRef, useState } from 'react'
import mapboxgl from 'mapbox-gl'
import 'mapbox-gl/dist/mapbox-gl.css'

// Default Mapbox token - should be replaced with your own
const MAPBOX_TOKEN = import.meta.env.VITE_MAPBOX_TOKEN || ''

interface MapPoint {
  lat: number
  lon: number
}

interface GeofenceMapProps {
  type: 'circle' | 'polygon'
  center?: MapPoint
  radius?: number
  polygonPoints?: MapPoint[]
  onMapClick?: (point: MapPoint) => void
  editable?: boolean
  style?: React.CSSProperties
}

const GeofenceMap: React.FC<GeofenceMapProps> = ({
  type,
  center = { lat: 39.9042, lon: 116.4074 },
  radius = 1000,
  polygonPoints = [],
  onMapClick,
  editable = false,
  style = {},
}) => {
  const mapContainer = useRef<HTMLDivElement>(null)
  const map = useRef<mapboxgl.Map | null>(null)
  const markersRef = useRef<mapboxgl.Marker[]>([])
  const circleLayerRef = useRef<string | null>(null)
  const polygonLayerRef = useRef<string | null>(null)
  const [isReady, setIsReady] = useState(false)

  // Initialize map
  useEffect(() => {
    if (!mapContainer.current || map.current) return

    if (!MAPBOX_TOKEN) {
      console.warn('Mapbox token not configured')
      return
    }

    mapboxgl.accessToken = MAPBOX_TOKEN

    const newMap = new mapboxgl.Map({
      container: mapContainer.current,
      style: 'mapbox://styles/mapbox/streets-v12',
      center: [center.lon, center.lat],
      zoom: 13,
    })

    newMap.on('load', () => {
      setIsReady(true)
    })

    // Add click handler
    newMap.on('click', (e) => {
      if (editable && onMapClick) {
        onMapClick({
          lat: e.lngLat.lat,
          lon: e.lngLat.lng,
        })
      }
    })

    // Add navigation controls
    newMap.addControl(new mapboxgl.NavigationControl(), 'top-right')

    map.current = newMap

    return () => {
      newMap.remove()
      map.current = null
    }
  }, [])

  // Update center when changed
  useEffect(() => {
    if (map.current && center) {
      map.current.setCenter([center.lon, center.lat])
    }
  }, [center])

  // Draw circle geofence
  useEffect(() => {
    if (!isReady || !map.current) return

    if (type === 'circle') {
      drawCircle()
    }
  }, [isReady, type, center, radius])

  // Draw polygon geofence
  useEffect(() => {
    if (!isReady || !map.current) return

    if (type === 'polygon') {
      drawPolygon()
    }
  }, [isReady, type, polygonPoints])

  // Clear markers and layers when type changes
  useEffect(() => {
    if (!isReady || !map.current) return

    clearMarkers()
    clearLayers()
  }, [isReady, type])

  const clearMarkers = () => {
    markersRef.current.forEach((marker) => marker.remove())
    markersRef.current = []
  }

  const clearLayers = () => {
    if (!map.current) return

    if (circleLayerRef.current) {
      if (map.current.getLayer(circleLayerRef.current)) {
        map.current.removeLayer(circleLayerRef.current)
      }
      if (map.current.getSource(circleLayerRef.current)) {
        map.current.removeSource(circleLayerRef.current)
      }
      circleLayerRef.current = null
    }

    if (polygonLayerRef.current) {
      if (map.current.getLayer(polygonLayerRef.current)) {
        map.current.removeLayer(polygonLayerRef.current)
      }
      if (map.current.getLayer(polygonLayerRef.current + '-outline')) {
        map.current.removeLayer(polygonLayerRef.current + '-outline')
      }
      if (map.current.getSource(polygonLayerRef.current)) {
        map.current.removeSource(polygonLayerRef.current)
      }
      polygonLayerRef.current = null
    }
  }

  const drawCircle = () => {
    if (!map.current) return

    clearLayers()
    clearMarkers()

    // Add center marker
    const centerMarker = new mapboxgl.Marker({ color: '#1890ff' })
      .setLngLat([center.lon, center.lat])
      .addTo(map.current)
    markersRef.current.push(centerMarker)

    // Generate circle points
    const points = generateCirclePoints(center, radius)
    const sourceId = `circle-source-${Date.now()}`
    const layerId = `circle-layer-${Date.now()}`

    map.current.addSource(sourceId, {
      type: 'geojson',
      data: {
        type: 'Feature',
        geometry: {
          type: 'Polygon',
          coordinates: [points],
        },
        properties: {},
      },
    })

    map.current.addLayer({
      id: layerId,
      type: 'fill',
      source: sourceId,
      paint: {
        'fill-color': '#1890ff',
        'fill-opacity': 0.3,
      },
    })

    // Add outline
    map.current.addLayer({
      id: layerId + '-outline',
      type: 'line',
      source: sourceId,
      paint: {
        'line-color': '#1890ff',
        'line-width': 2,
      },
    })

    circleLayerRef.current = sourceId

    // Fit bounds to show the entire circle
    const bounds = getCircleBounds(center, radius)
    map.current.fitBounds(bounds, { padding: 50 })
  }

  const drawPolygon = () => {
    if (!map.current || polygonPoints.length === 0) return

    clearLayers()
    clearMarkers()

    // Add markers for each point
    polygonPoints.forEach((point, index) => {
      const marker = new mapboxgl.Marker({
        color: index === 0 ? '#52c41a' : '#1890ff',
      })
        .setLngLat([point.lon, point.lat])
        .addTo(map.current!)
      markersRef.current.push(marker)
    })

    if (polygonPoints.length < 3) return

    // Close the polygon
    const coordinates = polygonPoints.map((p) => [p.lon, p.lat])
    coordinates.push([polygonPoints[0].lon, polygonPoints[0].lat])

    const sourceId = `polygon-source-${Date.now()}`
    const layerId = `polygon-layer-${Date.now()}`

    map.current.addSource(sourceId, {
      type: 'geojson',
      data: {
        type: 'Feature',
        geometry: {
          type: 'Polygon',
          coordinates: [coordinates],
        },
        properties: {},
      },
    })

    map.current.addLayer({
      id: layerId,
      type: 'fill',
      source: sourceId,
      paint: {
        'fill-color': '#52c41a',
        'fill-opacity': 0.3,
      },
    })

    map.current.addLayer({
      id: layerId + '-outline',
      type: 'line',
      source: sourceId,
      paint: {
        'line-color': '#52c41a',
        'line-width': 2,
      },
    })

    polygonLayerRef.current = sourceId

    // Fit bounds
    const bounds = getPolygonBounds(polygonPoints)
    map.current.fitBounds(bounds, { padding: 50 })
  }

  const generateCirclePoints = (center: MapPoint, radius: number): number[][] => {
    const points: number[][] = []
    const numPoints = 64
    const earthRadius = 6371000 // meters

    for (let i = 0; i < numPoints; i++) {
      const angle = (i / numPoints) * 2 * Math.PI
      const dx = radius * Math.cos(angle)
      const dy = radius * Math.sin(angle)

      const newLat = center.lat + (dy / earthRadius) * (180 / Math.PI)
      const newLon =
        center.lon +
        (dx / (earthRadius * Math.cos((center.lat * Math.PI) / 180))) *
          (180 / Math.PI)

      points.push([newLon, newLat])
    }
    points.push(points[0]) // Close the polygon

    return points
  }

  const getCircleBounds = (center: MapPoint, radius: number): mapboxgl.LngLatBoundsLike => {
    const earthRadius = 6371000
    const latDelta = (radius / earthRadius) * (180 / Math.PI)
    const lonDelta =
      (radius / (earthRadius * Math.cos((center.lat * Math.PI) / 180))) *
      (180 / Math.PI)

    return [
      [center.lon - lonDelta, center.lat - latDelta],
      [center.lon + lonDelta, center.lat + latDelta],
    ]
  }

  const getPolygonBounds = (points: MapPoint[]): mapboxgl.LngLatBoundsLike => {
    let minLat = points[0].lat
    let maxLat = points[0].lat
    let minLon = points[0].lon
    let maxLon = points[0].lon

    points.forEach((p) => {
      minLat = Math.min(minLat, p.lat)
      maxLat = Math.max(maxLat, p.lat)
      minLon = Math.min(minLon, p.lon)
      maxLon = Math.max(maxLon, p.lon)
    })

    return [
      [minLon, minLat],
      [maxLon, maxLat],
    ]
  }

  if (!MAPBOX_TOKEN) {
    return (
      <div
        style={{
          ...style,
          height: style.height || 300,
          backgroundColor: '#f0f0f0',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          borderRadius: 4,
        }}
      >
        <div style={{ textAlign: 'center', color: '#888' }}>
          <p>地图未配置</p>
          <p style={{ fontSize: 12 }}>请设置 VITE_MAPBOX_TOKEN 环境变量</p>
          {type === 'circle' && center && (
            <div style={{ marginTop: 16, fontSize: 12 }}>
              <p>圆心: ({center.lat.toFixed(6)}, {center.lon.toFixed(6)})</p>
              <p>半径: {radius} 米</p>
            </div>
          )}
          {type === 'polygon' && polygonPoints.length > 0 && (
            <div style={{ marginTop: 16, fontSize: 12 }}>
              <p>多边形顶点数: {polygonPoints.length}</p>
            </div>
          )}
        </div>
      </div>
    )
  }

  return (
    <div
      ref={mapContainer}
      style={{
        ...style,
        width: '100%',
        height: style.height || '100%',
        minHeight: 200,
      }}
    />
  )
}

export default GeofenceMap
