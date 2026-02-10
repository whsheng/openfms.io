-- 报表统计模块数据库表结构
-- 创建于: 2026-02-04

-- 日统计表
CREATE TABLE daily_stats (
    id          SERIAL PRIMARY KEY,
    date        DATE NOT NULL UNIQUE,
    
    -- 设备统计
    total_devices INTEGER DEFAULT 0,
    online_devices INTEGER DEFAULT 0,
    offline_devices INTEGER DEFAULT 0,
    
    -- 位置统计
    total_positions BIGINT DEFAULT 0,
    avg_positions_per_device INTEGER DEFAULT 0,
    
    -- 报警统计
    total_alarms INTEGER DEFAULT 0,
    critical_alarms INTEGER DEFAULT 0,
    warning_alarms INTEGER DEFAULT 0,
    info_alarms INTEGER DEFAULT 0,
    resolved_alarms INTEGER DEFAULT 0,
    
    -- 里程统计 (公里)
    total_mileage DECIMAL(12, 2) DEFAULT 0,
    avg_mileage_per_device DECIMAL(10, 2) DEFAULT 0,
    
    -- 驾驶行为
    harsh_acceleration INTEGER DEFAULT 0,  -- 急加速
    harsh_braking INTEGER DEFAULT 0,       -- 急刹车
    harsh_turning INTEGER DEFAULT 0,       -- 急转弯
    speeding_count INTEGER DEFAULT 0,      -- 超速次数
    
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_daily_stats_date ON daily_stats(date);

-- 设备日统计表
CREATE TABLE device_daily_stats (
    id          SERIAL PRIMARY KEY,
    device_id   VARCHAR(20) NOT NULL REFERENCES devices(sim_no) ON DELETE CASCADE,
    date        DATE NOT NULL,
    
    -- 里程
    start_mileage DECIMAL(10, 2),      -- 起始里程
    end_mileage DECIMAL(10, 2),        -- 结束里程
    daily_mileage DECIMAL(10, 2) DEFAULT 0,  -- 日里程
    
    -- 行驶统计
    driving_duration INTEGER DEFAULT 0,     -- 行驶时长(秒)
    idle_duration INTEGER DEFAULT 0,        -- 怠速时长(秒)
    stop_count INTEGER DEFAULT 0,           -- 停车次数
    
    -- 速度统计
    max_speed SMALLINT DEFAULT 0,
    avg_speed SMALLINT DEFAULT 0,
    
    -- 报警统计
    alarm_count INTEGER DEFAULT 0,
    
    -- 位置统计
    position_count INTEGER DEFAULT 0,
    
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(device_id, date)
);

CREATE INDEX idx_device_daily_stats_device ON device_daily_stats(device_id);
CREATE INDEX idx_device_daily_stats_date ON device_daily_stats(date);
CREATE INDEX idx_device_daily_stats_device_date ON device_daily_stats(device_id, date);

-- 停留点表
CREATE TABLE stop_points (
    id          SERIAL PRIMARY KEY,
    device_id   VARCHAR(20) NOT NULL REFERENCES devices(sim_no) ON DELETE CASCADE,
    
    start_time  TIMESTAMPTZ NOT NULL,
    end_time    TIMESTAMPTZ,
    duration    INTEGER,  -- 停留时长(秒)
    
    lat         DOUBLE PRECISION NOT NULL,
    lon         DOUBLE PRECISION NOT NULL,
    location_name VARCHAR(200),
    
    -- 停留类型
    stop_type   VARCHAR(20) DEFAULT 'normal',  -- normal, parking, rest
    
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_stop_points_device ON stop_points(device_id);
CREATE INDEX idx_stop_points_time ON stop_points(start_time, end_time);
CREATE INDEX idx_stop_points_device_time ON stop_points(device_id, start_time);

-- 驾驶行为事件表
CREATE TABLE driving_events (
    id          SERIAL PRIMARY KEY,
    device_id   VARCHAR(20) NOT NULL REFERENCES devices(sim_no) ON DELETE CASCADE,
    
    event_type  VARCHAR(50) NOT NULL,  -- harsh_acceleration, harsh_braking, harsh_turning, speeding
    event_time  TIMESTAMPTZ NOT NULL,
    
    lat         DOUBLE PRECISION,
    lon         DOUBLE PRECISION,
    
    speed       SMALLINT,           -- 当前速度
    limit_speed SMALLINT,           -- 限速
    value       DECIMAL(5, 2),      -- 事件值(如加速度)
    
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_driving_events_device ON driving_events(device_id);
CREATE INDEX idx_driving_events_type ON driving_events(event_type);
CREATE INDEX idx_driving_events_time ON driving_events(event_time);

-- 报表任务表
CREATE TABLE report_jobs (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    report_type VARCHAR(50) NOT NULL,  -- mileage, stop, alarm, driving
    
    -- 查询条件
    device_ids  VARCHAR(20)[],
    start_date  DATE NOT NULL,
    end_date    DATE NOT NULL,
    
    -- 任务状态
    status      VARCHAR(20) DEFAULT 'pending',  -- pending, running, completed, failed
    progress    INTEGER DEFAULT 0,  -- 0-100
    
    -- 结果
    file_url    VARCHAR(500),
    file_size   BIGINT,
    
    created_by  INTEGER REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_report_jobs_status ON report_jobs(status);
CREATE INDEX idx_report_jobs_created_by ON report_jobs(created_by);

-- 插入默认统计任务
INSERT INTO daily_stats (date, total_devices, online_devices) 
SELECT CURRENT_DATE, 0, 0
WHERE NOT EXISTS (SELECT 1 FROM daily_stats WHERE date = CURRENT_DATE);
