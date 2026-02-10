import React, { useEffect, useState, useRef, useCallback } from 'react'
import {
  Card,
  message,
  notification,
  Badge,
  Button,
  Tabs,
  Table,
  Switch,
  Modal,
  Form,
  Input,
  InputNumber,
  Space,
} from 'antd'
import {
  AlertOutlined,
  BellOutlined,
  SettingOutlined,
  EditOutlined,
  SoundOutlined,
  SoundFilled,
} from '@ant-design/icons'
import AlarmStats from './AlarmStats'
import AlarmList from './AlarmList'
import { alarmApi, AlarmWebSocket } from '../../services/alarm'
import type { TabsProps } from 'antd'

interface AlarmStatsData {
  total_today: number
  unread_count: number
  by_type: Record<string, number>
  by_level: Record<string, number>
}

interface AlarmRule {
  id: number
  type: string
  name: string
  description: string
  condition: Record<string, any>
  enabled: boolean
  notify_ws: boolean
  notify_sound: boolean
}

// Simple notification sound using Web Audio API
const playNotificationSound = () => {
  try {
    const audioContext = new (window.AudioContext || (window as any).webkitAudioContext)()
    const oscillator = audioContext.createOscillator()
    const gainNode = audioContext.createGain()

    oscillator.connect(gainNode)
    gainNode.connect(audioContext.destination)

    oscillator.frequency.value = 800
    oscillator.type = 'sine'

    gainNode.gain.setValueAtTime(0.3, audioContext.currentTime)
    gainNode.gain.exponentialRampToValueAtTime(0.01, audioContext.currentTime + 0.5)

    oscillator.start(audioContext.currentTime)
    oscillator.stop(audioContext.currentTime + 0.5)
  } catch (err) {
    console.log('Failed to play notification sound:', err)
  }
}

const AlarmCenter: React.FC = () => {
  const [stats, setStats] = useState<AlarmStatsData>({
    total_today: 0,
    unread_count: 0,
    by_type: {},
    by_level: {},
  })
  const [soundEnabled, setSoundEnabled] = useState(true)
  const [activeTab, setActiveTab] = useState('list')
  const wsRef = useRef<AlarmWebSocket | null>(null)

  // Load sound preference from localStorage
  useEffect(() => {
    const savedSound = localStorage.getItem('alarmSoundEnabled')
    if (savedSound !== null) {
      setSoundEnabled(savedSound === 'true')
    }
  }, [])

  // Play alarm sound
  const playAlarmSound = useCallback(() => {
    if (soundEnabled) {
      playNotificationSound()
    }
  }, [soundEnabled])

  // Toggle sound
  const toggleSound = useCallback(() => {
    const newValue = !soundEnabled
    setSoundEnabled(newValue)
    localStorage.setItem('alarmSoundEnabled', String(newValue))
    message.success(newValue ? '报警声音已开启' : '报警声音已关闭')
  }, [soundEnabled])

  // Fetch alarm stats
  const fetchStats = useCallback(async () => {
    try {
      const res = await alarmApi.getAlarmStats()
      setStats(res.data)
    } catch (error) {
      console.error('Failed to fetch alarm stats:', error)
    }
  }, [])

  // Handle new alarm from WebSocket
  const handleNewAlarm = useCallback(
    (alarm: any) => {
      // Play sound
      playAlarmSound()

      // Show notification
      notification.open({
        message: '新报警',
        description: (
          <div>
            <div>
              <strong>{alarm.device_name || alarm.device_id}</strong>
            </div>
            <div>{alarm.content}</div>
          </div>
        ),
        icon: <AlertOutlined style={{ color: '#ff4d4f' }} />,
        placement: 'topRight',
        duration: 5,
      })

      // Refresh stats
      fetchStats()
    },
    [playAlarmSound, fetchStats]
  )

  // Initialize WebSocket
  useEffect(() => {
    wsRef.current = new AlarmWebSocket(
      handleNewAlarm,
      () => console.log('[AlarmCenter] WebSocket connected'),
      () => console.log('[AlarmCenter] WebSocket disconnected')
    )
    wsRef.current.connect()

    return () => {
      wsRef.current?.disconnect()
    }
  }, [handleNewAlarm])

  // Initial data fetch
  useEffect(() => {
    fetchStats()
    const interval = setInterval(fetchStats, 30000) // Refresh every 30s
    return () => clearInterval(interval)
  }, [fetchStats])

  const items: TabsProps['items'] = [
    {
      key: 'list',
      label: (
        <span>
          <BellOutlined />
          报警列表
          {stats.unread_count > 0 && (
            <Badge count={stats.unread_count} style={{ marginLeft: 8 }} />
          )}
        </span>
      ),
      children: (
        <AlarmList
          onRefreshStats={fetchStats}
          soundEnabled={soundEnabled}
          onToggleSound={toggleSound}
        />
      ),
    },
    {
      key: 'rules',
      label: (
        <span>
          <SettingOutlined />
          报警规则
        </span>
      ),
      children: <AlarmRules />,
    },
  ]

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h2 style={{ margin: 0 }}>
          <AlertOutlined style={{ marginRight: 8 }} />
          报警中心
        </h2>
        <Space>
          <Button
            icon={soundEnabled ? <SoundFilled /> : <SoundOutlined />}
            onClick={toggleSound}
            type={soundEnabled ? 'primary' : 'default'}
          >
            {soundEnabled ? '声音开启' : '声音关闭'}
          </Button>
          <Button type="primary" onClick={fetchStats}>
            刷新统计
          </Button>
        </Space>
      </div>

      <AlarmStats stats={stats} />

      <Card>
        <Tabs
          activeKey={activeTab}
          items={items}
          onChange={setActiveTab}
        />
      </Card>
    </div>
  )
}

// Alarm Rules Component
const AlarmRules: React.FC = () => {
  const [rules, setRules] = useState<AlarmRule[]>([])
  const [loading, setLoading] = useState(false)
  const [editModalVisible, setEditModalVisible] = useState(false)
  const [editingRule, setEditingRule] = useState<AlarmRule | null>(null)
  const [form] = Form.useForm()

  const fetchRules = async () => {
    setLoading(true)
    try {
      const res = await alarmApi.getRules()
      setRules(res.data.data || [])
    } catch (error) {
      message.error('获取报警规则失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchRules()
  }, [])

  const handleToggleEnabled = async (rule: AlarmRule, checked: boolean) => {
    try {
      await alarmApi.updateRule(rule.id, { enabled: checked })
      message.success(checked ? '规则已启用' : '规则已禁用')
      fetchRules()
    } catch (error) {
      message.error('更新失败')
    }
  }

  const handleEdit = (rule: AlarmRule) => {
    setEditingRule(rule)
    form.setFieldsValue({
      name: rule.name,
      description: rule.description,
      notify_ws: rule.notify_ws,
      notify_sound: rule.notify_sound,
      ...rule.condition,
    })
    setEditModalVisible(true)
  }

  const handleSave = async (values: any) => {
    if (!editingRule) return
    try {
      const { name, description, notify_ws, notify_sound, ...condition } = values
      await alarmApi.updateRule(editingRule.id, {
        name,
        description,
        notify_ws,
        notify_sound,
        condition,
      })
      message.success('更新成功')
      setEditModalVisible(false)
      fetchRules()
    } catch (error) {
      message.error('更新失败')
    }
  }

  const getConditionFields = (type: string) => {
    switch (type) {
      case 'OVERSPEED':
        return [
          { name: 'speed_limit', label: '速度限制 (km/h)', min: 1, max: 200 },
          { name: 'duration', label: '持续时间 (秒)', min: 1, max: 300 },
        ]
      case 'LOW_BATTERY':
        return [{ name: 'battery_threshold', label: '电量阈值 (%)', min: 1, max: 100 }]
      case 'OFFLINE':
        return [{ name: 'offline_threshold', label: '离线阈值 (秒)', min: 60, max: 3600 }]
      default:
        return []
    }
  }

  const columns = [
    {
      title: '规则名称',
      dataIndex: 'name',
    },
    {
      title: '类型',
      dataIndex: 'type',
    },
    {
      title: '描述',
      dataIndex: 'description',
      ellipsis: true,
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      width: 100,
      render: (enabled: boolean, record: AlarmRule) => (
        <Switch
          checked={enabled}
          onChange={(checked) => handleToggleEnabled(record, checked)}
        />
      ),
    },
    {
      title: 'WebSocket推送',
      dataIndex: 'notify_ws',
      width: 120,
      render: (notify: boolean) => (notify ? '是' : '否'),
    },
    {
      title: '声音提醒',
      dataIndex: 'notify_sound',
      width: 100,
      render: (notify: boolean) => (notify ? <SoundFilled /> : <SoundOutlined />),
    },
    {
      title: '操作',
      key: 'action',
      width: 100,
      render: (_: any, record: AlarmRule) => (
        <Button type="text" icon={<EditOutlined />} onClick={() => handleEdit(record)}>
          编辑
        </Button>
      ),
    },
  ]

  const conditionFields = editingRule ? getConditionFields(editingRule.type) : []

  return (
    <div>
      <div style={{ marginBottom: 16 }}>
        <Button type="primary" onClick={fetchRules}>
          刷新
        </Button>
      </div>
      <Table
        columns={columns}
        dataSource={rules}
        rowKey="id"
        loading={loading}
        pagination={false}
      />

      <Modal
        title="编辑报警规则"
        open={editModalVisible}
        onCancel={() => setEditModalVisible(false)}
        onOk={() => form.submit()}
        okText="保存"
        cancelText="取消"
        width={600}
      >
        <Form form={form} layout="vertical" onFinish={handleSave}>
          <Form.Item
            name="name"
            label="规则名称"
            rules={[{ required: true, message: '请输入规则名称' }]}
          >
            <Input />
          </Form.Item>
          <Form.Item name="description" label="规则描述">
            <Input.TextArea rows={3} />
          </Form.Item>

          {conditionFields.map((field) => (
            <Form.Item
              key={field.name}
              name={field.name}
              label={field.label}
              rules={[{ required: true, message: `请输入${field.label}` }]}
            >
              <InputNumber min={field.min} max={field.max} style={{ width: '100%' }} />
            </Form.Item>
          ))}

          <Form.Item name="notify_ws" valuePropName="checked" label={null}>
            <Switch checkedChildren="WebSocket推送" unCheckedChildren="WebSocket推送" />
          </Form.Item>
          <Form.Item name="notify_sound" valuePropName="checked" label={null}>
            <Switch checkedChildren="声音提醒" unCheckedChildren="声音提醒" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default AlarmCenter
