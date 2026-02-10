-- OpenFMS Vehicle Module Database Schema
-- Migration: 006_vehicle

-- ============================================
-- Vehicles Table Enhancement
-- ============================================
-- Note: Base vehicles table exists from 001_init
-- Enhance with additional columns

ALTER TABLE vehicles 
    ADD COLUMN IF NOT EXISTS plate_color VARCHAR(10) DEFAULT '蓝色',
    ADD COLUMN IF NOT EXISTS vin VARCHAR(17),
    ADD COLUMN IF NOT EXISTS engine_no VARCHAR(30),
    ADD COLUMN IF NOT EXISTS owner_name VARCHAR(100),
    ADD COLUMN IF NOT EXISTS owner_phone VARCHAR(20),
    ADD COLUMN IF NOT EXISTS owner_idcard VARCHAR(18),
    ADD COLUMN IF NOT EXISTS registration_no VARCHAR(30),
    ADD COLUMN IF NOT EXISTS transport_no VARCHAR(30),
    ADD COLUMN IF NOT EXISTS insurance_no VARCHAR(30),
    ADD COLUMN IF NOT EXISTS insurance_expire DATE,
    ADD COLUMN IF NOT EXISTS device_id VARCHAR(32),
    ADD COLUMN IF NOT EXISTS remark TEXT;

-- Update existing vehicle type column if needed
ALTER TABLE vehicles 
    ALTER COLUMN type TYPE VARCHAR(50);

-- ============================================
-- Vehicle Groups Table
-- ============================================
CREATE TABLE IF NOT EXISTS vehicle_groups (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(50) NOT NULL,
    color       VARCHAR(20) DEFAULT '#1890ff',
    description TEXT,
    created_by  INTEGER REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================
-- Vehicle Group Members Table
-- ============================================
CREATE TABLE IF NOT EXISTS vehicle_group_members (
    id          SERIAL PRIMARY KEY,
    vehicle_id  INTEGER NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    group_id    INTEGER NOT NULL REFERENCES vehicle_groups(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(vehicle_id, group_id)
);

CREATE INDEX IF NOT EXISTS idx_vgm_vehicle ON vehicle_group_members(vehicle_id);
CREATE INDEX IF NOT EXISTS idx_vgm_group ON vehicle_group_members(group_id);

-- ============================================
-- Device Vehicle History Table
-- ============================================
CREATE TABLE IF NOT EXISTS device_vehicle_history (
    id          SERIAL PRIMARY KEY,
    device_id   VARCHAR(32) NOT NULL,
    vehicle_id  INTEGER NOT NULL,
    action      VARCHAR(20) NOT NULL,
    operated_by INTEGER REFERENCES users(id),
    operated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    remark      TEXT
);

CREATE INDEX IF NOT EXISTS idx_dvh_device ON device_vehicle_history(device_id);
CREATE INDEX IF NOT EXISTS idx_dvh_vehicle ON device_vehicle_history(vehicle_id);

-- ============================================
-- Update trigger for vehicles
-- ============================================
-- Trigger already exists from 001_init, ensure it's there
DROP TRIGGER IF EXISTS update_vehicles_updated_at ON vehicles;
CREATE TRIGGER update_vehicles_updated_at BEFORE UPDATE ON vehicles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Comments
COMMENT ON TABLE vehicles IS '车辆信息表';
COMMENT ON TABLE vehicle_groups IS '车辆分组表';
COMMENT ON TABLE vehicle_group_members IS '车辆分组关联表';
COMMENT ON TABLE device_vehicle_history IS '设备车辆绑定历史';
