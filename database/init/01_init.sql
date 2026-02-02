-- OpenFMS Database Initialization
-- This script initializes the TimescaleDB and creates necessary tables

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS timescaledb;
CREATE EXTENSION IF NOT EXISTS postgis;

-- Create schema
CREATE SCHEMA IF NOT EXISTS openfms;

-- ============================================
-- Users Table
-- ============================================
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    email VARCHAR(100) UNIQUE,
    phone VARCHAR(20),
    role VARCHAR(20) DEFAULT 'user', -- admin, manager, user
    status INTEGER DEFAULT 1, -- 0: inactive, 1: active
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Create index on username
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at) WHERE deleted_at IS NULL;

-- ============================================
-- Vehicles Table
-- ============================================
CREATE TABLE IF NOT EXISTS vehicles (
    id SERIAL PRIMARY KEY,
    plate_number VARCHAR(20) UNIQUE NOT NULL,
    type VARCHAR(50), -- truck, van, car, etc.
    brand VARCHAR(50),
    model VARCHAR(50),
    color VARCHAR(20),
    year INTEGER,
    status INTEGER DEFAULT 1, -- 0: inactive, 1: active
    organization VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_vehicles_plate ON vehicles(plate_number);

-- ============================================
-- Devices Table
-- ============================================
CREATE TABLE IF NOT EXISTS devices (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(32) UNIQUE NOT NULL, -- SIM number / Device ID
    name VARCHAR(100),
    protocol VARCHAR(20) DEFAULT 'JT808', -- JT808, GT06, Wialon, etc.
    vehicle_id INTEGER REFERENCES vehicles(id) ON DELETE SET NULL,
    status INTEGER DEFAULT 0, -- 0: inactive, 1: active
    last_online TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_devices_device_id ON devices(device_id);
CREATE INDEX IF NOT EXISTS idx_devices_vehicle_id ON devices(vehicle_id);

-- ============================================
-- Vehicle Positions (TimescaleDB Hypertable)
-- ============================================
CREATE TABLE IF NOT EXISTS vehicle_positions (
    time TIMESTAMPTZ NOT NULL,
    device_id VARCHAR(32) NOT NULL,
    lat DOUBLE PRECISION NOT NULL,
    lon DOUBLE PRECISION NOT NULL,
    speed SMALLINT, -- km/h
    angle SMALLINT, -- 0-360 degrees
    flags INTEGER, -- status bits (ACC, alarm, etc.)
    extras JSONB, -- additional data: fuel, temperature, etc.
    PRIMARY KEY (time, device_id)
);

-- Convert to hypertable
SELECT create_hypertable('vehicle_positions', 'time', 
    chunk_time_interval => INTERVAL '7 days',
    if_not_exists => TRUE
);

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_positions_device_time 
    ON vehicle_positions (device_id, time DESC);

CREATE INDEX IF NOT EXISTS idx_positions_location 
    ON vehicle_positions (lat, lon, time DESC);

-- Enable compression for older data
ALTER TABLE vehicle_positions SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'device_id',
    timescaledb.compress_orderby = 'time DESC'
);

-- Add compression policy: compress data older than 7 days
SELECT add_compression_policy('vehicle_positions', INTERVAL '7 days', if_not_exists => TRUE);

-- Add retention policy: delete data older than 90 days (adjust as needed)
-- SELECT add_retention_policy('vehicle_positions', INTERVAL '90 days', if_not_exists => TRUE);

-- ============================================
-- Geofences Table
-- ============================================
CREATE TABLE IF NOT EXISTS geofences (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(20) NOT NULL, -- circle, polygon
    coordinates JSONB NOT NULL, -- {center: {lat, lon}, radius} or {points: [...]}
    alert_type VARCHAR(20) DEFAULT 'both', -- enter, exit, both
    status INTEGER DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- ============================================
-- Device Geofence Association
-- ============================================
CREATE TABLE IF NOT EXISTS device_geofences (
    id SERIAL PRIMARY KEY,
    device_id INTEGER REFERENCES devices(id) ON DELETE CASCADE,
    geofence_id INTEGER REFERENCES geofences(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(device_id, geofence_id)
);

-- ============================================
-- Alerts Table
-- ============================================
CREATE TABLE IF NOT EXISTS alerts (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(32) NOT NULL,
    type VARCHAR(50) NOT NULL, -- geofence, speed, etc.
    severity VARCHAR(20) DEFAULT 'info', -- info, warning, critical
    message TEXT,
    lat DOUBLE PRECISION,
    lon DOUBLE PRECISION,
    acknowledged BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_alerts_device_id ON alerts(device_id);
CREATE INDEX IF NOT EXISTS idx_alerts_created_at ON alerts(created_at DESC);

-- ============================================
-- Insert default admin user
-- Password: admin (bcrypt hash)
-- ============================================
INSERT INTO users (username, password, email, role, status)
VALUES (
    'admin', 
    '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqQzBZN0UfGNEsKYGsGvJz1eKx3.K', -- 'admin'
    'admin@openfms.local',
    'admin',
    1
)
ON CONFLICT (username) DO NOTHING;

-- ============================================
-- Create update trigger for updated_at
-- ============================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply trigger to all tables with updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_vehicles_updated_at BEFORE UPDATE ON vehicles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_devices_updated_at BEFORE UPDATE ON devices
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_geofences_updated_at BEFORE UPDATE ON geofences
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
