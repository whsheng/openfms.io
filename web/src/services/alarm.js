import api from './api'

// Alarm API
export const alarmApi = {
  // Get alarm list with filters
  getAlarms: (params = {}) => {
    return api.get('/alarms', { params })
  },

  // Get alarm statistics
  getAlarmStats: () => {
    return api.get('/alarms/stats')
  },

  // Get unread count
  getUnreadCount: () => {
    return api.get('/alarms/unread-count')
  },

  // Get alarm by ID
  getAlarm: (id) => {
    return api.get(`/alarms/${id}`)
  },

  // Mark alarm as read
  markAsRead: (id) => {
    return api.post(`/alarms/${id}/read`)
  },

  // Resolve alarm
  resolveAlarm: (id, note) => {
    return api.post(`/alarms/${id}/resolve`, { note })
  },

  // Batch resolve alarms
  batchResolve: (ids, note = '') => {
    return api.post('/alarms/batch-resolve', { ids, note })
  },

  // Get alarm rules
  getRules: () => {
    return api.get('/alarms/rules')
  },

  // Update alarm rule
  updateRule: (id, data) => {
    return api.put(`/alarms/rules/${id}`, data)
  },
}

// WebSocket connection for real-time alarms
export class AlarmWebSocket {
  constructor(onMessage, onConnect, onDisconnect) {
    this.ws = null
    this.onMessage = onMessage
    this.onConnect = onConnect
    this.onDisconnect = onDisconnect
    this.reconnectInterval = 5000
    this.reconnectTimer = null
    this.url = `${import.meta.env.VITE_WS_URL || 'ws://localhost:3000'}/ws/location`
  }

  connect() {
    try {
      this.ws = new WebSocket(this.url)

      this.ws.onopen = () => {
        console.log('[AlarmWS] Connected')
        if (this.onConnect) this.onConnect()
        // Clear any pending reconnect
        if (this.reconnectTimer) {
          clearTimeout(this.reconnectTimer)
          this.reconnectTimer = null
        }
      }

      this.ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data)
          if (data.type === 'alarm' && this.onMessage) {
            this.onMessage(data.data)
          }
        } catch (err) {
          console.error('[AlarmWS] Failed to parse message:', err)
        }
      }

      this.ws.onclose = () => {
        console.log('[AlarmWS] Disconnected')
        if (this.onDisconnect) this.onDisconnect()
        this.scheduleReconnect()
      }

      this.ws.onerror = (error) => {
        console.error('[AlarmWS] Error:', error)
      }
    } catch (err) {
      console.error('[AlarmWS] Failed to connect:', err)
      this.scheduleReconnect()
    }
  }

  scheduleReconnect() {
    if (!this.reconnectTimer) {
      this.reconnectTimer = setTimeout(() => {
        console.log('[AlarmWS] Reconnecting...')
        this.connect()
      }, this.reconnectInterval)
    }
  }

  disconnect() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
  }

  // Send ping to keep connection alive
  ping() {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type: 'ping' }))
    }
  }
}

// Alarm type labels and colors
export const ALARM_TYPES = {
  GEOFENCE_ENTER: { label: '进入围栏', color: 'blue' },
  GEOFENCE_EXIT: { label: '离开围栏', color: 'orange' },
  OVERSPEED: { label: '超速报警', color: 'red' },
  LOW_BATTERY: { label: '低电量', color: 'gold' },
  OFFLINE: { label: '设备离线', color: 'gray' },
  SOS: { label: '紧急求救', color: 'purple' },
}

// Alarm level colors
export const ALARM_LEVELS = {
  info: { label: '信息', color: 'blue' },
  warning: { label: '警告', color: 'orange' },
  critical: { label: '紧急', color: 'red' },
}

// Alarm status labels
export const ALARM_STATUS = {
  unread: { label: '未读', color: 'red' },
  read: { label: '已读', color: 'blue' },
  resolved: { label: '已处理', color: 'green' },
}
