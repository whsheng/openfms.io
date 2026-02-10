-- OpenFMS RBAC Module Database Schema
-- Migration: 004_rbac

-- ============================================
-- Roles Table
-- ============================================
CREATE TABLE IF NOT EXISTS roles (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(50) NOT NULL UNIQUE,
    code        VARCHAR(50) NOT NULL UNIQUE,
    description VARCHAR(200),
    is_system   BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================
-- Permissions Table
-- ============================================
CREATE TABLE IF NOT EXISTS permissions (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    code        VARCHAR(100) NOT NULL UNIQUE,
    description VARCHAR(200),
    group_name  VARCHAR(50) NOT NULL DEFAULT 'other',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================
-- Role Permissions Table
-- ============================================
CREATE TABLE IF NOT EXISTS role_permissions (
    id            SERIAL PRIMARY KEY,
    role_id       INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id INTEGER NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(role_id, permission_id)
);

-- ============================================
-- User Roles Table
-- ============================================
CREATE TABLE IF NOT EXISTS user_roles (
    id         SERIAL PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id    INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, role_id)
);

-- ============================================
-- Insert default roles
-- ============================================
INSERT INTO roles (name, code, description, is_system) VALUES
    ('超级管理员', 'super_admin', '系统超级管理员，拥有所有权限', true),
    ('管理员', 'admin', '系统管理员，可管理设备和用户', true),
    ('操作员', 'operator', '日常操作人员，可查看和基础操作', true),
    ('观察员', 'viewer', '只读用户，仅可查看数据', true)
ON CONFLICT (code) DO NOTHING;

-- ============================================
-- Insert permissions
-- ============================================
INSERT INTO permissions (name, code, description, group_name) VALUES
    ('设备查看', 'device:read', '查看设备列表和详情', 'device'),
    ('设备创建', 'device:create', '创建设备', 'device'),
    ('设备编辑', 'device:update', '编辑设备信息', 'device'),
    ('设备删除', 'device:delete', '删除设备', 'device'),
    ('设备指令', 'device:command', '发送设备指令', 'device'),
    ('位置查看', 'position:read', '查看实时位置和历史轨迹', 'position'),
    ('围栏查看', 'geofence:read', '查看围栏列表', 'geofence'),
    ('围栏创建', 'geofence:create', '创建围栏', 'geofence'),
    ('围栏编辑', 'geofence:update', '编辑围栏', 'geofence'),
    ('围栏删除', 'geofence:delete', '删除围栏', 'geofence'),
    ('报警查看', 'alarm:read', '查看报警列表', 'alarm'),
    ('报警处理', 'alarm:resolve', '处理报警', 'alarm'),
    ('报警规则', 'alarm:rule', '管理报警规则', 'alarm'),
    ('用户查看', 'user:read', '查看用户列表', 'user'),
    ('用户创建', 'user:create', '创建用户', 'user'),
    ('用户编辑', 'user:update', '编辑用户信息', 'user'),
    ('用户删除', 'user:delete', '删除用户', 'user'),
    ('角色管理', 'role:manage', '管理角色和权限', 'user'),
    ('设置查看', 'setting:read', '查看系统设置', 'setting'),
    ('设置编辑', 'setting:update', '修改系统设置', 'setting')
ON CONFLICT (code) DO NOTHING;

-- ============================================
-- Assign permissions to roles
-- ============================================
-- Super admin has all permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT 1, id FROM permissions
ON CONFLICT DO NOTHING;

-- Admin permissions (all except role:manage and user:delete)
INSERT INTO role_permissions (role_id, permission_id)
SELECT 2, id FROM permissions WHERE code NOT IN ('role:manage', 'user:delete')
ON CONFLICT DO NOTHING;

-- Operator permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT 3, id FROM permissions 
WHERE code IN (
    'device:read', 'device:command',
    'position:read',
    'geofence:read', 'geofence:create', 'geofence:update',
    'alarm:read', 'alarm:resolve',
    'user:read'
)
ON CONFLICT DO NOTHING;

-- Viewer permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT 4, id FROM permissions 
WHERE code IN (
    'device:read',
    'position:read',
    'geofence:read',
    'alarm:read'
)
ON CONFLICT DO NOTHING;

-- ============================================
-- Add role_id to users table
-- ============================================
ALTER TABLE users ADD COLUMN IF NOT EXISTS role_id INTEGER REFERENCES roles(id);

-- Update existing users to admin role
UPDATE users SET role_id = 2 WHERE role_id IS NULL;

-- ============================================
-- Create indexes
-- ============================================
CREATE INDEX IF NOT EXISTS idx_role_permissions_role ON role_permissions(role_id);
CREATE INDEX IF NOT EXISTS idx_role_permissions_permission ON role_permissions(permission_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_user ON user_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role ON user_roles(role_id);
CREATE INDEX IF NOT EXISTS idx_permissions_group ON permissions(group_name);

-- ============================================
-- Update triggers
-- ============================================
DROP TRIGGER IF EXISTS update_roles_updated_at ON roles;
CREATE TRIGGER update_roles_updated_at BEFORE UPDATE ON roles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Comments
COMMENT ON TABLE roles IS '角色表';
COMMENT ON TABLE permissions IS '权限表';
COMMENT ON TABLE role_permissions IS '角色权限关联表';
COMMENT ON TABLE user_roles IS '用户角色关联表';
