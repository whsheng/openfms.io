import React from 'react';
import {
  Form,
  Input,
  Select,
  Button,
  Space,
  message,
} from 'antd';
import { userApi } from '../../services/api';
import type { User, Role } from '../../types/user';

const { Option } = Select;

interface UserFormProps {
  initialValues?: User | null;
  roles: Role[];
  onSuccess: () => void;
  onCancel: () => void;
}

const UserForm: React.FC<UserFormProps> = ({
  initialValues,
  roles,
  onSuccess,
  onCancel,
}) => {
  const [form] = Form.useForm();
  const isEditing = !!initialValues;

  const handleSubmit = async (values: any) => {
    try {
      if (isEditing) {
        await userApi.update(initialValues!.id, values);
        message.success('用户更新成功');
      } else {
        await userApi.create(values);
        message.success('用户创建成功');
      }
      onSuccess();
    } catch (error: any) {
      message.error(error.response?.data?.error || '操作失败');
    }
  };

  return (
    <Form
      form={form}
      layout="vertical"
      onFinish={handleSubmit}
      initialValues={initialValues || { status: 'active' }}
    >
      <Form.Item
        name="username"
        label="用户名"
        rules={[
          { required: true, message: '请输入用户名' },
          { min: 3, message: '用户名至少3个字符' },
          { pattern: /^[a-zA-Z0-9_]+$/, message: '用户名只能包含字母、数字和下划线' },
        ]}
      >
        <Input disabled={isEditing} placeholder="请输入用户名" />
      </Form.Item>

      <Form.Item
        name="nickname"
        label="昵称"
      >
        <Input placeholder="请输入昵称" />
      </Form.Item>

      {!isEditing && (
        <Form.Item
          name="password"
          label="密码"
          rules={[
            { required: true, message: '请输入密码' },
            { min: 6, message: '密码至少6个字符' },
          ]}
        >
          <Input.Password placeholder="请输入密码" />
        </Form.Item>
      )}

      {isEditing && (
        <Form.Item
          name="password"
          label="新密码"
          help="不填写则保持原密码不变"
        >
          <Input.Password placeholder="如需修改密码请输入新密码" />
        </Form.Item>
      )}

      <Form.Item
        name="role_id"
        label="角色"
        rules={[{ required: true, message: '请选择角色' }]}
      >
        <Select placeholder="请选择角色">
          {roles.map((role) => (
            <Option key={role.id} value={role.id}>
              {role.name}
            </Option>
          ))}
        </Select>
      </Form.Item>

      <Form.Item className="form-footer">
        <Space>
          <Button onClick={onCancel}>取消</Button>
          <Button type="primary" htmlType="submit">
            {isEditing ? '保存' : '创建'}
          </Button>
        </Space>
      </Form.Item>
    </Form>
  );
};

export default UserForm;
