import React, { useEffect, useRef, useState, useCallback, forwardRef, useImperativeHandle } from 'react';
import mapboxgl from 'mapbox-gl';
import { Spin, Alert } from 'antd';
import { useMapbox } from './useMapbox';
import { VehicleData, useVehicleMarkers } from './VehicleMarker';
import type { LngLatBoundsLike } from 'mapbox-gl';

// 样式
const mapContainerStyle: React.CSSProperties = {
  width: '100%',
  height: '100%',
  position: 'relative',
  overflow: 'hidden',
};

const loadingStyle: React.CSSProperties = {
  position: 'absolute',
  top: 0,
  left: 0,
  right: 0,
  bottom: 0,
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  backgroundColor: 'rgba(255, 255, 255, 0.8)',
  zIndex: 10,
};

const errorStyle: React.CSSProperties = {
  position: 'absolute',
  top: 16,
  left: 16,
  right: 16,
  zIndex: 10,
};

// 地图引用类型
export interface MapboxMapRef {
  getMap: () => mapboxgl.Map | null;
  setCenter: (center: [number, number], zoom?: number) => void;
  fitBounds: (bounds: LngLatBoundsLike, options?: mapboxgl.FitBoundsOptions) => void;
  flyToVehicle: (vehicle: VehicleData, zoom?: number) => void;
}

// 地图组件属性
export interface MapboxMapProps {
  vehicles?: VehicleData[];
  selectedVehicleId?: string | null;
  onVehicleClick?: (vehicle: VehicleData) => void;
  onMapLoad?: (map: mapboxgl.Map) => void;
  onMapError?: (error: Error) => void;
  initialCenter?: [number, number];
  initialZoom?: number;
  className?: string;
  style?: React.CSSProperties;
  showTraffic?: boolean;
  showSatellite?: boolean;
}

export const MapboxMap = forwardRef<MapboxMapRef, MapboxMapProps>((
  {
    vehicles = [],
    selectedVehicleId = null,
    onVehicleClick,
    onMapLoad,
    onMapError,
    initialCenter,
    initialZoom,
    className,
    style,
    showTraffic = false,
    showSatellite = false,
  },
  ref
) => {
  const {
    mapRef,
    containerRef,
    isLoaded,
    error,
    setCenter,
    fitBounds,
    getMap,
  } = useMapbox({
    initialCenter,
    initialZoom,
    onMapLoad,
    onMapError,
  });

  const { updateMarkers, clearMarkers, setSelectedVehicle } = useVehicleMarkers(
    mapRef.current
  );

  // 暴露方法给父组件
  useImperativeHandle(ref, () => ({
    getMap,
    setCenter,
    fitBounds,
    flyToVehicle: (vehicle: VehicleData, zoom = 16) => {
      setCenter([vehicle.lon, vehicle.lat], zoom);
    },
  }));

  // 更新车辆标记
  useEffect(() => {
    if (isLoaded && mapRef.current) {
      updateMarkers(vehicles, onVehicleClick);
    }
  }, [vehicles, isLoaded, updateMarkers, onVehicleClick]);

  // 更新选中状态
  useEffect(() => {
    setSelectedVehicle(selectedVehicleId);
  }, [selectedVehicleId, setSelectedVehicle]);

  // 清理标记
  useEffect(() => {
    return () => {
      clearMarkers();
    };
  }, [clearMarkers]);

  // 切换交通图层
  useEffect(() => {
    const map = mapRef.current;
    if (!map || !isLoaded) return;

    const trafficLayerId = 'traffic-layer';
    
    if (showTraffic) {
      if (!map.getLayer(trafficLayerId)) {
        map.addLayer({
          id: trafficLayerId,
          type: 'line',
          source: {
            type: 'vector',
            url: 'mapbox://mapbox.mapbox-traffic-v1',
          },
          'source-layer': 'traffic',
          paint: {
            'line-width': 2,
            'line-color': [
              'case',
              ['==', 'low', ['get', 'congestion']], '#00ff00',
              ['==', 'moderate', ['get', 'congestion']], '#ffff00',
              ['==', 'heavy', ['get', 'congestion']], '#ff0000',
              ['==', 'severe', ['get', 'congestion']], '#800000',
              '#000000'
            ],
          },
        });
      }
    } else {
      if (map.getLayer(trafficLayerId)) {
        map.removeLayer(trafficLayerId);
      }
    }
  }, [showTraffic, isLoaded, mapRef]);

  // 切换卫星图层
  useEffect(() => {
    const map = mapRef.current;
    if (!map || !isLoaded) return;

    const style = showSatellite
      ? 'mapbox://styles/mapbox/satellite-streets-v12'
      : 'mapbox://styles/mapbox/streets-v12';
    
    map.setStyle(style);
  }, [showSatellite, isLoaded, mapRef]);

  // 适应所有车辆
  const fitToVehicles = useCallback(() => {
    if (!mapRef.current || vehicles.length === 0) return;

    const bounds = new mapboxgl.LngLatBounds();
    vehicles.forEach(v => {
      bounds.extend([v.lon, v.lat]);
    });

    fitBounds(bounds, { padding: 100 });
  }, [vehicles, fitBounds]);

  // 当车辆数据变化时自动适应边界
  useEffect(() => {
    if (isLoaded && vehicles.length > 0 && !selectedVehicleId) {
      fitToVehicles();
    }
  }, [isLoaded, vehicles.length, selectedVehicleId, fitToVehicles]);

  return (
    <div style={{ ...mapContainerStyle, ...style }} className={className}>
      <div ref={containerRef} style={{ width: '100%', height: '100%' }} />
      
      {/* 加载状态 */}
      {!isLoaded && !error && (
        <div style={loadingStyle}>
          <Spin size="large" tip="地图加载中..." />
        </div>
      )}
      
      {/* 错误提示 */}
      {error && (
        <div style={errorStyle}>
          <Alert
            message="地图加载失败"
            description={error.message || '请检查 Mapbox Token 配置'}
            type="error"
            showIcon
          />
        </div>
      )}
    </div>
  );
});

MapboxMap.displayName = 'MapboxMap';

export default MapboxMap;
