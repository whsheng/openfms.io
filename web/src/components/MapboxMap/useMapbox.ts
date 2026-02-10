import { useEffect, useRef, useState, useCallback } from 'react';
import mapboxgl from 'mapbox-gl';
import 'mapbox-gl/dist/mapbox-gl.css';

// Mapbox Token - 使用占位符，后续替换
const MAPBOX_TOKEN = 'YOUR_MAPBOX_TOKEN';

// 地图配置
const MAP_CONFIG = {
  style: 'mapbox://styles/mapbox/streets-v12',
  center: [116.4074, 39.9042] as [number, number], // 北京
  zoom: 12,
  minZoom: 3,
  maxZoom: 20,
  language: 'zh-Hans', // 简体中文
};

export interface MapInstance {
  map: mapboxgl.Map | null;
  container: HTMLDivElement | null;
}

export interface UseMapboxOptions {
  containerId?: string;
  initialCenter?: [number, number];
  initialZoom?: number;
  onMapLoad?: (map: mapboxgl.Map) => void;
  onMapError?: (error: Error) => void;
}

export function useMapbox(options: UseMapboxOptions = {}) {
  const {
    containerId,
    initialCenter = MAP_CONFIG.center,
    initialZoom = MAP_CONFIG.zoom,
    onMapLoad,
    onMapError,
  } = options;

  const mapRef = useRef<mapboxgl.Map | null>(null);
  const containerRef = useRef<HTMLDivElement | null>(null);
  const [isLoaded, setIsLoaded] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  // 初始化地图
  const initializeMap = useCallback(() => {
    if (!containerRef.current) return;
    if (mapRef.current) return;

    try {
      // 设置 Mapbox Token
      mapboxgl.accessToken = MAPBOX_TOKEN;

      const map = new mapboxgl.Map({
        container: containerRef.current,
        style: MAP_CONFIG.style,
        center: initialCenter,
        zoom: initialZoom,
        minZoom: MAP_CONFIG.minZoom,
        maxZoom: MAP_CONFIG.maxZoom,
        attributionControl: false,
      });

      // 添加语言支持
      map.on('style.load', () => {
        map.setConfigProperty('basemap', 'language', MAP_CONFIG.language);
      });

      // 添加控件
      map.addControl(
        new mapboxgl.NavigationControl({
          showCompass: true,
          showZoom: true,
          visualizePitch: true,
        }),
        'top-right'
      );

      map.addControl(
        new mapboxgl.FullscreenControl(),
        'top-right'
      );

      map.addControl(
        new mapboxgl.GeolocateControl({
          positionOptions: {
            enableHighAccuracy: true,
          },
          trackUserLocation: true,
          showUserHeading: true,
        }),
        'top-right'
      );

      map.addControl(
        new mapboxgl.ScaleControl({
          maxWidth: 100,
          unit: 'metric',
        }),
        'bottom-left'
      );

      map.addControl(
        new mapboxgl.AttributionControl({
          compact: true,
        }),
        'bottom-right'
      );

      // 地图加载完成
      map.on('load', () => {
        setIsLoaded(true);
        onMapLoad?.(map);
      });

      // 错误处理
      map.on('error', (e) => {
        const err = e.error || new Error('Mapbox error');
        setError(err);
        onMapError?.(err);
      });

      mapRef.current = map;
    } catch (err) {
      const error = err instanceof Error ? err : new Error('Failed to initialize map');
      setError(error);
      onMapError?.(error);
    }
  }, [initialCenter, initialZoom, onMapLoad, onMapError]);

  // 销毁地图
  const destroyMap = useCallback(() => {
    if (mapRef.current) {
      mapRef.current.remove();
      mapRef.current = null;
      setIsLoaded(false);
    }
  }, []);

  // 设置地图中心
  const setCenter = useCallback((center: [number, number], zoom?: number) => {
    if (mapRef.current) {
      mapRef.current.flyTo({
        center,
        zoom: zoom || mapRef.current.getZoom(),
        essential: true,
        duration: 1000,
      });
    }
  }, []);

  // 适应边界
  const fitBounds = useCallback((bounds: mapboxgl.LngLatBoundsLike, options?: mapboxgl.FitBoundsOptions) => {
    if (mapRef.current) {
      mapRef.current.fitBounds(bounds, {
        padding: 50,
        maxZoom: 16,
        ...options,
      });
    }
  }, []);

  // 获取地图实例
  const getMap = useCallback(() => mapRef.current, []);

  // 组件挂载时初始化
  useEffect(() => {
    initializeMap();
    return () => {
      destroyMap();
    };
  }, [initializeMap, destroyMap]);

  return {
    mapRef,
    containerRef,
    isLoaded,
    error,
    setCenter,
    fitBounds,
    getMap,
  };
}

export default useMapbox;
