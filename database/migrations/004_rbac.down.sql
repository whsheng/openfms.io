-- OpenFMS Database Migration Down: 004_rbac
-- Rollback all changes from 004_rbac

-- Drop triggers
DROP TRIGGER IF EXISTS update_roles_updated_at ON roles;

-- Remove role_id from users
ALTER TABLE users DROP COLUMN IF EXISTS role_id;

-- Drop indexes
DROP INDEX IF EXISTS idx_role_permissions_role;
DROP INDEX IF EXISTS idx_role_permissions_permission;
DROP INDEX IF EXISTS idx_user_roles_user;
DROP INDEX IF EXISTS idx_user_roles_role;
DROP INDEX IF EXISTS idx_permissions_group;

-- Drop tables
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
