-- OpenFMS Database Migration Down: 006_vehicle
-- Rollback all changes from 006_vehicle

-- Drop triggers
DROP TRIGGER IF EXISTS update_vehicles_updated_at ON vehicles;

-- Drop indexes
DROP INDEX IF EXISTS idx_vgm_vehicle;
DROP INDEX IF EXISTS idx_vgm_group;
DROP INDEX IF EXISTS idx_dvh_device;
DROP INDEX IF EXISTS idx_dvh_vehicle;

-- Drop tables
DROP TABLE IF EXISTS device_vehicle_history;
DROP TABLE IF EXISTS vehicle_group_members;
DROP TABLE IF EXISTS vehicle_groups;

-- Remove columns from vehicles
ALTER TABLE vehicles 
    DROP COLUMN IF EXISTS plate_color,
    DROP COLUMN IF EXISTS vin,
    DROP COLUMN IF EXISTS engine_no,
    DROP COLUMN IF EXISTS owner_name,
    DROP COLUMN IF EXISTS owner_phone,
    DROP COLUMN IF EXISTS owner_idcard,
    DROP COLUMN IF EXISTS registration_no,
    DROP COLUMN IF EXISTS transport_no,
    DROP COLUMN IF EXISTS insurance_no,
    DROP COLUMN IF EXISTS insurance_expire,
    DROP COLUMN IF EXISTS device_id,
    DROP COLUMN IF EXISTS remark;
