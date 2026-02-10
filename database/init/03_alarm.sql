-- 报警模块数据库表结构
-- 创建于: 2026-02-04

-- 报警类型枚举
CREATE TYPE alarm_type AS ENUM (
    'GEOFENCE_ENTER',   -- 进入围栏
    'GEOFENCE_EXIT',    -- 离开围栏
    'OVERSPEED',        -- 超速
    'LOW_BATTERY',      -- 低电量
    'OFFLINE',          -- 设备离线
    'SOS',              -- 紧急求救
    'POWER_CUT',        -- 断电报警
    'VIBRATION',        -- 震动报警
    'ILLEGAL_MOVE'      -- 非法移动
);

-- 报警级别枚举
CREATE TYPE alarm_level AS ENUM (
    'info',      -- 信息
    'warning',   -- 警告
    'critical'   -- 严重
);

-- 报警状态枚举
CREATE TYPE alarm_status AS ENUM (
    'unread',    -- 未读
    'read',      -- 已读
    'resolved'   -- 已处理
);

-- 报警主表
CREATE TABLE alarms (
    id              SERIAL PRIMARY KEY,
    type            alarm_type NOT NULL,
    level           alarm_level NOT NULL DEFAULT 'warning',
    device_id       VARCHAR(20) NOT NULL REFERENCES devices(sim_no) ON DELETE CASCADE,
    device_name     VARCHAR(100),  -- 冗余存储，方便查询
    
    -- 报警内容
    title           VARCHAR(200) NOT NULL,
    content         TEXT,
    
    -- 位置信息
    lat             DOUBLE PRECISION,
    lon             DOUBLE PRECISION,
    location_name   VARCHAR(200),  -- 位置描述
    
    -- 速度信息（超速报警用）
    speed           SMALLINT,      -- 当前速度
    speed_limit     SMALLINT,      -- 限速
    
    -- 处理状态
    status          alarm_status NOT NULL DEFAULT 'unread',
    
    -- 处理信息
    resolved_at     TIMESTAMPTZ,
    resolved_by     INTEGER REFERENCES users(id),
    resolve_note    TEXT,
    
    -- 关联围栏（围栏报警用）
    geofence_id     INTEGER REFERENCES geofences(id) ON DELETE SET NULL,
    geofence_name   VARCHAR(100),
    
    -- 扩展字段
    extras          JSONB,         -- 额外数据
    
    -- 时间戳
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 创建索引
CREATE INDEX idx_alarms_device_id ON alarms(device_id);
CREATE INDEX idx_alarms_type ON alarms(type);
CREATE INDEX idx_alarms_status ON alarms(status);
CREATE INDEX idx_alarms_level ON alarms(level);
CREATE INDEX idx_alarms_created_at ON alarms(created_at DESC);
CREATE INDEX idx_alarms_device_created ON alarms(device_id, created_at DESC);

-- 复合索引：查询未读的重要报警
CREATE INDEX idx_alarms_unread_critical ON alarms(status, level) WHERE status = 'unread' AND level = 'critical';

-- 报警规则表
CREATE TABLE alarm_rules (
    id              SERIAL PRIMARY KEY,
    
    -- 基本信息
    name            VARCHAR(100) NOT NULL,
    type            alarm_type NOT NULL,
    description     TEXT,
    
    -- 规则条件（JSON格式，灵活配置）
    -- 例如超速: {"speed_limit": 120}
    -- 例如围栏: {"geofence_ids": [1, 2, 3]}
    -- 例如离线: {"offline_minutes": 10}
    conditions      JSONB NOT NULL DEFAULT '{}',
    
    -- 生效范围
    all_devices     BOOLEAN NOT NULL DEFAULT true,  -- 是否对所有设备生效
    device_ids      INTEGER[],  -- 指定设备ID列表（all_devices=false时使用）
    
    -- 通知设置
    notify_webhook  BOOLEAN NOT NULL DEFAULT false,
    webhook_url     VARCHAR(500),
    notify_ws       BOOLEAN NOT NULL DEFAULT true,   -- WebSocket推送
    notify_sound    BOOLEAN NOT NULL DEFAULT true,   -- 声音提醒
    
    -- 状态
    enabled         BOOLEAN NOT NULL DEFAULT true,
    
    -- 时间戳
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 创建索引
CREATE INDEX idx_alarm_rules_type ON alarm_rules(type);
CREATE INDEX idx_alarm_rules_enabled ON alarm_rules(enabled);

-- 插入默认报警规则
INSERT INTO alarm_rules (name, type, description, conditions, all_devices, enabled) VALUES
    ('超速报警', 'OVERSPEED', '车辆速度超过设定阈值时触发', '{"speed_limit": 120}', true, true),
    ('设备离线', 'OFFLINE', '设备超过10分钟未上报数据', '{"offline_minutes": 10}', true, true),
    ('紧急求救', 'SOS', '设备触发SOS紧急按钮', '{}', true, true),
    ('低电量报警', 'LOW_BATTERY', '设备电量低于20%', '{"battery_threshold": 20}', true, true),
    ('断电报警', 'POWER_CUT', '设备外部电源被切断', '{}', true, true);

-- 报警静默规则（防止报警风暴）
CREATE TABLE alarm_silences (
    id              SERIAL PRIMARY KEY,
    device_id       VARCHAR(20) NOT NULL REFERENCES devices(sim_no) ON DELETE CASCADE,
    alarm_type      alarm_type NOT NULL,
    
    -- 静默时间段
    silence_until   TIMESTAMPTZ NOT NULL,
    
    -- 原因
    reason          TEXT,
    created_by      INTEGER REFERENCES users(id),
    
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_alarm_silences_device ON alarm_silences(device_id, alarm_type);
CREATE INDEX idx_alarm_silences_until ON alarm_silences(silence_until);

-- 报警统计视图（按天统计）
CREATE VIEW alarm_stats_daily AS
SELECT 
    DATE(created_at) as date,
    type,
    level,
    status,
    COUNT(*) as count
FROM alarms
WHERE created_at > NOW() - INTERVAL '90 days'
GROUP BY DATE(created_at), type, level, status;

-- 更新时间戳触发器
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_alarms_updated_at BEFORE UPDATE ON alarms
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_alarm_rules_updated_at BEFORE UPDATE ON alarm_rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 注释
COMMENT ON TABLE alarms IS '报警记录表';
COMMENT ON TABLE alarm_rules IS '报警规则配置表';
COMMENT ON TABLE alarm_silences IS '报警静默规则表';
