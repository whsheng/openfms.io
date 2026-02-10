import { useEffect, useRef, useState, useCallback } from 'react';

// WebSocket connection states
export enum WSConnectionState {
  CONNECTING = 'CONNECTING',
  CONNECTED = 'CONNECTED',
  DISCONNECTED = 'DISCONNECTED',
  RECONNECTING = 'RECONNECTING',
  ERROR = 'ERROR',
}

// WebSocket message types
export interface WSMessage<T = unknown> {
  type: string;
  data?: T;
  message?: string;
  client_id?: string;
}

// Location message data
export interface LocationMessage {
  device_id: string;
  lat: number;
  lon: number;
  speed: number;
  direction: number;
  timestamp: number;
  status?: number;
}

// Alarm message data
export interface AlarmMessage {
  id: string;
  type: string;
  level: string;
  device_id: string;
  device_name: string;
  content: string;
  location?: {
    lat: number;
    lon: number;
  };
  created_at: string;
  extra_data?: Record<string, unknown>;
}

// WebSocket options
export interface UseWebSocketOptions {
  url?: string;
  autoConnect?: boolean;
  reconnectInterval?: number;
  maxReconnectAttempts?: number;
  heartbeatInterval?: number;
  deviceId?: string;
  onMessage?: (message: WSMessage) => void;
  onLocation?: (data: LocationMessage) => void;
  onAlarm?: (data: AlarmMessage) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
  onError?: (error: Event) => void;
}

// WebSocket hook return type
export interface UseWebSocketReturn {
  connectionState: WSConnectionState;
  lastMessage: WSMessage | null;
  locationMessages: LocationMessage[];
  alarmMessages: AlarmMessage[];
  connect: () => void;
  disconnect: () => void;
  sendMessage: (message: unknown) => void;
  subscribeToDevice: (deviceId: string) => void;
  clearMessages: () => void;
  connected: boolean;
}

const WS_URL = '/ws/location';

/**
 * React hook for WebSocket connection with auto-reconnect and heartbeat
 */
export function useWebSocket(options: UseWebSocketOptions = {}): UseWebSocketReturn {
  const {
    url = WS_URL,
    autoConnect = true,
    reconnectInterval = 3000,
    maxReconnectAttempts = 10,
    heartbeatInterval = 30000,
    deviceId,
    onMessage,
    onLocation,
    onAlarm,
    onConnect,
    onDisconnect,
    onError,
  } = options;

  const [connectionState, setConnectionState] = useState<WSConnectionState>(
    WSConnectionState.DISCONNECTED
  );
  const [lastMessage, setLastMessage] = useState<WSMessage | null>(null);
  const [locationMessages, setLocationMessages] = useState<LocationMessage[]>([]);
  const [alarmMessages, setAlarmMessages] = useState<AlarmMessage[]>([]);

  const wsRef = useRef<WebSocket | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const reconnectTimerRef = useRef<NodeJS.Timeout | null>(null);
  const heartbeatTimerRef = useRef<NodeJS.Timeout | null>(null);
  const isManualDisconnectRef = useRef(false);

  // Clear all timers
  const clearTimers = useCallback(() => {
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current);
      reconnectTimerRef.current = null;
    }
    if (heartbeatTimerRef.current) {
      clearInterval(heartbeatTimerRef.current);
      heartbeatTimerRef.current = null;
    }
  }, []);

  // Start heartbeat
  const startHeartbeat = useCallback(() => {
    if (heartbeatTimerRef.current) {
      clearInterval(heartbeatTimerRef.current);
    }
    heartbeatTimerRef.current = setInterval(() => {
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        wsRef.current.send(JSON.stringify({ type: 'ping' }));
      }
    }, heartbeatInterval);
  }, [heartbeatInterval]);

  // Connect to WebSocket
  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return;
    }

    setConnectionState(WSConnectionState.CONNECTING);
    isManualDisconnectRef.current = false;

    // Build URL with query parameters
    const wsUrl = new URL(url, window.location.origin);
    wsUrl.protocol = wsUrl.protocol === 'https:' ? 'wss:' : 'ws:';
    
    if (deviceId) {
      wsUrl.searchParams.set('device_id', deviceId);
    }

    try {
      const ws = new WebSocket(wsUrl.toString());
      wsRef.current = ws;

      ws.onopen = () => {
        console.log('[WebSocket] Connected');
        setConnectionState(WSConnectionState.CONNECTED);
        reconnectAttemptsRef.current = 0;
        startHeartbeat();
        onConnect?.();
      };

      ws.onmessage = (event) => {
        try {
          const message: WSMessage = JSON.parse(event.data);
          setLastMessage(message);

          // Handle different message types
          switch (message.type) {
            case 'location':
              if (message.data) {
                const locationData = message.data as LocationMessage;
                setLocationMessages((prev) => {
                  const newMessages = [...prev, locationData];
                  // Keep last 1000 messages to prevent memory issues
                  if (newMessages.length > 1000) {
                    return newMessages.slice(-1000);
                  }
                  return newMessages;
                });
                onLocation?.(locationData);
              }
              break;
            case 'alarm':
              if (message.data) {
                const alarmData = message.data as AlarmMessage;
                setAlarmMessages((prev) => {
                  const newMessages = [alarmData, ...prev];
                  // Keep last 100 messages
                  if (newMessages.length > 100) {
                    return newMessages.slice(0, 100);
                  }
                  return newMessages;
                });
                onAlarm?.(alarmData);
              }
              break;
            case 'connected':
              console.log('[WebSocket] Server message:', message.message);
              break;
            case 'pong':
              // Heartbeat response, no action needed
              break;
            default:
              break;
          }

          onMessage?.(message);
        } catch (error) {
          console.error('[WebSocket] Failed to parse message:', error);
        }
      };

      ws.onclose = (event) => {
        console.log('[WebSocket] Disconnected:', event.code, event.reason);
        setConnectionState(WSConnectionState.DISCONNECTED);
        clearTimers();
        onDisconnect?.();

        // Auto reconnect if not manually disconnected
        if (!isManualDisconnectRef.current && reconnectAttemptsRef.current < maxReconnectAttempts) {
          setConnectionState(WSConnectionState.RECONNECTING);
          reconnectAttemptsRef.current += 1;
          console.log(
            `[WebSocket] Reconnecting in ${reconnectInterval}ms (attempt ${reconnectAttemptsRef.current}/${maxReconnectAttempts})`
          );
          
          reconnectTimerRef.current = setTimeout(() => {
            connect();
          }, reconnectInterval);
        }
      };

      ws.onerror = (error) => {
        console.error('[WebSocket] Error:', error);
        setConnectionState(WSConnectionState.ERROR);
        onError?.(error);
      };
    } catch (error) {
      console.error('[WebSocket] Failed to create connection:', error);
      setConnectionState(WSConnectionState.ERROR);
    }
  }, [url, deviceId, reconnectInterval, maxReconnectAttempts, startHeartbeat, onConnect, onDisconnect, onMessage, onLocation, onAlarm, onError]);

  // Disconnect from WebSocket
  const disconnect = useCallback(() => {
    isManualDisconnectRef.current = true;
    clearTimers();
    
    if (wsRef.current) {
      if (wsRef.current.readyState === WebSocket.OPEN) {
        wsRef.current.close(1000, 'Manual disconnect');
      }
      wsRef.current = null;
    }
    
    setConnectionState(WSConnectionState.DISCONNECTED);
  }, [clearTimers]);

  // Send message to server
  const sendMessage = useCallback((message: unknown) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(message));
    } else {
      console.warn('[WebSocket] Cannot send message, not connected');
    }
  }, []);

  // Subscribe to specific device updates
  const subscribeToDevice = useCallback((deviceId: string) => {
    sendMessage({
      type: 'subscribe',
      data: { device_id: deviceId },
    });
  }, [sendMessage]);

  // Clear all messages
  const clearMessages = useCallback(() => {
    setLocationMessages([]);
    setAlarmMessages([]);
    setLastMessage(null);
  }, []);

  // Auto connect on mount
  useEffect(() => {
    if (autoConnect) {
      connect();
    }

    return () => {
      disconnect();
    };
  }, [autoConnect, connect, disconnect]);

  const connected = connectionState === WSConnectionState.CONNECTED;

  return {
    connectionState,
    lastMessage,
    locationMessages,
    alarmMessages,
    connect,
    disconnect,
    sendMessage,
    subscribeToDevice,
    clearMessages,
    connected,
  };
}

export default useWebSocket;
