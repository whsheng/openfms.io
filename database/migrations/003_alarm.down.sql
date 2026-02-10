-- OpenFMS Database Migration Down: 003_alarm
-- Rollback all changes from 003_alarm

-- Drop triggers
DROP TRIGGER IF EXISTS update_alarms_updated_at ON alarms;
DROP TRIGGER IF EXISTS update_alarm_rules_updated_at ON alarm_rules;

-- Drop view
DROP VIEW IF EXISTS alarm_stats_daily;

-- Drop tables
DROP TABLE IF EXISTS alarm_silences;
DROP TABLE IF EXISTS alarm_rules;
DROP TABLE IF EXISTS alarms;

-- Drop types
DROP TYPE IF EXISTS alarm_status;
DROP TYPE IF EXISTS alarm_level;
DROP TYPE IF EXISTS alarm_type;
