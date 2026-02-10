import React, { useState, useEffect } from 'react';
import {
  Table,
  Button,
  Space,
  Tag,
  Modal,
  message,
  Popconfirm,
  Card,
  Tree,
  Checkbox,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SafetyOutlined,
  LockOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { userApi } from '../../services/api';
import RoleForm from './RoleForm';
import type { Role, Permission, PermissionGroup } from '../../types/user';
import styles from './user.module.css';

const RoleList: React.FC = () => {
  const [roles, setRoles] = useState<Role[]>([]);
  const [permissionGroups, setPermissionGroups] = useState<PermissionGroup[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [permModalVisible, setPermModalVisible] = useState(false);
  const [editingRole, setEditingRole] = useState<Role | null>(null);
  const [selectedPermissions, setSelectedPermissions] = useState<number[]>([]);

  // 获取角色列表
  const fetchRoles = async () => {
    setLoading(true);
    try {
      const res = await userApi.getRoles();
      setRoles(res.data);
    } catch (error) {
      message.error('获取角色列表失败');
    } finally {
      setLoading(false);
    }
  };

  // 获取权限分组
  const fetchPermissions = async () => {
    try {
      const res = await userApi.getPermissionGroups();
      setPermissionGroups(res.data);
    } catch (error) {
      message.error('获取权限列表失败');
    }
  };

  useEffect(() => {
    fetchRoles();
    fetchPermissions();
  }, []);

  // 创建角色
  const handleCreate = () => {
    setEditingRole(null);
    setModalVisible(true);
  };

  // 编辑角色
  const handleEdit = (role: Role) => {
    setEditingRole(role);
    setModalVisible(true);
  };

  // 删除角色
  const handleDelete = async (id: number) => {
    try {
      await userApi.deleteRole(id);
      message.success('删除成功');
      fetchRoles();
    } catch (error) {
      message.error('删除失败');
    }
  };

  // 配置权限
  const handleConfigPermission = async (role: Role) => {
    setEditingRole(role);
    try {
      const res = await userApi.getRolePermissions(role.id);
      const permIds = res.data.map((p: Permission) => p.id);
      setSelectedPermissions(permIds);
      setPermModalVisible(true);
    } catch (error) {
      message.error('获取角色权限失败');
    }
  };

  // 保存权限配置
  const handleSavePermissions = async () => {
    if (!editingRole) return;
    try {
      await userApi.updateRole(editingRole.id, {
        permission_ids: selectedPermissions,
      });
      message.success('权限配置已保存');
      setPermModalVisible(false);
      fetchRoles();
    } catch (error) {
      message.error('保存失败');
    }
  };

  // 表单提交成功
  const handleFormSuccess = () => {
    setModalVisible(false);
    fetchRoles();
  };

  // 获取权限树数据
  const getTreeData = () => {
    return permissionGroups.map((group) => ({
      title: group.label,
      key: `group-${group.name}`,
      children: group.permissions.map((perm) => ({
        title: perm.name,
        key: perm.id,
      })),
    }));
  };

  const columns: ColumnsType<Role> = [
    {
      title: '角色名称',
      dataIndex: 'name',
      render: (name, record) => (
        <Space>
          <SafetyOutlined />
          <span>{name}</span>
          {record.is_system && <Tag color="blue">系统</Tag>}
        </Space>
      ),
    },
    {
      title: '角色编码',
      dataIndex: 'code',
    },
    {
      title: '描述',
      dataIndex: 'description',
      ellipsis: true,
    },
    {
      title: '操作',
      key: 'action',
      width: 250,
      render: (_, record) => (
        <Space size="small">
          <Button
            type="text"
            icon={<LockOutlined />}
            onClick={() => handleConfigPermission(record)}
          >
            权限
          </Button>
          {!record.is_system && (
            <>
              <Button
                type="text"
                icon={<EditOutlined />}
                onClick={() => handleEdit(record)}
              >
                编辑
              </Button>
              <Popconfirm
                title="确定删除此角色吗？"
                onConfirm={() => handleDelete(record.id)}
                okText="确定"
                cancelText="取消"
              >
                <Button type="text" danger icon={<DeleteOutlined />}>
                  删除
                </Button>
              </Popconfirm>
            </>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div className={styles.roleList}>
      <Card
        title={
          <Space>
            <LockOutlined />
            角色管理
          </Space>
        }
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
            新建角色
          </Button>
        }
      >
        <Table
          columns={columns}
          dataSource={roles}
          rowKey="id"
          loading={loading}
          pagination={false}
        />
      </Card>

      <Modal
        title={editingRole ? '编辑角色' : '新建角色'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        destroyOnClose
        width={500}
      >
        <RoleForm
          initialValues={editingRole}
          onSuccess={handleFormSuccess}
          onCancel={() => setModalVisible(false)}
        />
      </Modal>

      <Modal
        title={`配置权限 - ${editingRole?.name}`}
        open={permModalVisible}
        onCancel={() => setPermModalVisible(false)}
        onOk={handleSavePermissions}
        width={600}
      >
        <Tree
          checkable
          treeData={getTreeData()}
          checkedKeys={selectedPermissions}
          onCheck={(checked) => setSelectedPermissions(checked as number[])}
        />
      </Modal>
    </div>
  );
};

export default RoleList;
