-- OpenFMS Database Migration Down: 002_geofence
-- Rollback all changes from 002_geofence

-- Drop triggers
DROP TRIGGER IF EXISTS update_device_geofence_states_updated_at ON device_geofence_states;

-- Drop tables
DROP TABLE IF EXISTS device_geofence_states;
DROP TABLE IF EXISTS geofence_events;

-- Rename back if needed
ALTER TABLE IF EXISTS geofence_devices RENAME TO device_geofences;

-- Remove constraints from geofences
ALTER TABLE geofences 
    DROP CONSTRAINT IF EXISTS chk_geofences_type,
    DROP CONSTRAINT IF EXISTS chk_geofences_alert_type;

-- Remove columns from geofences
ALTER TABLE geofences 
    DROP COLUMN IF EXISTS description,
    DROP COLUMN IF EXISTS user_id;

-- Drop indexes
DROP INDEX IF EXISTS idx_geofences_status;
DROP INDEX IF EXISTS idx_geofences_user_id;
DROP INDEX IF EXISTS idx_geofences_deleted_at;
DROP INDEX IF EXISTS idx_geofence_devices_geofence_id;
DROP INDEX IF EXISTS idx_geofence_devices_device_id;
