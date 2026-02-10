-- OpenFMS Alarm Module Database Schema
-- Migration: 003_alarm

-- ============================================
-- Alarm Types and Status Enums
-- ============================================
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'alarm_type') THEN
        CREATE TYPE alarm_type AS ENUM (
            'GEOFENCE_ENTER',
            'GEOFENCE_EXIT',
            'OVERSPEED',
            'LOW_BATTERY',
            'OFFLINE',
            'SOS',
            'POWER_CUT',
            'VIBRATION',
            'ILLEGAL_MOVE'
        );
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'alarm_level') THEN
        CREATE TYPE alarm_level AS ENUM ('info', 'warning', 'critical');
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'alarm_status') THEN
        CREATE TYPE alarm_status AS ENUM ('unread', 'read', 'resolved');
    END IF;
END $$;

-- ============================================
-- Alarms Table
-- ============================================
CREATE TABLE IF NOT EXISTS alarms (
    id              SERIAL PRIMARY KEY,
    type            alarm_type NOT NULL,
    level           alarm_level NOT NULL DEFAULT 'warning',
    device_id       VARCHAR(32) NOT NULL,
    device_name     VARCHAR(100),
    title           VARCHAR(200) NOT NULL,
    content         TEXT,
    lat             DOUBLE PRECISION,
    lon             DOUBLE PRECISION,
    location_name   VARCHAR(200),
    speed           SMALLINT,
    speed_limit     SMALLINT,
    status          alarm_status NOT NULL DEFAULT 'unread',
    resolved_at     TIMESTAMPTZ,
    resolved_by     INTEGER REFERENCES users(id),
    resolve_note    TEXT,
    geofence_id     INTEGER REFERENCES geofences(id) ON DELETE SET NULL,
    geofence_name   VARCHAR(100),
    extras          JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_alarms_device_id ON alarms(device_id);
CREATE INDEX IF NOT EXISTS idx_alarms_type ON alarms(type);
CREATE INDEX IF NOT EXISTS idx_alarms_status ON alarms(status);
CREATE INDEX IF NOT EXISTS idx_alarms_level ON alarms(level);
CREATE INDEX IF NOT EXISTS idx_alarms_created_at ON alarms(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_alarms_device_created ON alarms(device_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_alarms_unread_critical ON alarms(status, level) WHERE status = 'unread' AND level = 'critical';

-- ============================================
-- Alarm Rules Table
-- ============================================
CREATE TABLE IF NOT EXISTS alarm_rules (
    id              SERIAL PRIMARY KEY,
    name            VARCHAR(100) NOT NULL,
    type            alarm_type NOT NULL,
    description     TEXT,
    conditions      JSONB NOT NULL DEFAULT '{}',
    all_devices     BOOLEAN NOT NULL DEFAULT true,
    device_ids      INTEGER[],
    notify_webhook  BOOLEAN NOT NULL DEFAULT false,
    webhook_url     VARCHAR(500),
    notify_ws       BOOLEAN NOT NULL DEFAULT true,
    notify_sound    BOOLEAN NOT NULL DEFAULT true,
    enabled         BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_alarm_rules_type ON alarm_rules(type);
CREATE INDEX IF NOT EXISTS idx_alarm_rules_enabled ON alarm_rules(enabled);

-- Insert default alarm rules
INSERT INTO alarm_rules (name, type, description, conditions, all_devices, enabled) VALUES
    ('超速报警', 'OVERSPEED', '车辆速度超过设定阈值时触发', '{"speed_limit": 120}', true, true),
    ('设备离线', 'OFFLINE', '设备超过10分钟未上报数据', '{"offline_minutes": 10}', true, true),
    ('紧急求救', 'SOS', '设备触发SOS紧急按钮', '{}', true, true),
    ('低电量报警', 'LOW_BATTERY', '设备电量低于20%', '{"battery_threshold": 20}', true, true),
    ('断电报警', 'POWER_CUT', '设备外部电源被切断', '{}', true, true)
ON CONFLICT DO NOTHING;

-- ============================================
-- Alarm Silence Table
-- ============================================
CREATE TABLE IF NOT EXISTS alarm_silences (
    id              SERIAL PRIMARY KEY,
    device_id       VARCHAR(32) NOT NULL,
    alarm_type      alarm_type NOT NULL,
    silence_until   TIMESTAMPTZ NOT NULL,
    reason          TEXT,
    created_by      INTEGER REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_alarm_silences_device ON alarm_silences(device_id, alarm_type);
CREATE INDEX IF NOT EXISTS idx_alarm_silences_until ON alarm_silences(silence_until);

-- ============================================
-- Alarm Stats Daily View
-- ============================================
CREATE OR REPLACE VIEW alarm_stats_daily AS
SELECT 
    DATE(created_at) as date,
    type,
    level,
    status,
    COUNT(*) as count
FROM alarms
WHERE created_at > NOW() - INTERVAL '90 days'
GROUP BY DATE(created_at), type, level, status;

-- ============================================
-- Update triggers
-- ============================================
DROP TRIGGER IF EXISTS update_alarms_updated_at ON alarms;
CREATE TRIGGER update_alarms_updated_at BEFORE UPDATE ON alarms
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_alarm_rules_updated_at ON alarm_rules;
CREATE TRIGGER update_alarm_rules_updated_at BEFORE UPDATE ON alarm_rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Comments
COMMENT ON TABLE alarms IS '报警记录表';
COMMENT ON TABLE alarm_rules IS '报警规则配置表';
COMMENT ON TABLE alarm_silences IS '报警静默规则表';
