-- 车辆管理模块数据库表结构
-- 创建于: 2026-02-04

-- 车辆信息表
CREATE TABLE vehicles (
    id              SERIAL PRIMARY KEY,
    
    -- 基本信息
    plate_number    VARCHAR(20) NOT NULL UNIQUE,     -- 车牌号
    plate_color     VARCHAR(10) DEFAULT '蓝色',       -- 车牌颜色
    vehicle_type    VARCHAR(50),                     -- 车辆类型: 小型车、货车等
    brand           VARCHAR(50),                     -- 品牌
    model           VARCHAR(50),                     -- 型号
    color           VARCHAR(20),                     -- 车身颜色
    
    -- 车辆识别信息
    vin             VARCHAR(17),                     -- 车架号
    engine_no       VARCHAR(30),                     -- 发动机号
    
    -- 运营信息
    owner_name      VARCHAR(100),                    -- 车主姓名
    owner_phone     VARCHAR(20),                     -- 车主电话
    owner_idcard    VARCHAR(18),                     -- 车主身份证号
    
    -- 证件信息
    registration_no VARCHAR(30),                     -- 行驶证号
    transport_no    VARCHAR(30),                     -- 运输证号
    insurance_no    VARCHAR(30),                     -- 保险单号
    insurance_expire DATE,                          -- 保险到期日
    
    -- 关联设备
    device_id       VARCHAR(20) REFERENCES devices(sim_no) ON DELETE SET NULL,
    
    -- 状态
    status          VARCHAR(20) DEFAULT 'active',   -- active, inactive, scrapped
    
    -- 备注
    remark          TEXT,
    
    -- 时间戳
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_vehicles_plate ON vehicles(plate_number);
CREATE INDEX idx_vehicles_device ON vehicles(device_id);
CREATE INDEX idx_vehicles_status ON vehicles(status);

-- 车辆分组表
CREATE TABLE vehicle_groups (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(50) NOT NULL,
    color       VARCHAR(20) DEFAULT '#1890ff',      -- 分组颜色
    description TEXT,
    created_by  INTEGER REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 车辆分组关联表
CREATE TABLE vehicle_group_members (
    id          SERIAL PRIMARY KEY,
    vehicle_id  INTEGER NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    group_id    INTEGER NOT NULL REFERENCES vehicle_groups(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(vehicle_id, group_id)
);

CREATE INDEX idx_vgm_vehicle ON vehicle_group_members(vehicle_id);
CREATE INDEX idx_vgm_group ON vehicle_group_members(group_id);

-- 设备车辆绑定历史记录
CREATE TABLE device_vehicle_history (
    id          SERIAL PRIMARY KEY,
    device_id   VARCHAR(20) NOT NULL,
    vehicle_id  INTEGER NOT NULL,
    action      VARCHAR(20) NOT NULL,   -- bind, unbind
    operated_by INTEGER REFERENCES users(id),
    operated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    remark      TEXT
);

CREATE INDEX idx_dvh_device ON device_vehicle_history(device_id);
CREATE INDEX idx_dvh_vehicle ON device_vehicle_history(vehicle_id);

-- 更新时间戳触发器
CREATE TRIGGER update_vehicles_updated_at BEFORE UPDATE ON vehicles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 注释
COMMENT ON TABLE vehicles IS '车辆信息表';
COMMENT ON TABLE vehicle_groups IS '车辆分组表';
COMMENT ON TABLE vehicle_group_members IS '车辆分组关联表';
COMMENT ON TABLE device_vehicle_history IS '设备车辆绑定历史';
