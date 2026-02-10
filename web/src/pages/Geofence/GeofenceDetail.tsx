import React, { useEffect, useState } from 'react'
import {
  Card,
  Descriptions,
  Tag,
  Table,
  Button,
  Space,
  Modal,
  message,
  Tabs,
  Row,
  Col,
  Select,
  Empty,
} from 'antd'
import {
  EditOutlined,
  LinkOutlined,
  UnlinkOutlined,
  HistoryOutlined,
  EnvironmentOutlined,
} from '@ant-design/icons'
import dayjs from 'dayjs'
import { geofenceApi, deviceApi } from '../../services/api'
import GeofenceMap from './GeofenceMap'

const { TabPane } = Tabs
const { Option } = Select

interface Geofence {
  id: number
  name: string
  description?: string
  type: 'circle' | 'polygon'
  coordinates: any
  alert_type: 'enter' | 'exit' | 'both'
  status: number
  created_at: string
  updated_at: string
}

interface Device {
  id: number
  device_id: string
  name: string
  protocol: string
  status: number
}

interface GeofenceEvent {
  id: number
  device_id: number
  device?: Device
  event_type: 'enter' | 'exit'
  location: { lat: number; lon: number }
  speed: number
  triggered_at: string
}

interface GeofenceDetailProps {
  geofence: Geofence
  onRefresh: () => void
}

const GeofenceDetail: React.FC<GeofenceDetailProps> = ({ geofence, onRefresh }) => {
  const [activeTab, setActiveTab] = useState('info')
  const [devices, setDevices] = useState<Device[]>([])
  const [events, setEvents] = useState<GeofenceEvent[]>([])
  const [allDevices, setAllDevices] = useState<Device[]>([])
  const [loading, setLoading] = useState(false)
  const [bindModalVisible, setBindModalVisible] = useState(false)
  const [selectedDeviceIds, setSelectedDeviceIds] = useState<number[]>([])
  const [eventPagination, setEventPagination] = useState({
    current: 1,
    pageSize: 10,
    total: 0,
  })

  useEffect(() => {
    if (activeTab === 'devices') {
      fetchDevices()
    } else if (activeTab === 'events') {
      fetchEvents()
    }
  }, [activeTab])

  useEffect(() => {
    if (bindModalVisible) {
      fetchAllDevices()
    }
  }, [bindModalVisible])

  const fetchDevices = async () => {
    setLoading(true)
    try {
      const res = await geofenceApi.getGeofenceDevices(geofence.id)
      setDevices(res.data.data || [])
    } catch (error) {
      message.error('获取绑定设备失败')
    } finally {
      setLoading(false)
    }
  }

  const fetchAllDevices = async () => {
    try {
      const res = await deviceApi.getDevices({ page: 1, page_size: 1000 })
      setAllDevices(res.data.data || [])
    } catch (error) {
      message.error('获取设备列表失败')
    }
  }

  const fetchEvents = async () => {
    setLoading(true)
    try {
      const res = await geofenceApi.getGeofenceEvents(geofence.id, {
        page: eventPagination.current,
        page_size: eventPagination.pageSize,
      })
      setEvents(res.data.data || [])
      setEventPagination((prev) => ({ ...prev, total: res.data.total || 0 }))
    } catch (error) {
      message.error('获取事件记录失败')
    } finally {
      setLoading(false)
    }
  }

  const handleBindDevices = async () => {
    if (selectedDeviceIds.length === 0) {
      message.warning('请选择要绑定的设备')
      return
    }
    try {
      await geofenceApi.bindDevices(geofence.id, selectedDeviceIds)
      message.success('设备绑定成功')
      setBindModalVisible(false)
      setSelectedDeviceIds([])
      fetchDevices()
      onRefresh()
    } catch (error) {
      message.error('设备绑定失败')
    }
  }

  const handleUnbindDevice = async (deviceId: number) => {
    try {
      await geofenceApi.unbindDevices(geofence.id, [deviceId])
      message.success('设备解绑成功')
      fetchDevices()
      onRefresh()
    } catch (error) {
      message.error('设备解绑失败')
    }
  }

  const getTypeTag = (type: string) => {
    switch (type) {
      case 'circle':
        return <Tag color="blue">圆形</Tag>
      case 'polygon':
        return <Tag color="green">多边形</Tag>
      default:
        return <Tag>{type}</Tag>
    }
  }

  const getAlertTypeTag = (type: string) => {
    switch (type) {
      case 'enter':
        return <Tag color="green">进入</Tag>
      case 'exit':
        return <Tag color="orange">离开</Tag>
      case 'both':
        return <Tag color="purple">进出都报警</Tag>
      default:
        return <Tag>{type}</Tag>
    }
  }

  const getEventTypeTag = (type: string) => {
    switch (type) {
      case 'enter':
        return <Tag color="green">进入</Tag>
      case 'exit':
        return <Tag color="red">离开</Tag>
      default:
        return <Tag>{type}</Tag>
    }
  }

  const deviceColumns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 80,
    },
    {
      title: '设备ID',
      dataIndex: 'device_id',
    },
    {
      title: '名称',
      dataIndex: 'name',
    },
    {
      title: '协议',
      dataIndex: 'protocol',
      render: (protocol: string) => <Tag>{protocol || 'JT808'}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      render: (status: number) => (
        <Tag color={status === 1 ? 'green' : 'red'}>
          {status === 1 ? '启用' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: Device) => (
        <Button
          type="text"
          danger
          icon={<UnlinkOutlined />}
          onClick={() => handleUnbindDevice(record.id)}
        >
          解绑
        </Button>
      ),
    },
  ]

  const eventColumns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 80,
    },
    {
      title: '设备',
      dataIndex: 'device',
      render: (device?: Device) => device?.name || '-',
    },
    {
      title: '事件类型',
      dataIndex: 'event_type',
      render: (type: string) => getEventTypeTag(type),
    },
    {
      title: '位置',
      dataIndex: 'location',
      render: (location: { lat: number; lon: number }) =>
        location ? `${location.lat.toFixed(6)}, ${location.lon.toFixed(6)}` : '-',
    },
    {
      title: '速度',
      dataIndex: 'speed',
      render: (speed: number) => (speed ? `${speed.toFixed(1)} km/h` : '-'),
    },
    {
      title: '触发时间',
      dataIndex: 'triggered_at',
      render: (time: string) => dayjs(time).format('YYYY-MM-DD HH:mm:ss'),
    },
  ]

  // Get center and points for map display
  const getMapData = () => {
    if (geofence.type === 'circle') {
      return {
        center: geofence.coordinates?.center || { lat: 39.9042, lon: 116.4074 },
        radius: geofence.coordinates?.radius || 1000,
        polygonPoints: [],
      }
    } else {
      return {
        center: { lat: 39.9042, lon: 116.4074 },
        radius: 1000,
        polygonPoints: geofence.coordinates?.points || [],
      }
    }
  }

  const mapData = getMapData()

  return (
    <div>
      <Tabs activeKey={activeTab} onChange={setActiveTab}>
        <TabPane
          tab={
            <span>
              <EnvironmentOutlined />
              基本信息
            </span>
          }
          key="info"
        >
          <Row gutter={16}>
            <Col span={12}>
              <Descriptions bordered column={1} size="small">
                <Descriptions.Item label="围栏名称">
                  {geofence.name}
                </Descriptions.Item>
                <Descriptions.Item label="围栏类型">
                  {getTypeTag(geofence.type)}
                </Descriptions.Item>
                <Descriptions.Item label="报警类型">
                  {getAlertTypeTag(geofence.alert_type)}
                </Descriptions.Item>
                <Descriptions.Item label="状态">
                  <Tag color={geofence.status === 1 ? 'green' : 'red'}>
                    {geofence.status === 1 ? '启用' : '禁用'}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label="描述">
                  {geofence.description || '-'}
                </Descriptions.Item>
                <Descriptions.Item label="创建时间">
                  {dayjs(geofence.created_at).format('YYYY-MM-DD HH:mm:ss')}
                </Descriptions.Item>
                <Descriptions.Item label="更新时间">
                  {dayjs(geofence.updated_at).format('YYYY-MM-DD HH:mm:ss')}
                </Descriptions.Item>
              </Descriptions>

              {geofence.type === 'circle' && geofence.coordinates && (
                <Descriptions bordered column={1} size="small" style={{ marginTop: 16 }}>
                  <Descriptions.Item label="圆心纬度">
                    {geofence.coordinates.center?.lat}
                  </Descriptions.Item>
                  <Descriptions.Item label="圆心经度">
                    {geofence.coordinates.center?.lon}
                  </Descriptions.Item>
                  <Descriptions.Item label="半径（米）">
                    {geofence.coordinates.radius}
                  </Descriptions.Item>
                </Descriptions>
              )}

              {geofence.type === 'polygon' && geofence.coordinates && (
                <Card size="small" title="多边形顶点" style={{ marginTop: 16 }}>
                  {geofence.coordinates.points?.map((point: any, index: number) => (
                    <div key={index} style={{ fontSize: 12, marginBottom: 4 }}>
                      点 {index + 1}: ({point.lat}, {point.lon})
                    </div>
                  ))}
                </Card>
              )}
            </Col>
            <Col span={12}>
              <Card size="small" title="围栏地图">
                <div style={{ height: 350, borderRadius: 4, overflow: 'hidden' }}>
                  <GeofenceMap
                    type={geofence.type}
                    center={mapData.center}
                    radius={mapData.radius}
                    polygonPoints={mapData.polygonPoints}
                    editable={false}
                  />
                </div>
              </Card>
            </Col>
          </Row>
        </TabPane>

        <TabPane
          tab={
            <span>
              <LinkOutlined />
              绑定设备 ({devices.length})
            </span>
          }
          key="devices"
        >
          <div style={{ marginBottom: 16 }}>
            <Button
              type="primary"
              icon={<LinkOutlined />}
              onClick={() => setBindModalVisible(true)}
            >
              绑定设备
            </Button>
          </div>

          <Table
            columns={deviceColumns}
            dataSource={devices}
            rowKey="id"
            loading={loading}
            pagination={false}
            locale={{
              emptyText: <Empty description="暂无绑定设备" />,
            }}
          />
        </TabPane>

        <TabPane
          tab={
            <span>
              <HistoryOutlined />
              事件记录
            </span>
          }
          key="events"
        >
          <Table
            columns={eventColumns}
            dataSource={events}
            rowKey="id"
            loading={loading}
            pagination={eventPagination}
            onChange={(p) => setEventPagination({ ...eventPagination, current: p.current || 1 })}
            locale={{
              emptyText: <Empty description="暂无事件记录" />,
            }}
          />
        </TabPane>
      </Tabs>

      <Modal
        title="绑定设备"
        open={bindModalVisible}
        onOk={handleBindDevices}
        onCancel={() => {
          setBindModalVisible(false)
          setSelectedDeviceIds([])
        }}
        width={600}
      >
        <Select
          mode="multiple"
          placeholder="选择要绑定的设备"
          style={{ width: '100%' }}
          value={selectedDeviceIds}
          onChange={setSelectedDeviceIds}
          optionFilterProp="children"
          showSearch
        >
          {allDevices
            .filter((d) => !devices.find((bd) => bd.id === d.id))
            .map((device) => (
              <Option key={device.id} value={device.id}>
                {device.name} ({device.device_id})
              </Option>
            ))}
        </Select>
        <div style={{ marginTop: 8, color: '#888' }}>
          已选择 {selectedDeviceIds.length} 个设备
        </div>
      </Modal>
    </div>
  )
}

export default GeofenceDetail
