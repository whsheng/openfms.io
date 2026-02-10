// API Base URL
export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || ''

// WebSocket URL
export const WS_URL = import.meta.env.VITE_WS_URL || '/ws/location'

// Mapbox Token
export const MAPBOX_TOKEN = import.meta.env.VITE_MAPBOX_TOKEN || ''

// Default pagination
export const DEFAULT_PAGE_SIZE = 20
export const DEFAULT_PAGE_SIZE_OPTIONS = [10, 20, 50, 100]

// Date formats
export const DATE_FORMAT = 'YYYY-MM-DD'
export const DATETIME_FORMAT = 'YYYY-MM-DD HH:mm:ss'

// Alarm types
export const ALARM_TYPE_GEOFENCE_ENTER = 'GEOFENCE_ENTER'
export const ALARM_TYPE_GEOFENCE_EXIT = 'GEOFENCE_EXIT'
export const ALARM_TYPE_OVERSPEED = 'OVERSPEED'
export const ALARM_TYPE_LOW_BATTERY = 'LOW_BATTERY'
export const ALARM_TYPE_OFFLINE = 'OFFLINE'
export const ALARM_TYPE_SOS = 'SOS'

// Alarm levels
export const ALARM_LEVEL_INFO = 'info'
export const ALARM_LEVEL_WARNING = 'warning'
export const ALARM_LEVEL_CRITICAL = 'critical'

// Alarm status
export const ALARM_STATUS_UNREAD = 'unread'
export const ALARM_STATUS_READ = 'read'
export const ALARM_STATUS_RESOLVED = 'resolved'
