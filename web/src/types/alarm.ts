// 报警类型
export type AlarmType = 
  | 'GEOFENCE_ENTER'
  | 'GEOFENCE_EXIT'
  | 'OVERSPEED'
  | 'LOW_BATTERY'
  | 'OFFLINE'
  | 'SOS'
  | 'POWER_CUT'
  | 'VIBRATION'
  | 'ILLEGAL_MOVE';

// 报警级别
export type AlarmLevel = 'info' | 'warning' | 'critical';

// 报警状态
export type AlarmStatus = 'unread' | 'read' | 'resolved';

// 报警记录
export interface Alarm {
  id: number;
  type: AlarmType;
  level: AlarmLevel;
  device_id: string;
  device_name?: string;
  title: string;
  content?: string;
  lat?: number;
  lon?: number;
  location_name?: string;
  speed?: number;
  speed_limit?: number;
  status: AlarmStatus;
  resolved_at?: string;
  resolved_by?: number;
  resolve_note?: string;
  geofence_id?: number;
  geofence_name?: string;
  extras?: Record<string, any>;
  created_at: string;
  updated_at: string;
}

// 报警规则
export interface AlarmRule {
  id: number;
  name: string;
  type: AlarmType;
  description?: string;
  conditions: {
    speed_limit?: number;
    offline_minutes?: number;
    battery_threshold?: number;
    geofence_ids?: number[];
  };
  all_devices: boolean;
  device_ids?: number[];
  notify_webhook: boolean;
  webhook_url?: string;
  notify_ws: boolean;
  notify_sound: boolean;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

// 报警统计
export interface AlarmStats {
  total: number;
  unread: number;
  read: number;
  resolved: number;
  today: number;
  critical: number;
  warning: number;
  info: number;
}

// 报警列表查询参数
export interface AlarmListQuery {
  device_id?: string;
  type?: AlarmType;
  level?: AlarmLevel;
  status?: AlarmStatus;
  start_time?: string;
  end_time?: string;
  page?: number;
  page_size?: number;
}

// WebSocket 报警消息
export interface WSAlarmMessage {
  type: 'ALARM';
  data: Alarm;
}
