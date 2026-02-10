import { useAuthStore } from '../stores/auth';

// WebSocket service configuration
const WS_CONFIG = {
  baseUrl: import.meta.env.VITE_WS_URL || '',
  reconnectInterval: 3000,
  maxReconnectAttempts: 10,
  heartbeatInterval: 30000,
  messageBufferSize: 1000,
};

// Message types
export type WSMessageType = 
  | 'location' 
  | 'connected' 
  | 'disconnected' 
  | 'ping' 
  | 'pong' 
  | 'subscribe' 
  | 'error';

export interface WSMessage<T = unknown> {
  type: WSMessageType;
  data?: T;
  message?: string;
  client_id?: string;
  error?: string;
}

export interface LocationData {
  device_id: string;
  lat: number;
  lon: number;
  speed: number;
  direction: number;
  timestamp: number;
  status?: number;
}

// Connection state
export enum ConnectionState {
  DISCONNECTED = 'disconnected',
  CONNECTING = 'connecting',
  CONNECTED = 'connected',
  RECONNECTING = 'reconnecting',
  ERROR = 'error',
}

// Event handlers type
export type MessageHandler = (message: WSMessage) => void;
export type ConnectionHandler = () => void;
export type ErrorHandler = (error: Event | Error) => void;

/**
 * WebSocket Service - Singleton class for managing WebSocket connections
 */
class WebSocketService {
  private static instance: WebSocketService;
  private ws: WebSocket | null = null;
  private connectionState: ConnectionState = ConnectionState.DISCONNECTED;
  private reconnectAttempts = 0;
  private reconnectTimer: NodeJS.Timeout | null = null;
  private heartbeatTimer: NodeJS.Timeout | null = null;
  private isManualDisconnect = false;

  // Event listeners
  private messageHandlers: Set<MessageHandler> = new Set();
  private connectHandlers: Set<ConnectionHandler> = new Set();
  private disconnectHandlers: Set<ConnectionHandler> = new Set();
  private errorHandlers: Set<ErrorHandler> = new Set();

  // Message buffer for offline mode
  private messageBuffer: WSMessage[] = [];

  private constructor() {}

  static getInstance(): WebSocketService {
    if (!WebSocketService.instance) {
      WebSocketService.instance = new WebSocketService();
    }
    return WebSocketService.instance;
  }

  /**
   * Get current connection state
   */
  getConnectionState(): ConnectionState {
    return this.connectionState;
  }

  /**
   * Check if connected
   */
  isConnected(): boolean {
    return this.connectionState === ConnectionState.CONNECTED;
  }

  /**
   * Get WebSocket URL
   */
  private getWebSocketUrl(deviceId?: string): string {
    let baseUrl = WS_CONFIG.baseUrl;
    
    if (!baseUrl) {
      // Derive from current location
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const host = window.location.host;
      baseUrl = `${protocol}//${host}`;
    }

    const url = new URL('/ws/location', baseUrl);
    
    if (deviceId) {
      url.searchParams.set('device_id', deviceId);
    }

    // Add auth token if available
    const token = useAuthStore.getState().token;
    if (token) {
      url.searchParams.set('token', token);
    }

    return url.toString();
  }

  /**
   * Connect to WebSocket server
   */
  connect(deviceId?: string): Promise<void> {
    return new Promise((resolve, reject) => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        resolve();
        return;
      }

      this.connectionState = ConnectionState.CONNECTING;
      this.isManualDisconnect = false;

      try {
        const wsUrl = this.getWebSocketUrl(deviceId);
        this.ws = new WebSocket(wsUrl);

        this.ws.onopen = () => {
          console.log('[WebSocketService] Connected');
          this.connectionState = ConnectionState.CONNECTED;
          this.reconnectAttempts = 0;
          this.startHeartbeat();
          this.connectHandlers.forEach((handler) => handler());
          resolve();
        };

        this.ws.onmessage = (event) => {
          this.handleMessage(event.data);
        };

        this.ws.onclose = (event) => {
          console.log('[WebSocketService] Disconnected:', event.code, event.reason);
          this.connectionState = ConnectionState.DISCONNECTED;
          this.stopHeartbeat();
          this.disconnectHandlers.forEach((handler) => handler());
          this.attemptReconnect(deviceId);
        };

        this.ws.onerror = (error) => {
          console.error('[WebSocketService] Error:', error);
          this.connectionState = ConnectionState.ERROR;
          this.errorHandlers.forEach((handler) => handler(error));
          reject(error);
        };
      } catch (error) {
        console.error('[WebSocketService] Failed to connect:', error);
        this.connectionState = ConnectionState.ERROR;
        reject(error);
      }
    });
  }

  /**
   * Disconnect from WebSocket server
   */
  disconnect(): void {
    this.isManualDisconnect = true;
    this.stopReconnect();
    this.stopHeartbeat();

    if (this.ws) {
      if (this.ws.readyState === WebSocket.OPEN) {
        this.ws.close(1000, 'Manual disconnect');
      }
      this.ws = null;
    }

    this.connectionState = ConnectionState.DISCONNECTED;
  }

  /**
   * Send message to server
   */
  send(message: unknown): boolean {
    if (this.ws?.readyState === WebSocket.OPEN) {
      try {
        this.ws.send(JSON.stringify(message));
        return true;
      } catch (error) {
        console.error('[WebSocketService] Failed to send message:', error);
        return false;
      }
    }
    console.warn('[WebSocketService] Cannot send message, not connected');
    return false;
  }

  /**
   * Subscribe to device updates
   */
  subscribeToDevice(deviceId: string): void {
    this.send({
      type: 'subscribe',
      data: { device_id: deviceId },
    });
  }

  /**
   * Send ping message
   */
  ping(): void {
    this.send({ type: 'ping' });
  }

  /**
   * Handle incoming message
   */
  private handleMessage(data: string): void {
    try {
      const message: WSMessage = JSON.parse(data);
      
      // Buffer message if needed
      this.bufferMessage(message);

      // Notify all handlers
      this.messageHandlers.forEach((handler) => {
        try {
          handler(message);
        } catch (error) {
          console.error('[WebSocketService] Message handler error:', error);
        }
      });
    } catch (error) {
      console.error('[WebSocketService] Failed to parse message:', error);
    }
  }

  /**
   * Buffer message for offline access
   */
  private bufferMessage(message: WSMessage): void {
    this.messageBuffer.push(message);
    if (this.messageBuffer.length > WS_CONFIG.messageBufferSize) {
      this.messageBuffer.shift();
    }
  }

  /**
   * Get message buffer
   */
  getMessageBuffer(): WSMessage[] {
    return [...this.messageBuffer];
  }

  /**
   * Clear message buffer
   */
  clearBuffer(): void {
    this.messageBuffer = [];
  }

  /**
   * Attempt to reconnect
   */
  private attemptReconnect(deviceId?: string): void {
    if (this.isManualDisconnect) return;
    if (this.reconnectAttempts >= WS_CONFIG.maxReconnectAttempts) {
      console.error('[WebSocketService] Max reconnect attempts reached');
      return;
    }

    this.connectionState = ConnectionState.RECONNECTING;
    this.reconnectAttempts++;

    console.log(
      `[WebSocketService] Reconnecting in ${WS_CONFIG.reconnectInterval}ms (attempt ${this.reconnectAttempts}/${WS_CONFIG.maxReconnectAttempts})`
    );

    this.reconnectTimer = setTimeout(() => {
      this.connect(deviceId).catch(() => {
        // Reconnect failed, will try again
      });
    }, WS_CONFIG.reconnectInterval);
  }

  /**
   * Stop reconnect timer
   */
  private stopReconnect(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  /**
   * Start heartbeat
   */
  private startHeartbeat(): void {
    this.stopHeartbeat();
    this.heartbeatTimer = setInterval(() => {
      this.ping();
    }, WS_CONFIG.heartbeatInterval);
  }

  /**
   * Stop heartbeat
   */
  private stopHeartbeat(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
  }

  // Event handler management

  /**
   * Add message handler
   */
  onMessage(handler: MessageHandler): () => void {
    this.messageHandlers.add(handler);
    return () => {
      this.messageHandlers.delete(handler);
    };
  }

  /**
   * Add connect handler
   */
  onConnect(handler: ConnectionHandler): () => void {
    this.connectHandlers.add(handler);
    return () => {
      this.connectHandlers.delete(handler);
    };
  }

  /**
   * Add disconnect handler
   */
  onDisconnect(handler: ConnectionHandler): () => void {
    this.disconnectHandlers.add(handler);
    return () => {
      this.disconnectHandlers.delete(handler);
    };
  }

  /**
   * Add error handler
   */
  onError(handler: ErrorHandler): () => void {
    this.errorHandlers.add(handler);
    return () => {
      this.errorHandlers.delete(handler);
    };
  }

  /**
   * Remove all handlers
   */
  removeAllHandlers(): void {
    this.messageHandlers.clear();
    this.connectHandlers.clear();
    this.disconnectHandlers.clear();
    this.errorHandlers.clear();
  }
}

// Export singleton instance
export const wsService = WebSocketService.getInstance();

// Export class for testing
export { WebSocketService };

// Helper functions for common operations

/**
 * Connect to location WebSocket
 */
export function connectToLocationStream(deviceId?: string): Promise<void> {
  return wsService.connect(deviceId);
}

/**
 * Disconnect from location WebSocket
 */
export function disconnectFromLocationStream(): void {
  wsService.disconnect();
}

/**
 * Subscribe to location updates
 */
export function subscribeToLocationUpdates(
  callback: (location: LocationData) => void
): () => void {
  return wsService.onMessage((message) => {
    if (message.type === 'location' && message.data) {
      callback(message.data as LocationData);
    }
  });
}

/**
 * Get current connection state
 */
export function getConnectionState(): ConnectionState {
  return wsService.getConnectionState();
}

export default wsService;
