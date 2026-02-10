-- OpenFMS Report Module Database Schema
-- Migration: 005_report

-- ============================================
-- Daily Stats Table
-- ============================================
CREATE TABLE IF NOT EXISTS daily_stats (
    id          SERIAL PRIMARY KEY,
    date        DATE NOT NULL UNIQUE,
    total_devices INTEGER DEFAULT 0,
    online_devices INTEGER DEFAULT 0,
    offline_devices INTEGER DEFAULT 0,
    total_positions BIGINT DEFAULT 0,
    avg_positions_per_device INTEGER DEFAULT 0,
    total_alarms INTEGER DEFAULT 0,
    critical_alarms INTEGER DEFAULT 0,
    warning_alarms INTEGER DEFAULT 0,
    info_alarms INTEGER DEFAULT 0,
    resolved_alarms INTEGER DEFAULT 0,
    total_mileage DECIMAL(12, 2) DEFAULT 0,
    avg_mileage_per_device DECIMAL(10, 2) DEFAULT 0,
    harsh_acceleration INTEGER DEFAULT 0,
    harsh_braking INTEGER DEFAULT 0,
    harsh_turning INTEGER DEFAULT 0,
    speeding_count INTEGER DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_daily_stats_date ON daily_stats(date);

-- ============================================
-- Device Daily Stats Table
-- ============================================
CREATE TABLE IF NOT EXISTS device_daily_stats (
    id          SERIAL PRIMARY KEY,
    device_id   VARCHAR(32) NOT NULL,
    date        DATE NOT NULL,
    start_mileage DECIMAL(10, 2),
    end_mileage DECIMAL(10, 2),
    daily_mileage DECIMAL(10, 2) DEFAULT 0,
    driving_duration INTEGER DEFAULT 0,
    idle_duration INTEGER DEFAULT 0,
    stop_count INTEGER DEFAULT 0,
    max_speed SMALLINT DEFAULT 0,
    avg_speed SMALLINT DEFAULT 0,
    alarm_count INTEGER DEFAULT 0,
    position_count INTEGER DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(device_id, date)
);

CREATE INDEX IF NOT EXISTS idx_device_daily_stats_device ON device_daily_stats(device_id);
CREATE INDEX IF NOT EXISTS idx_device_daily_stats_date ON device_daily_stats(date);
CREATE INDEX IF NOT EXISTS idx_device_daily_stats_device_date ON device_daily_stats(device_id, date);

-- ============================================
-- Stop Points Table
-- ============================================
CREATE TABLE IF NOT EXISTS stop_points (
    id          SERIAL PRIMARY KEY,
    device_id   VARCHAR(32) NOT NULL,
    start_time  TIMESTAMPTZ NOT NULL,
    end_time    TIMESTAMPTZ,
    duration    INTEGER,
    lat         DOUBLE PRECISION NOT NULL,
    lon         DOUBLE PRECISION NOT NULL,
    location_name VARCHAR(200),
    stop_type   VARCHAR(20) DEFAULT 'normal',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_stop_points_device ON stop_points(device_id);
CREATE INDEX IF NOT EXISTS idx_stop_points_time ON stop_points(start_time, end_time);
CREATE INDEX IF NOT EXISTS idx_stop_points_device_time ON stop_points(device_id, start_time);

-- ============================================
-- Driving Events Table
-- ============================================
CREATE TABLE IF NOT EXISTS driving_events (
    id          SERIAL PRIMARY KEY,
    device_id   VARCHAR(32) NOT NULL,
    event_type  VARCHAR(50) NOT NULL,
    event_time  TIMESTAMPTZ NOT NULL,
    lat         DOUBLE PRECISION,
    lon         DOUBLE PRECISION,
    speed       SMALLINT,
    limit_speed SMALLINT,
    value       DECIMAL(5, 2),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_driving_events_device ON driving_events(device_id);
CREATE INDEX IF NOT EXISTS idx_driving_events_type ON driving_events(event_type);
CREATE INDEX IF NOT EXISTS idx_driving_events_time ON driving_events(event_time);

-- ============================================
-- Report Jobs Table
-- ============================================
CREATE TABLE IF NOT EXISTS report_jobs (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    report_type VARCHAR(50) NOT NULL,
    device_ids  VARCHAR(32)[],
    start_date  DATE NOT NULL,
    end_date    DATE NOT NULL,
    status      VARCHAR(20) DEFAULT 'pending',
    progress    INTEGER DEFAULT 0,
    file_url    VARCHAR(500),
    file_size   BIGINT,
    created_by  INTEGER REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_report_jobs_status ON report_jobs(status);
CREATE INDEX IF NOT EXISTS idx_report_jobs_created_by ON report_jobs(created_by);

-- Insert default stats for today
INSERT INTO daily_stats (date, total_devices, online_devices) 
SELECT CURRENT_DATE, 0, 0
WHERE NOT EXISTS (SELECT 1 FROM daily_stats WHERE date = CURRENT_DATE);

-- Comments
COMMENT ON TABLE daily_stats IS '每日统计表';
COMMENT ON TABLE device_daily_stats IS '设备每日统计表';
COMMENT ON TABLE stop_points IS '停留点表';
COMMENT ON TABLE driving_events IS '驾驶行为事件表';
COMMENT ON TABLE report_jobs IS '报表任务表';
