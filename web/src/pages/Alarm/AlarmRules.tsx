import React, { useState, useEffect } from 'react';
import {
  Card,
  Table,
  Switch,
  Button,
  Modal,
  Form,
  Input,
  InputNumber,
  Select,
  Space,
  message,
  Tag,
} from 'antd';
import { EditOutlined, BellOutlined, BellMutedOutlined } from '@ant-design/icons';
import { alarmApi } from '../../services/api';
import type { AlarmRule, AlarmType } from '../../types/alarm';
import styles from './alarm.module.css';

const { Option } = Select;

const AlarmRules: React.FC = () => {
  const [rules, setRules] = useState<AlarmRule[]>([]);
  const [loading, setLoading] = useState(false);
  const [editingRule, setEditingRule] = useState<AlarmRule | null>(null);
  const [form] = Form.useForm();

  // 获取规则列表
  const fetchRules = async () => {
    setLoading(true);
    try {
      const res = await alarmApi.getRules();
      setRules(res.data);
    } catch (error) {
      message.error('获取规则列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchRules();
  }, []);

  // 切换规则启用状态
  const handleToggle = async (id: number, enabled: boolean) => {
    try {
      await alarmApi.toggleRule(id);
      message.success(enabled ? '规则已禁用' : '规则已启用');
      fetchRules();
    } catch (error) {
      message.error('操作失败');
    }
  };

  // 编辑规则
  const handleEdit = (rule: AlarmRule) => {
    setEditingRule(rule);
    form.setFieldsValue({
      name: rule.name,
      description: rule.description,
      conditions: rule.conditions,
      notify_webhook: rule.notify_webhook,
      webhook_url: rule.webhook_url,
      notify_ws: rule.notify_ws,
      notify_sound: rule.notify_sound,
    });
  };

  // 保存规则
  const handleSave = async (values: any) => {
    if (!editingRule) return;
    
    try {
      await alarmApi.updateRule(editingRule.id, values);
      message.success('规则更新成功');
      setEditingRule(null);
      fetchRules();
    } catch (error) {
      message.error('更新失败');
    }
  };

  // 获取报警类型标签
  const getTypeLabel = (type: AlarmType) => {
    const labels: Record<string, string> = {
      GEOFENCE_ENTER: '进入围栏',
      GEOFENCE_EXIT: '离开围栏',
      OVERSPEED: '超速',
      LOW_BATTERY: '低电量',
      OFFLINE: '设备离线',
      SOS: '紧急求救',
      POWER_CUT: '断电报警',
      VIBRATION: '震动报警',
      ILLEGAL_MOVE: '非法移动',
    };
    return labels[type] || type;
  };

  const columns = [
    {
      title: '规则名称',
      dataIndex: 'name',
      key: 'name',
    },
      {
      title: '报警类型',
      dataIndex: 'type',
      key: 'type',
      render: (type: AlarmType) => <Tag color="blue">{getTypeLabel(type)}</Tag>,
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: '条件',
      key: 'conditions',
      render: (_: any, record: AlarmRule) => {
        if (record.type === 'OVERSPEED') {
          return `限速 ${record.conditions.speed_limit || 120} km/h`;
        }
        if (record.type === 'OFFLINE') {
          return `离线 ${record.conditions.offline_minutes || 10} 分钟`;
        }
        return '-';
      },
    },
    {
      title: '通知方式',
      key: 'notifications',
      render: (_: any, record: AlarmRule) => (
        <Space>
          {record.notify_ws && <Tag color="green">WebSocket</Tag>}
          {record.notify_webhook && <Tag color="blue">Webhook</Tag>}
          {record.notify_sound && <Tag color="orange">声音</Tag>}
        </Space>
      ),
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      key: 'enabled',
      render: (enabled: boolean, record: AlarmRule) => (
        <Switch
          checked={enabled}
          onChange={() => handleToggle(record.id, enabled)}
          checkedChildren="启用"
          unCheckedChildren="禁用"
        />
      ),
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: AlarmRule) => (
        <Button
          type="text"
          icon={<EditOutlined />}
          onClick={() => handleEdit(record)}
        >
          编辑
        </Button>
      ),
    },
  ];

  return (
    <div className={styles.alarmRules}>
      <Card>
        <Table
          columns={columns}
          dataSource={rules}
          rowKey="id"
          loading={loading}
          pagination={false}
        />
      </Card>

      <Modal
        title="编辑报警规则"
        open={!!editingRule}
        onCancel={() => setEditingRule(null)}
        onOk={() => form.submit()}
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSave}
        >
          <Form.Item
            name="name"
            label="规则名称"
            rules={[{ required: true, message: '请输入规则名称' }]}
          >
            <Input />
          </Form.Item>

          <Form.Item
            name="description"
            label="描述"
          >
            <Input.TextArea rows={2} />
          </Form.Item>

          {editingRule?.type === 'OVERSPEED' && (
            <Form.Item
              name={['conditions', 'speed_limit']}
              label="限速阈值 (km/h)"
              rules={[{ required: true, message: '请输入限速阈值' }]}
            >
              <InputNumber min={1} max={300} style={{ width: '100%' }} />
            </Form.Item>
          )}

          {editingRule?.type === 'OFFLINE' && (
            <Form.Item
              name={['conditions', 'offline_minutes']}
              label="离线时间阈值 (分钟)"
              rules={[{ required: true, message: '请输入离线时间阈值' }]}
            >
              <InputNumber min={1} max={1440} style={{ width: '100%' }} />
            </Form.Item>
          )}

          <Form.Item
            name="notify_ws"
            valuePropName="checked"
          >
            <Switch checkedChildren="WebSocket推送开启" unCheckedChildren="WebSocket推送关闭" />
          </Form.Item>

          <Form.Item
            name="notify_sound"
            valuePropName="checked"
          >
            <Switch checkedChildren="声音提醒开启" unCheckedChildren="声音提醒关闭" />
          </Form.Item>

          <Form.Item
            name="notify_webhook"
            valuePropName="checked"
          >
            <Switch checkedChildren="Webhook开启" unCheckedChildren="Webhook关闭" />
          </Form.Item>

          <Form.Item
            noStyle
            shouldUpdate={(prev, curr) => prev.notify_webhook !== curr.notify_webhook}
          >
            {({ getFieldValue }) =>
              getFieldValue('notify_webhook') ? (
                <Form.Item
                  name="webhook_url"
                  label="Webhook URL"
                  rules={[{ required: true, message: '请输入Webhook URL' }]}
                >
                  <Input placeholder="https://example.com/webhook" />
                </Form.Item>
              ) : null
            }
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default AlarmRules;
