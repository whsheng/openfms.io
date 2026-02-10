-- Webhook 模块数据库表结构
-- 创建于: 2026-02-04

-- Webhook 事件类型枚举
CREATE TYPE webhook_event_type AS ENUM (
    'alarm.created',           -- 报警创建
    'alarm.resolved',          -- 报警已处理
    'geofence.enter',          -- 进入围栏
    'geofence.exit',           -- 离开围栏
    'device.online',           -- 设备上线
    'device.offline',          -- 设备离线
    'device.position',         -- 位置更新
    'device.command_result',   -- 指令执行结果
    'vehicle.created',         -- 车辆创建
    'vehicle.updated',         -- 车辆更新
    'all'                      -- 所有事件
);

-- Webhook 状态枚举
CREATE TYPE webhook_status AS ENUM (
    'active',      -- 启用
    'inactive',    -- 禁用
    'failed'       -- 失败次数过多，自动禁用
);

-- Webhooks 主表
CREATE TABLE webhooks (
    id              SERIAL PRIMARY KEY,
    
    -- 基本信息
    name            VARCHAR(100) NOT NULL,           -- Webhook 名称
    description     TEXT,                            -- 描述
    
    -- 配置信息
    url             VARCHAR(500) NOT NULL,           -- 推送 URL
    secret          VARCHAR(255),                    -- 签名密钥
    
    -- 事件订阅
    events          webhook_event_type[] NOT NULL DEFAULT '{}',  -- 订阅的事件类型
    
    -- 状态
    status          webhook_status NOT NULL DEFAULT 'active',
    
    -- 重试配置
    retry_count     INTEGER NOT NULL DEFAULT 3,      -- 最大重试次数
    retry_interval  INTEGER NOT NULL DEFAULT 5,      -- 重试间隔（秒）
    timeout         INTEGER NOT NULL DEFAULT 30,     -- 请求超时（秒）
    
    -- 统计信息
    success_count   INTEGER NOT NULL DEFAULT 0,      -- 成功次数
    fail_count      INTEGER NOT NULL DEFAULT 0,      -- 失败次数
    last_triggered_at TIMESTAMPTZ,                   -- 最后触发时间
    last_error      TEXT,                            -- 最后错误信息
    
    -- 创建者
    created_by      INTEGER REFERENCES users(id),
    
    -- 时间戳
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- 软删除
    deleted_at      TIMESTAMPTZ
);

-- 创建索引
CREATE INDEX idx_webhooks_status ON webhooks(status);
CREATE INDEX idx_webhooks_events ON webhooks USING GIN(events);
CREATE INDEX idx_webhooks_created_by ON webhooks(created_by);
CREATE INDEX idx_webhooks_deleted_at ON webhooks(deleted_at) WHERE deleted_at IS NULL;

-- Webhook 投递日志表
CREATE TABLE webhook_deliveries (
    id              BIGSERIAL PRIMARY KEY,
    webhook_id      INTEGER NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    
    -- 请求信息
    event_type      webhook_event_type NOT NULL,     -- 事件类型
    payload         JSONB NOT NULL,                  -- 请求体
    
    -- 响应信息
    response_status INTEGER,                         -- HTTP 响应状态码
    response_body   TEXT,                            -- 响应内容
    
    -- 执行信息
    attempt_count   INTEGER NOT NULL DEFAULT 1,      -- 尝试次数
    duration_ms     INTEGER,                         -- 请求耗时（毫秒）
    error_message   TEXT,                            -- 错误信息
    
    -- 时间戳
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ                      -- 完成时间
);

-- 创建索引
CREATE INDEX idx_webhook_deliveries_webhook_id ON webhook_deliveries(webhook_id);
CREATE INDEX idx_webhook_deliveries_event_type ON webhook_deliveries(event_type);
CREATE INDEX idx_webhook_deliveries_created_at ON webhook_deliveries(created_at DESC);
CREATE INDEX idx_webhook_deliveries_status ON webhook_deliveries(response_status);

-- 投递日志分区（按时间分区，保留最近90天）
-- 注意：TimescaleDB 会自动处理，这里创建普通表即可

-- 更新时间戳触发器（如果已存在则跳过）
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_webhooks_updated_at') THEN
        CREATE TRIGGER update_webhooks_updated_at BEFORE UPDATE ON webhooks
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
END
$$;

-- 注释
COMMENT ON TABLE webhooks IS 'Webhook 配置表';
COMMENT ON TABLE webhook_deliveries IS 'Webhook 投递日志表';
COMMENT ON COLUMN webhooks.secret IS '用于 HMAC-SHA256 签名验证的密钥';
COMMENT ON COLUMN webhooks.events IS '订阅的事件类型数组，空数组表示订阅所有事件';
