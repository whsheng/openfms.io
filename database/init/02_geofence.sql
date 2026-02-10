-- OpenFMS Geofence Module Database Schema
-- 电子围栏模块数据库表结构

-- ============================================
-- Geofences Table - 围栏主表
-- ============================================
CREATE TABLE IF NOT EXISTS geofences (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    type VARCHAR(20) NOT NULL CHECK (type IN ('circle', 'polygon')), -- 围栏类型：圆形或多边形
    coordinates JSONB NOT NULL, -- 坐标数据
    -- circle: {center: {lat: float, lon: float}, radius: float} (单位：米)
    -- polygon: {points: [{lat: float, lon: float}, ...]}
    alert_type VARCHAR(20) DEFAULT 'both' CHECK (alert_type IN ('enter', 'exit', 'both')), -- 报警类型
    status INTEGER DEFAULT 1, -- 0: 禁用, 1: 启用
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL, -- 创建者
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_geofences_status ON geofences(status) WHERE status = 1;
CREATE INDEX IF NOT EXISTS idx_geofences_user_id ON geofences(user_id);
CREATE INDEX IF NOT EXISTS idx_geofences_deleted_at ON geofences(deleted_at) WHERE deleted_at IS NULL;

-- ============================================
-- Geofence Devices Table - 围栏设备关联表
-- ============================================
CREATE TABLE IF NOT EXISTS geofence_devices (
    id SERIAL PRIMARY KEY,
    geofence_id INTEGER NOT NULL REFERENCES geofences(id) ON DELETE CASCADE,
    device_id INTEGER NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(geofence_id, device_id)
);

CREATE INDEX IF NOT EXISTS idx_geofence_devices_geofence_id ON geofence_devices(geofence_id);
CREATE INDEX IF NOT EXISTS idx_geofence_devices_device_id ON geofence_devices(device_id);

-- ============================================
-- Geofence Events Table - 围栏事件表
-- ============================================
CREATE TABLE IF NOT EXISTS geofence_events (
    id SERIAL PRIMARY KEY,
    geofence_id INTEGER NOT NULL REFERENCES geofences(id) ON DELETE CASCADE,
    device_id INTEGER NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    event_type VARCHAR(20) NOT NULL CHECK (event_type IN ('enter', 'exit')), -- 事件类型：进入或离开
    location JSONB NOT NULL, -- 触发事件时的位置 {lat: float, lon: float}
    speed DOUBLE PRECISION, -- 触发时的速度
    triggered_at TIMESTAMPTZ DEFAULT NOW(), -- 触发时间
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_geofence_events_geofence_id ON geofence_events(geofence_id);
CREATE INDEX IF NOT EXISTS idx_geofence_events_device_id ON geofence_events(device_id);
CREATE INDEX IF NOT EXISTS idx_geofence_events_triggered_at ON geofence_events(triggered_at DESC);
CREATE INDEX IF NOT EXISTS idx_geofence_events_type ON geofence_events(event_type);

-- Convert to hypertable for time-series data (optional, for high volume)
-- SELECT create_hypertable('geofence_events', 'triggered_at', if_not_exists => TRUE);

-- ============================================
-- Device Geofence State Table - 设备围栏状态缓存表
-- 用于记录设备当前是否在围栏内，避免重复触发
-- ============================================
CREATE TABLE IF NOT EXISTS device_geofence_states (
    id SERIAL PRIMARY KEY,
    device_id INTEGER NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    geofence_id INTEGER NOT NULL REFERENCES geofences(id) ON DELETE CASCADE,
    is_inside BOOLEAN NOT NULL DEFAULT FALSE, -- 当前是否在围栏内
    last_event_type VARCHAR(20), -- 最后事件类型
    last_triggered_at TIMESTAMPTZ, -- 最后触发时间
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(device_id, geofence_id)
);

CREATE INDEX IF NOT EXISTS idx_device_geofence_states_device_id ON device_geofence_states(device_id);
CREATE INDEX IF NOT EXISTS idx_device_geofence_states_geofence_id ON device_geofence_states(geofence_id);

-- ============================================
-- Apply update trigger for updated_at
-- ============================================
CREATE TRIGGER update_geofences_updated_at BEFORE UPDATE ON geofences
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_device_geofence_states_updated_at BEFORE UPDATE ON device_geofence_states
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
