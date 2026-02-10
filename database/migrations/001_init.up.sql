-- 初始数据库迁移
-- 创建基础表结构

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id          SERIAL PRIMARY KEY,
    username    VARCHAR(50) NOT NULL UNIQUE,
    password    VARCHAR(255) NOT NULL,
    nickname    VARCHAR(100),
    email       VARCHAR(100),
    phone       VARCHAR(20),
    status      VARCHAR(20) DEFAULT 'active',
    role_id     INTEGER,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 设备表
CREATE TABLE IF NOT EXISTS devices (
    id              SERIAL PRIMARY KEY,
    sim_no          VARCHAR(20) NOT NULL UNIQUE,
    name            VARCHAR(100),
    protocol        VARCHAR(20) DEFAULT 'JT808',
    status          VARCHAR(20) DEFAULT 'offline',
    last_report_at  TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 位置表
CREATE TABLE IF NOT EXISTS positions (
    id          BIGSERIAL PRIMARY KEY,
    time        TIMESTAMPTZ NOT NULL,
    device_id   VARCHAR(20) NOT NULL,
    lat         DOUBLE PRECISION NOT NULL,
    lon         DOUBLE PRECISION NOT NULL,
    speed       SMALLINT,
    angle       SMALLINT,
    altitude    INTEGER,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_positions_device_time ON positions(device_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_positions_time ON positions(time DESC);
