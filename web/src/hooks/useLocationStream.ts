import { useEffect, useRef, useState, useCallback } from 'react';
import { wsService, LocationData, ConnectionState } from '../services/websocket';
import type { Alarm } from '../types/alarm';

interface UseLocationStreamOptions {
  deviceId?: string;
  autoConnect?: boolean;
  onLocationUpdate?: (location: LocationData) => void;
  onAlarm?: (alarm: Alarm) => void;
}

interface UseLocationStreamReturn {
  locations: LocationData[];
  latestLocation: LocationData | null;
  isConnected: boolean;
  connectionState: ConnectionState;
  connect: () => void;
  disconnect: () => void;
  clearLocations: () => void;
}

/**
 * Hook for streaming location updates via WebSocket
 * 
 * @example
 * ```tsx
 * function VehicleTracker({ deviceId }) {
 *   const { locations, latestLocation, isConnected } = useLocationStream({
 *     deviceId,
 *     onLocationUpdate: (loc) => console.log('New location:', loc),
 *   });
 * 
 *   return (
 *     <div>
 *       <div>Connection: {isConnected ? 'Connected' : 'Disconnected'}</div>
 *       <div>Latest: {latestLocation?.lat}, {latestLocation?.lon}</div>
 *       <div>Total updates: {locations.length}</div>
 *     </div>
 *   );
 * }
 * ```
 */
export function useLocationStream(options: UseLocationStreamOptions = {}): UseLocationStreamReturn {
  const { deviceId, autoConnect = true, onLocationUpdate, onAlarm } = options;

  const [locations, setLocations] = useState<LocationData[]>([]);
  const [latestLocation, setLatestLocation] = useState<LocationData | null>(null);
  const [connectionState, setConnectionState] = useState<ConnectionState>(
    ConnectionState.DISCONNECTED
  );
  const unsubscribeRef = useRef<(() => void) | null>(null);

  // Update connection state
  useEffect(() => {
    const updateState = () => {
      setConnectionState(wsService.getConnectionState());
    };

    updateState();

    const unsubConnect = wsService.onConnect(updateState);
    const unsubDisconnect = wsService.onDisconnect(updateState);

    return () => {
      unsubConnect();
      unsubDisconnect();
    };
  }, []);

  // Subscribe to location updates
  useEffect(() => {
    if (unsubscribeRef.current) {
      unsubscribeRef.current();
    }

    unsubscribeRef.current = wsService.onMessage((message) => {
      if (message.type === 'location' && message.data) {
        const location = message.data as LocationData;
        
        setLocations((prev) => {
          const newLocations = [...prev, location];
          // Keep last 500 locations per device to prevent memory issues
          if (newLocations.length > 500) {
            return newLocations.slice(-500);
          }
          return newLocations;
        });

        setLatestLocation(location);
        onLocationUpdate?.(location);
      } else if (message.type === 'ALARM' && message.data) {
        const alarm = message.data as Alarm;
        onAlarm?.(alarm);
      }
    });

    return () => {
      if (unsubscribeRef.current) {
        unsubscribeRef.current();
        unsubscribeRef.current = null;
      }
    };
  }, [onLocationUpdate]);

  // Auto connect
  useEffect(() => {
    if (autoConnect && wsService.getConnectionState() === ConnectionState.DISCONNECTED) {
      wsService.connect(deviceId);
    }

    return () => {
      // Don't disconnect on unmount if other components are using it
      // wsService.disconnect();
    };
  }, [autoConnect, deviceId]);

  const connect = useCallback(() => {
    wsService.connect(deviceId);
  }, [deviceId]);

  const disconnect = useCallback(() => {
    wsService.disconnect();
  }, []);

  const clearLocations = useCallback(() => {
    setLocations([]);
    setLatestLocation(null);
  }, []);

  return {
    locations,
    latestLocation,
    isConnected: connectionState === ConnectionState.CONNECTED,
    connectionState,
    connect,
    disconnect,
    clearLocations,
  };
}

export default useLocationStream;
