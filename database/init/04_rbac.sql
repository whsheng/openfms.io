-- RBAC 权限模块数据库表结构
-- 创建于: 2026-02-04

-- 角色表
CREATE TABLE roles (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(50) NOT NULL UNIQUE,
    code        VARCHAR(50) NOT NULL UNIQUE,  -- 角色编码，如 admin, operator, viewer
    description VARCHAR(200),
    
    -- 内置角色不可删除
    is_system   BOOLEAN NOT NULL DEFAULT false,
    
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 权限表
CREATE TABLE permissions (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    code        VARCHAR(100) NOT NULL UNIQUE,  -- 权限编码，如 device:read, device:write
    description VARCHAR(200),
    
    -- 权限分组，用于前端展示
    group_name  VARCHAR(50) NOT NULL DEFAULT 'other',
    
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 角色-权限关联表
CREATE TABLE role_permissions (
    id            SERIAL PRIMARY KEY,
    role_id       INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id INTEGER NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(role_id, permission_id)
);

-- 用户-角色关联表
CREATE TABLE user_roles (
    id         SERIAL PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id    INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(user_id, role_id)
);

-- 插入默认角色
INSERT INTO roles (name, code, description, is_system) VALUES
    ('超级管理员', 'super_admin', '系统超级管理员，拥有所有权限', true),
    ('管理员', 'admin', '系统管理员，可管理设备和用户', true),
    ('操作员', 'operator', '日常操作人员，可查看和基础操作', true),
    ('观察员', 'viewer', '只读用户，仅可查看数据', true);

-- 插入权限列表
INSERT INTO permissions (name, code, description, group_name) VALUES
    -- 设备管理
    ('设备查看', 'device:read', '查看设备列表和详情', 'device'),
    ('设备创建', 'device:create', '创建设备', 'device'),
    ('设备编辑', 'device:update', '编辑设备信息', 'device'),
    ('设备删除', 'device:delete', '删除设备', 'device'),
    ('设备指令', 'device:command', '发送设备指令', 'device'),
    
    -- 位置轨迹
    ('位置查看', 'position:read', '查看实时位置和历史轨迹', 'position'),
    
    -- 电子围栏
    ('围栏查看', 'geofence:read', '查看围栏列表', 'geofence'),
    ('围栏创建', 'geofence:create', '创建围栏', 'geofence'),
    ('围栏编辑', 'geofence:update', '编辑围栏', 'geofence'),
    ('围栏删除', 'geofence:delete', '删除围栏', 'geofence'),
    
    -- 报警中心
    ('报警查看', 'alarm:read', '查看报警列表', 'alarm'),
    ('报警处理', 'alarm:resolve', '处理报警', 'alarm'),
    ('报警规则', 'alarm:rule', '管理报警规则', 'alarm'),
    
    -- 用户管理
    ('用户查看', 'user:read', '查看用户列表', 'user'),
    ('用户创建', 'user:create', '创建用户', 'user'),
    ('用户编辑', 'user:update', '编辑用户信息', 'user'),
    ('用户删除', 'user:delete', '删除用户', 'user'),
    ('角色管理', 'role:manage', '管理角色和权限', 'user'),
    
    -- 系统设置
    ('设置查看', 'setting:read', '查看系统设置', 'setting'),
    ('设置编辑', 'setting:update', '修改系统设置', 'setting');

-- 超级管理员拥有所有权限
INSERT INTO role_permissions (role_id, permission_id)
SELECT 1, id FROM permissions;

-- 管理员权限
INSERT INTO role_permissions (role_id, permission_id)
SELECT 2, id FROM permissions WHERE code NOT IN ('role:manage', 'user:delete');

-- 操作员权限
INSERT INTO role_permissions (role_id, permission_id)
SELECT 3, id FROM permissions 
WHERE code IN (
    'device:read', 'device:command',
    'position:read',
    'geofence:read', 'geofence:create', 'geofence:update',
    'alarm:read', 'alarm:resolve',
    'user:read'
);

-- 观察员权限
INSERT INTO role_permissions (role_id, permission_id)
SELECT 4, id FROM permissions 
WHERE code IN (
    'device:read',
    'position:read',
    'geofence:read',
    'alarm:read'
);

-- 给用户表添加角色字段（冗余存储，方便查询）
ALTER TABLE users ADD COLUMN IF NOT EXISTS role_id INTEGER REFERENCES roles(id);

-- 更新现有用户为管理员
UPDATE users SET role_id = 2 WHERE role_id IS NULL;

-- 创建索引
CREATE INDEX idx_role_permissions_role ON role_permissions(role_id);
CREATE INDEX idx_role_permissions_permission ON role_permissions(permission_id);
CREATE INDEX idx_user_roles_user ON user_roles(user_id);
CREATE INDEX idx_user_roles_role ON user_roles(role_id);
CREATE INDEX idx_permissions_group ON permissions(group_name);

-- 更新时间戳触发器
CREATE TRIGGER update_roles_updated_at BEFORE UPDATE ON roles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 注释
COMMENT ON TABLE roles IS '角色表';
COMMENT ON TABLE permissions IS '权限表';
COMMENT ON TABLE role_permissions IS '角色权限关联表';
COMMENT ON TABLE user_roles IS '用户角色关联表';
