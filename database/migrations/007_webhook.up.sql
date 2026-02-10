-- OpenFMS Webhook Module Database Schema
-- Migration: 007_webhook

-- ============================================
-- Webhook Event Type Enum
-- ============================================
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'webhook_event_type') THEN
        CREATE TYPE webhook_event_type AS ENUM (
            'alarm.created',
            'alarm.resolved',
            'geofence.enter',
            'geofence.exit',
            'device.online',
            'device.offline',
            'device.position',
            'device.command_result',
            'vehicle.created',
            'vehicle.updated',
            'all'
        );
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'webhook_status') THEN
        CREATE TYPE webhook_status AS ENUM ('active', 'inactive', 'failed');
    END IF;
END $$;

-- ============================================
-- Webhooks Table
-- ============================================
CREATE TABLE IF NOT EXISTS webhooks (
    id              SERIAL PRIMARY KEY,
    name            VARCHAR(100) NOT NULL,
    description     TEXT,
    url             VARCHAR(500) NOT NULL,
    secret          VARCHAR(255),
    events          webhook_event_type[] NOT NULL DEFAULT '{}',
    status          webhook_status NOT NULL DEFAULT 'active',
    retry_count     INTEGER NOT NULL DEFAULT 3,
    retry_interval  INTEGER NOT NULL DEFAULT 5,
    timeout         INTEGER NOT NULL DEFAULT 30,
    success_count   INTEGER NOT NULL DEFAULT 0,
    fail_count      INTEGER NOT NULL DEFAULT 0,
    last_triggered_at TIMESTAMPTZ,
    last_error      TEXT,
    created_by      INTEGER REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_webhooks_status ON webhooks(status);
CREATE INDEX IF NOT EXISTS idx_webhooks_events ON webhooks USING GIN(events);
CREATE INDEX IF NOT EXISTS idx_webhooks_created_by ON webhooks(created_by);
CREATE INDEX IF NOT EXISTS idx_webhooks_deleted_at ON webhooks(deleted_at) WHERE deleted_at IS NULL;

-- ============================================
-- Webhook Deliveries Table
-- ============================================
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id              BIGSERIAL PRIMARY KEY,
    webhook_id      INTEGER NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    event_type      webhook_event_type NOT NULL,
    payload         JSONB NOT NULL,
    response_status INTEGER,
    response_body   TEXT,
    attempt_count   INTEGER NOT NULL DEFAULT 1,
    duration_ms     INTEGER,
    error_message   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_webhook_id ON webhook_deliveries(webhook_id);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_event_type ON webhook_deliveries(event_type);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_created_at ON webhook_deliveries(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_status ON webhook_deliveries(response_status);

-- ============================================
-- Update trigger for webhooks
-- ============================================
DROP TRIGGER IF EXISTS update_webhooks_updated_at ON webhooks;
CREATE TRIGGER update_webhooks_updated_at BEFORE UPDATE ON webhooks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Comments
COMMENT ON TABLE webhooks IS 'Webhook 配置表';
COMMENT ON TABLE webhook_deliveries IS 'Webhook 投递日志表';
COMMENT ON COLUMN webhooks.secret IS '用于 HMAC-SHA256 签名验证的密钥';
COMMENT ON COLUMN webhooks.events IS '订阅的事件类型数组，空数组表示订阅所有事件';
