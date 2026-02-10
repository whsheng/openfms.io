import React, { useEffect, useRef, useCallback } from 'react';
import mapboxgl from 'mapbox-gl';
import { createRoot, Root } from 'react-dom/client';

export interface VehicleData {
  deviceId: string;
  lat: number;
  lon: number;
  speed: number;
  heading: number; // 方向角度 0-360
  status: 'online' | 'offline' | 'moving' | 'idle';
  lastReportTime: string;
  plateNumber?: string;
  driverName?: string;
}

interface VehicleMarkerProps {
  map: mapboxgl.Map;
  vehicle: VehicleData;
  onClick?: (vehicle: VehicleData) => void;
  isSelected?: boolean;
}

// 车辆状态颜色
const STATUS_COLORS = {
  online: '#52c41a',
  offline: '#bfbfbf',
  moving: '#1890ff',
  idle: '#faad14',
};

// 车辆状态文本
const STATUS_TEXT = {
  online: '在线',
  offline: '离线',
  moving: '行驶中',
  idle: '静止',
};

// 车辆标记组件
const VehicleIcon: React.FC<{
  vehicle: VehicleData;
  isSelected: boolean;
  onClick: () => void;
}> = ({ vehicle, isSelected, onClick }) => {
  const color = STATUS_COLORS[vehicle.status];
  
  return (
    <div
      onClick={onClick}
      style={{
        cursor: 'pointer',
        transform: `rotate(${vehicle.heading}deg)`,
        transition: 'transform 0.3s ease',
      }}
    >
      <svg
        width={isSelected ? 48 : 36}
        height={isSelected ? 48 : 36}
        viewBox="0 0 36 36"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
      >
        {/* 车辆图标 - 箭头形状 */}
        <g transform="translate(18, 18)">
          <path
            d="M-8 -12 L0 -18 L8 -12 L4 0 L4 12 L-4 12 L-4 0 Z"
            fill={color}
            stroke="#fff"
            strokeWidth="2"
          />
          {/* 中心点 */}
          <circle cx="0" cy="0" r="3" fill="#fff" />
        </g>
      </svg>
      {/* 选中状态光环 */}
      {isSelected && (
        <div
          style={{
            position: 'absolute',
            top: '50%',
            left: '50%',
            transform: 'translate(-50%, -50%)',
            width: 60,
            height: 60,
            borderRadius: '50%',
            border: `3px solid ${color}`,
            opacity: 0.5,
            animation: 'pulse 1.5s infinite',
            pointerEvents: 'none',
          }}
        />
      )}
    </div>
  );
};

// 信息弹窗组件
const VehiclePopup: React.FC<{
  vehicle: VehicleData;
}> = ({ vehicle }) => {
  const color = STATUS_COLORS[vehicle.status];
  
  return (
    <div
      style={{
        padding: 12,
        minWidth: 200,
        fontSize: 14,
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          marginBottom: 12,
          paddingBottom: 8,
          borderBottom: '1px solid #f0f0f0',
        }}
      >
        <div
          style={{
            width: 10,
            height: 10,
            borderRadius: '50%',
            backgroundColor: color,
          }}
        />
        <span style={{ fontWeight: 600, fontSize: 16 }}>
          {vehicle.plateNumber || vehicle.deviceId}
        </span>
      </div>
      
      <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
          <span style={{ color: '#666' }}>状态:</span>
          <span style={{ color, fontWeight: 500 }}>{STATUS_TEXT[vehicle.status]}</span>
        </div>
        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
          <span style={{ color: '#666' }}>速度:</span>
          <span style={{ fontWeight: 500 }}>{vehicle.speed} km/h</span>
        </div>
        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
          <span style={{ color: '#666' }}>方向:</span>
          <span style={{ fontWeight: 500 }}>{vehicle.heading}°</span>
        </div>
        {vehicle.driverName && (
          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <span style={{ color: '#666' }}>驾驶员:</span>
            <span>{vehicle.driverName}</span>
          </div>
        )}
        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
          <span style={{ color: '#666' }}>上报时间:</span>
          <span style={{ fontSize: 12 }}>
            {new Date(vehicle.lastReportTime).toLocaleString('zh-CN')}
          </span>
        </div>
      </div>
    </div>
  );
};

export const VehicleMarker: React.FC<VehicleMarkerProps> = ({
  map,
  vehicle,
  onClick,
  isSelected = false,
}) => {
  const markerRef = useRef<mapboxgl.Marker | null>(null);
  const popupRef = useRef<mapboxgl.Popup | null>(null);
  const rootRef = useRef<Root | null>(null);
  const containerRef = useRef<HTMLDivElement | null>(null);

  // 创建或更新标记
  useEffect(() => {
    if (!containerRef.current) {
      containerRef.current = document.createElement('div');
      rootRef.current = createRoot(containerRef.current);
    }

    // 渲染车辆图标
    const handleClick = () => {
      onClick?.(vehicle);
    };

    rootRef.current?.render(
      <VehicleIcon
        vehicle={vehicle}
        isSelected={isSelected}
        onClick={handleClick}
      />
    );

    // 创建或更新 marker
    if (!markerRef.current) {
      markerRef.current = new mapboxgl.Marker({
        element: containerRef.current,
        anchor: 'center',
        rotationAlignment: 'map',
        pitchAlignment: 'map',
      })
        .setLngLat([vehicle.lon, vehicle.lat])
        .addTo(map);
    } else {
      markerRef.current.setLngLat([vehicle.lon, vehicle.lat]);
    }

    return () => {
      // 组件卸载时不销毁 marker，由父组件管理
    };
  }, [map, vehicle, isSelected, onClick]);

  // 处理弹窗
  useEffect(() => {
    if (isSelected) {
      // 创建弹窗
      const popupContainer = document.createElement('div');
      const popupRoot = createRoot(popupContainer);
      popupRoot.render(<VehiclePopup vehicle={vehicle} />);

      popupRef.current = new mapboxgl.Popup({
        closeButton: false,
        closeOnClick: false,
        offset: [0, -20],
        className: 'vehicle-popup',
      })
        .setDOMContent(popupContainer)
        .setLngLat([vehicle.lon, vehicle.lat]);

      if (markerRef.current) {
        markerRef.current.setPopup(popupRef.current);
        popupRef.current.addTo(map);
      }

      return () => {
        popupRef.current?.remove();
        popupRoot.unmount();
      };
    } else {
      popupRef.current?.remove();
      popupRef.current = null;
    }
  }, [isSelected, map, vehicle]);

  // 清理
  useEffect(() => {
    return () => {
      markerRef.current?.remove();
      rootRef.current?.unmount();
    };
  }, []);

  return null;
};

// 管理多个车辆标记的 Hook
export function useVehicleMarkers(map: mapboxgl.Map | null) {
  const markersRef = useRef<Map<string, mapboxgl.Marker>>(new Map());
  const selectedIdRef = useRef<string | null>(null);

  const updateMarkers = useCallback((
    vehicles: VehicleData[],
    onVehicleClick?: (vehicle: VehicleData) => void
  ) => {
    if (!map) return;

    const currentIds = new Set(vehicles.map(v => v.deviceId));
    
    // 移除不再存在的标记
    markersRef.current.forEach((marker, id) => {
      if (!currentIds.has(id)) {
        marker.remove();
        markersRef.current.delete(id);
      }
    });

    // 更新或创建标记
    vehicles.forEach(vehicle => {
      const existingMarker = markersRef.current.get(vehicle.deviceId);
      
      if (existingMarker) {
        // 更新位置
        existingMarker.setLngLat([vehicle.lon, vehicle.lat]);
        
        // 更新旋转（通过更新 DOM 元素）
        const element = existingMarker.getElement();
        const svgContainer = element.querySelector('div');
        if (svgContainer) {
          svgContainer.style.transform = `rotate(${vehicle.heading}deg)`;
        }
      } else {
        // 创建新标记
        const container = document.createElement('div');
        const root = createRoot(container);
        
        const handleClick = () => {
          selectedIdRef.current = vehicle.deviceId;
          onVehicleClick?.(vehicle);
        };

        root.render(
          <VehicleIcon
            vehicle={vehicle}
            isSelected={selectedIdRef.current === vehicle.deviceId}
            onClick={handleClick}
          />
        );

        const marker = new mapboxgl.Marker({
          element: container,
          anchor: 'center',
          rotationAlignment: 'map',
          pitchAlignment: 'map',
        })
          .setLngLat([vehicle.lon, vehicle.lat])
          .addTo(map);

        markersRef.current.set(vehicle.deviceId, marker);
      }
    });
  }, [map]);

  const clearMarkers = useCallback(() => {
    markersRef.current.forEach(marker => marker.remove());
    markersRef.current.clear();
  }, []);

  const setSelectedVehicle = useCallback((deviceId: string | null) => {
    selectedIdRef.current = deviceId;
  }, []);

  return {
    updateMarkers,
    clearMarkers,
    setSelectedVehicle,
  };
}

export default VehicleMarker;
