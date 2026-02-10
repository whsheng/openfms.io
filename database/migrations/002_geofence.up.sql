-- OpenFMS Geofence Module Database Schema
-- Migration: 002_geofence

-- ============================================
-- Geofences Table - Enhanced
-- ============================================
-- Note: Base geofences table created in 001_init
-- Here we add enhanced columns if not exists

ALTER TABLE geofences 
    ADD COLUMN IF NOT EXISTS description TEXT,
    ADD COLUMN IF NOT EXISTS user_id INTEGER REFERENCES users(id) ON DELETE SET NULL;

-- Add constraints if not exists
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'chk_geofences_type' AND table_name = 'geofences'
    ) THEN
        ALTER TABLE geofences ADD CONSTRAINT chk_geofences_type 
            CHECK (type IN ('circle', 'polygon'));
    END IF;
    
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'chk_geofences_alert_type' AND table_name = 'geofences'
    ) THEN
        ALTER TABLE geofences ADD CONSTRAINT chk_geofences_alert_type 
            CHECK (alert_type IN ('enter', 'exit', 'both'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_geofences_status ON geofences(status) WHERE status = 1;
CREATE INDEX IF NOT EXISTS idx_geofences_user_id ON geofences(user_id);
CREATE INDEX IF NOT EXISTS idx_geofences_deleted_at ON geofences(deleted_at) WHERE deleted_at IS NULL;

-- ============================================
-- Geofence Devices Table - Rename from device_geofences
-- ============================================
-- Note: device_geofences already exists from 001_init, enhance it
ALTER TABLE device_geofences 
    DROP CONSTRAINT IF EXISTS device_geofences_device_id_fkey,
    ADD CONSTRAINT device_geofences_device_id_fkey 
        FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE;

-- Rename to match model naming
ALTER TABLE IF EXISTS device_geofences RENAME TO geofence_devices;

CREATE INDEX IF NOT EXISTS idx_geofence_devices_geofence_id ON geofence_devices(geofence_id);
CREATE INDEX IF NOT EXISTS idx_geofence_devices_device_id ON geofence_devices(device_id);

-- ============================================
-- Geofence Events Table
-- ============================================
CREATE TABLE IF NOT EXISTS geofence_events (
    id SERIAL PRIMARY KEY,
    geofence_id INTEGER NOT NULL REFERENCES geofences(id) ON DELETE CASCADE,
    device_id INTEGER NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    event_type VARCHAR(20) NOT NULL CHECK (event_type IN ('enter', 'exit')),
    location JSONB NOT NULL,
    speed DOUBLE PRECISION,
    triggered_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_geofence_events_geofence_id ON geofence_events(geofence_id);
CREATE INDEX IF NOT EXISTS idx_geofence_events_device_id ON geofence_events(device_id);
CREATE INDEX IF NOT EXISTS idx_geofence_events_triggered_at ON geofence_events(triggered_at DESC);
CREATE INDEX IF NOT EXISTS idx_geofence_events_type ON geofence_events(event_type);

-- ============================================
-- Device Geofence State Table
-- ============================================
CREATE TABLE IF NOT EXISTS device_geofence_states (
    id SERIAL PRIMARY KEY,
    device_id INTEGER NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    geofence_id INTEGER NOT NULL REFERENCES geofences(id) ON DELETE CASCADE,
    is_inside BOOLEAN NOT NULL DEFAULT FALSE,
    last_event_type VARCHAR(20),
    last_triggered_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(device_id, geofence_id)
);

CREATE INDEX IF NOT EXISTS idx_device_geofence_states_device_id ON device_geofence_states(device_id);
CREATE INDEX IF NOT EXISTS idx_device_geofence_states_geofence_id ON device_geofence_states(geofence_id);

-- ============================================
-- Apply update trigger for updated_at
-- ============================================
DROP TRIGGER IF EXISTS update_device_geofence_states_updated_at ON device_geofence_states;
CREATE TRIGGER update_device_geofence_states_updated_at BEFORE UPDATE ON device_geofence_states
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
