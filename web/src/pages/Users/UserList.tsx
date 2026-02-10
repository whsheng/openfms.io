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
  Avatar,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  UserOutlined,
  SafetyOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { userApi } from '../../services/api';
import UserForm from './UserForm';
import type { User, Role } from '../../types/user';
import styles from './user.module.css';

const UserList: React.FC = () => {
  const [users, setUsers] = useState<User[]>([]);
  const [roles, setRoles] = useState<Role[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingUser, setEditingUser] = useState<User | null>(null);

  // 获取用户列表
  const fetchUsers = async () => {
    setLoading(true);
    try {
      const res = await userApi.getList();
      setUsers(res.data);
    } catch (error) {
      message.error('获取用户列表失败');
    } finally {
      setLoading(false);
    }
  };

  // 获取角色列表
  const fetchRoles = async () => {
    try {
      const res = await userApi.getRoles();
      setRoles(res.data);
    } catch (error) {
      message.error('获取角色列表失败');
    }
  };

  useEffect(() => {
    fetchUsers();
    fetchRoles();
  }, []);

  // 创建用户
  const handleCreate = () => {
    setEditingUser(null);
    setModalVisible(true);
  };

  // 编辑用户
  const handleEdit = (user: User) => {
    setEditingUser(user);
    setModalVisible(true);
  };

  // 删除用户
  const handleDelete = async (id: number) => {
    try {
      await userApi.delete(id);
      message.success('删除成功');
      fetchUsers();
    } catch (error) {
      message.error('删除失败');
    }
  };

  // 表单提交成功
  const handleFormSuccess = () => {
    setModalVisible(false);
    fetchUsers();
  };

  // 获取角色标签颜色
  const getRoleColor = (roleCode?: string) => {
    const colors: Record<string, string> = {
      super_admin: 'red',
      admin: 'orange',
      operator: 'blue',
      viewer: 'default',
    };
    return colors[roleCode || ''] || 'default';
  };

  const columns: ColumnsType<User> = [
    {
      title: '用户',
      key: 'user',
      render: (_, record) => (
        <Space>
          <Avatar icon={<UserOutlined />} />
          <div>
            <div className={styles.userName}>{record.nickname || record.username}</div>
            <div className={styles.userUsername}>{record.username}</div>
          </div>
        </Space>
      ),
    },
    {
      title: '角色',
      dataIndex: 'role_name',
      render: (name, record) => (
        <Tag color={getRoleColor(record.role_code)}>
          {name || '未分配'}
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      render: (status) => (
        <Tag color={status === 'active' ? 'green' : 'red'}>
          {status === 'active' ? '正常' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      render: (time) => new Date(time).toLocaleString(),
    },
    {
      title: '操作',
      key: 'action',
      width: 200,
      render: (_, record) => (
        <Space size="small">
          <Button
            type="text"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          >
            编辑
          </Button>
          <Popconfirm
            title="确定删除此用户吗？"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button type="text" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className={styles.userList}>
      <Card
        title={
          <Space>
            <SafetyOutlined />
            用户管理
          </Space>
        }
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
            新建用户
          </Button>
        }
      >
        <Table
          columns={columns}
          dataSource={users}
          rowKey="id"
          loading={loading}
          pagination={{
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条`,
          }}
        />
      </Card>

      <Modal
        title={editingUser ? '编辑用户' : '新建用户'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        destroyOnClose
        width={500}
      >
        <UserForm
          initialValues={editingUser}
          roles={roles}
          onSuccess={handleFormSuccess}
          onCancel={() => setModalVisible(false)}
        />
      </Modal>
    </div>
  );
};

export default UserList;
