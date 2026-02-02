import React, { useEffect, useState, useRef } from 'react'
import { Card, List, Tag, Badge, Space, Typography } from 'antd'
import { CarOutlined } from '@ant-design/icons'
import { deviceApi, positionApi } from '../services/api'

const { Text } = Typography

function RealtimeMap() {
  const [devices, setDevices] = useState([])
  const [selectedDevice, setSelectedDevice] = useState(null)
  const mapRef = useRef(null)
  const markersRef = useRef({})

  useEffect(() => {
    fetchDevices()
    const interval = setInterval(fetchDevices, 10000) // Refresh every 10s
    return () => clearInterval(interval)
  }, [])

  const fetchDevices = async () => {
    try {
      const res = await positionApi.getLatestPositions()
      const positions = res.data.data || []
      setDevices(positions)
    } catch (error) {
      console.error('Failed to fetch devices:', error)
    }
  }

  const handleDeviceClick = (device) => {
    setSelectedDevice(device)
  }

  return (
    <div style={{ height: 'calc(100vh - 112px)', display: 'flex', gap: 16 }}>
      {/* Device List */}
      <Card 
        title={`在线设备 (${devices.length})`}
        style={{ width: 320, overflow: 'auto' }}
        bodyStyle={{ padding: 0 }}
      >
        <List
          dataSource={devices}
          renderItem={(item) => (
            <List.Item
              style={{
                cursor: 'pointer',
                backgroundColor: selectedDevice?.device_id === item.device_id ? '#e6f7ff' : 'transparent',
                padding: '12px 16px',
              }}
              onClick={() => handleDeviceClick(item)}
            >
              <List.Item.Meta
                avatar={<CarOutlined style={{ fontSize: 24, color: '#1890ff' }} />}
                title={
                  <Space>
                    <Text strong>{item.device_id}</Text>
                    <Badge status={item.speed > 0 ? 'processing' : 'default'} />
                  </Space>
                }
                description={
                  <Space direction="vertical" size={0}>
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      速度: {item.speed} km/h
                    </Text>
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      位置: {item.lat?.toFixed(4)}, {item.lon?.toFixed(4)}
                    </Text>
                  </Space>
                }
              />
            </List.Item>
          )}
        />
      </Card>

      {/* Map Placeholder */}
      <Card style={{ flex: 1, position: 'relative' }} bodyStyle={{ height: '100%', padding: 0 }}>
        <div
          ref={mapRef}
          style={{
            width: '100%',
            height: '100%',
            background: '#f5f5f5',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            flexDirection: 'column',
          }}
        >
          <CarOutlined style={{ fontSize: 64, color: '#d9d9d9', marginBottom: 16 }} />
          <Text type="secondary">地图组件 (需要配置 Mapbox Token)</Text>
          {selectedDevice && (
            <div style={{ marginTop: 24, textAlign: 'center' }}>
              <Text strong>选中设备: {selectedDevice.device_id}</Text>
              <br />
              <Text type="secondary">
                坐标: {selectedDevice.lat?.toFixed(6)}, {selectedDevice.lon?.toFixed(6)}
              </Text>
            </div>
          )}
        </div>
      </Card>
    </div>
  )
}

export default RealtimeMap
