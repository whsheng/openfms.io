-- OpenFMS Database Migration Down: 005_report
-- Rollback all changes from 005_report

-- Drop indexes
DROP INDEX IF EXISTS idx_daily_stats_date;
DROP INDEX IF EXISTS idx_device_daily_stats_device;
DROP INDEX IF EXISTS idx_device_daily_stats_date;
DROP INDEX IF EXISTS idx_device_daily_stats_device_date;
DROP INDEX IF EXISTS idx_stop_points_device;
DROP INDEX IF EXISTS idx_stop_points_time;
DROP INDEX IF EXISTS idx_stop_points_device_time;
DROP INDEX IF EXISTS idx_driving_events_device;
DROP INDEX IF EXISTS idx_driving_events_type;
DROP INDEX IF EXISTS idx_driving_events_time;
DROP INDEX IF EXISTS idx_report_jobs_status;
DROP INDEX IF EXISTS idx_report_jobs_created_by;

-- Drop tables
DROP TABLE IF EXISTS report_jobs;
DROP TABLE IF EXISTS driving_events;
DROP TABLE IF EXISTS stop_points;
DROP TABLE IF EXISTS device_daily_stats;
DROP TABLE IF EXISTS daily_stats;
