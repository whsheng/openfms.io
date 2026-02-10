// 用户
type UserStatus = 'active' | 'disabled';

export interface User {
  id: number;
  username: string;
  nickname?: string;
  status: UserStatus;
  role_id?: number;
  role_name?: string;
  role_code?: string;
  created_at: string;
  updated_at: string;
}

// 角色
export interface Role {
  id: number;
  name: string;
  code: string;
  description?: string;
  is_system: boolean;
  created_at: string;
  updated_at: string;
}

// 权限
export interface Permission {
  id: number;
  name: string;
  code: string;
  description?: string;
  group_name: string;
  created_at: string;
}

// 权限分组
export interface PermissionGroup {
  name: string;
  label: string;
  permissions: Permission[];
}

// 用户权限响应
export interface UserPermissionsResponse {
  user_id: number;
  role: {
    role_id: number;
    role_name: string;
    role_code: string;
  };
  permissions: string[];
}
