import React, { useEffect, useState } from 'react'
import { Row, Col, Card, Statistic, Table, Tag } from 'antd'
import {
  CarOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined,
  GlobalOutlined,
} from '@ant-design/icons'
import { deviceApi, positionApi } from '../services/api'

function Dashboard() {
  const [stats, setStats] = useState({
    totalDevices: 0,
    onlineDevices: 0,
    offlineDevices: 0,
  })
  const [latestPositions, setLatestPositions] = useState([])
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    fetchData()
    const interval = setInterval(fetchData, 30000) // Refresh every 30s
    return () => clearInterval(interval)
  }, [])

  const fetchData = async () => {
    try {
      // Fetch devices
      const devicesRes = await deviceApi.getDevices({ page_size: 1 })
      const total = devicesRes.data.total || 0

      // Fetch latest positions
      const positionsRes = await positionApi.getLatestPositions()
      const positions = positionsRes.data.data || []

      setStats({
        totalDevices: total,
        onlineDevices: positions.length,
        offlineDevices: total - positions.length,
      })
      setLatestPositions(positions.slice(0, 10))
    } catch (error) {
      console.error('Failed to fetch dashboard data:', error)
    }
  }

  const columns = [
    {
      title: '设备ID',
      dataIndex: 'device_id',
      key: 'device_id',
    },
    {
      title: '位置',
      key: 'location',
      render: (_, record) => (
        <span>
          {record.lat?.toFixed(6)}, {record.lon?.toFixed(6)}
        </span>
      ),
    },
    {
      title: '速度',
      key: 'speed',
      render: (_, record) => (
        <span>{record.speed} km/h</span>
      ),
    },
    {
      title: '状态',
      key: 'status',
      render: (_, record) => (
        <Tag color={record.speed > 0 ? 'green' : 'blue'}>
          {record.speed > 0 ? '行驶中' : '静止'}
        </Tag>
      ),
    },
    {
      title: '更新时间',
      dataIndex: 'time',
      key: 'time',
    },
  ]

  return (
    <div>
      <h2 style={{ marginBottom: 24 }}>仪表盘</h2>
      
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="设备总数"
              value={stats.totalDevices}
              prefix={<CarOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="在线设备"
              value={stats.onlineDevices}
              valueStyle={{ color: '#3f8600' }}
              prefix={<CheckCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="离线设备"
              value={stats.offlineDevices}
              valueStyle={{ color: '#cf1322' }}
              prefix={<ExclamationCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="在线率"
              value={stats.totalDevices > 0 
                ? Math.round((stats.onlineDevices / stats.totalDevices) * 100) 
                : 0}
              suffix="%"
              prefix={<GlobalOutlined />}
            />
          </Card>
        </Col>
      </Row>

      <Card title="实时位置">
        <Table
          columns={columns}
          dataSource={latestPositions}
          rowKey="device_id"
          loading={loading}
          pagination={false}
        />
      </Card>
    </div>
  )
}

export default Dashboard
