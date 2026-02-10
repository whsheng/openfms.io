-- 油耗管理模块数据库表结构
-- 创建于: 2026-02-04

-- 加油记录表
CREATE TABLE fuel_records (
    id              SERIAL PRIMARY KEY,
    vehicle_id      INTEGER NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    device_id       VARCHAR(20) REFERENCES devices(sim_no),
    
    -- 加油信息
    fuel_time       TIMESTAMPTZ NOT NULL,
    fuel_type       VARCHAR(20) DEFAULT '汽油',  -- 汽油、柴油、电
    fuel_volume     DECIMAL(8, 2) NOT NULL,      -- 加油量（升）
    fuel_price      DECIMAL(6, 2),               -- 单价（元/升）
    total_cost      DECIMAL(10, 2),              -- 总金额
    
    -- 里程信息
    current_mileage DECIMAL(10, 2) NOT NULL,     -- 当前里程
    last_mileage    DECIMAL(10, 2),              -- 上次里程
    trip_distance   DECIMAL(10, 2),              -- 行驶里程
    
    -- 油耗计算
    fuel_consumption DECIMAL(5, 2),              -- 百公里油耗
    
    -- 加油站信息
    station_name    VARCHAR(100),
    station_location VARCHAR(200),
    lat             DOUBLE PRECISION,
    lon             DOUBLE PRECISION,
    
    -- 备注
    remark          TEXT,
    
    -- 操作人
    created_by      INTEGER REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_fuel_records_vehicle ON fuel_records(vehicle_id);
CREATE INDEX idx_fuel_records_time ON fuel_records(fuel_time);
CREATE INDEX idx_fuel_records_device ON fuel_records(device_id);

-- 油耗统计表（按日）
CREATE TABLE fuel_daily_stats (
    id              SERIAL PRIMARY KEY,
    vehicle_id      INTEGER NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    date            DATE NOT NULL,
    
    -- 行驶统计
    start_mileage   DECIMAL(10, 2),
    end_mileage     DECIMAL(10, 2),
    daily_mileage   DECIMAL(10, 2),
    
    -- 油耗统计
    fuel_volume     DECIMAL(8, 2),               -- 当日加油量
    fuel_cost       DECIMAL(10, 2),              -- 当日油费
    avg_consumption DECIMAL(5, 2),               -- 平均油耗
    
    -- 成本统计
    cost_per_km     DECIMAL(6, 2),               -- 每公里成本
    
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(vehicle_id, date)
);

CREATE INDEX idx_fuel_daily_vehicle ON fuel_daily_stats(vehicle_id);
CREATE INDEX idx_fuel_daily_date ON fuel_daily_stats(date);

-- 油耗异常告警表
CREATE TABLE fuel_anomalies (
    id              SERIAL PRIMARY KEY,
    vehicle_id      INTEGER NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    
    anomaly_type    VARCHAR(50) NOT NULL,        -- sudden_drop, abnormal_high, etc.
    anomaly_time    TIMESTAMPTZ NOT NULL,
    
    -- 异常数据
    expected_consumption DECIMAL(5, 2),          -- 预期油耗
    actual_consumption   DECIMAL(5, 2),          -- 实际油耗
    
    -- 位置信息
    lat             DOUBLE PRECISION,
    lon             DOUBLE PRECISION,
    
    -- 处理状态
    status          VARCHAR(20) DEFAULT 'unprocessed',  -- unprocessed, confirmed, false_positive
    remark          TEXT,
    
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_fuel_anomalies_vehicle ON fuel_anomalies(vehicle_id);
CREATE INDEX idx_fuel_anomalies_time ON fuel_anomalies(anomaly_time);

-- 更新时间戳触发器
CREATE TRIGGER update_fuel_records_updated_at BEFORE UPDATE ON fuel_records
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_fuel_anomalies_updated_at BEFORE UPDATE ON fuel_anomalies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 注释
COMMENT ON TABLE fuel_records IS '加油记录表';
COMMENT ON TABLE fuel_daily_stats IS '油耗日统计表';
COMMENT ON TABLE fuel_anomalies IS '油耗异常告警表';
