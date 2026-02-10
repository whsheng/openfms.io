import React from 'react';
import {
  Form,
  Input,
  Button,
  Space,
  message,
} from 'antd';
import { userApi } from '../../services/api';
import type { Role } from '../../types/user';

interface RoleFormProps {
  initialValues?: Role | null;
  onSuccess: () => void;
  onCancel: () => void;
}

const RoleForm: React.FC<RoleFormProps> = ({
  initialValues,
  onSuccess,
  onCancel,
}) => {
  const [form] = Form.useForm();
  const isEditing = !!initialValues;

  const handleSubmit = async (values: any) => {
    try {
      if (isEditing) {
        await userApi.updateRole(initialValues!.id, values);
        message.success('角色更新成功');
      } else {
        await userApi.createRole(values);
        message.success('角色创建成功');
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
      initialValues={initialValues || {}}
    >
      <Form.Item
        name="name"
        label="角色名称"
        rules={[{ required: true, message: '请输入角色名称' }]}
      >
        <Input placeholder="请输入角色名称" />
      </Form.Item>

      {!isEditing && (
        <Form.Item
          name="code"
          label="角色编码"
          rules={[
            { required: true, message: '请输入角色编码' },
            { pattern: /^[a-z_]+$/, message: '角色编码只能使用小写字母和下划线' },
          ]}
          help="用于系统识别，创建后不可修改"
        >
          <Input placeholder="如：admin, operator" />
        </Form.Item>
      )}

      <Form.Item
        name="description"
        label="描述"
      >
        <Input.TextArea rows={3} placeholder="请输入角色描述" />
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

export default RoleForm;
