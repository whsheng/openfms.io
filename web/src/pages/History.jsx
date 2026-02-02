import React, { useState } from 'react'
import { Card, Form, Select, DatePicker, Button, Table, message } from 'antd'
import { SearchOutlined, PlayCircleOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import { deviceApi, positionApi } from '../services/api'

const { RangePicker } = DatePicker

function History() {
  const [form] = Form.useForm()
  const [devices, setDevices] = useState([])
  const [positions, setPositions] = useState([])
  const [loading, setLoading] = useState(false)

  React.useEffect(() => {
    fetchDevices()
  }, [])

  const fetchDevices = async () => {
    try {
      const res = await deviceApi.getDevices({ page_size: 1000 })
      setDevices(res.data.data || [])
    } catch (error) {
      console.error('Failed to fetch devices:', error)
    }
  }

  const handleSearch = async (values) => {
    if (!values.device_id || !values.timeRange) {
      message.warning('请选择设备和时间范围')
      return
    }

    setLoading(true)
    try {
      const [start, end] = values.timeRange
      const res = await positionApi.getDeviceHistory(
        values.device_id,
        start.toISOString(),
        end.toISOString(),
        1000
      )
      setPositions(res.data.data || [])
    } catch (error) {
      message.error('获取历史轨迹失败')
    } finally {
      setLoading(false)
    }
  }

  const columns = [
    {
      title: '时间',
      dataIndex: 'time',
      render: (time) => dayjs(time).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '纬度',
      dataIndex: 'lat',
      render: (lat) => lat?.toFixed(6),
    },
    {
      title: '经度',
      dataIndex: 'lon',
      render: (lon) => lon?.toFixed(6),
    },
    {
      title: '速度',
      dataIndex: 'speed',
      render: (speed) => `${speed} km/h`,
    },
    {
      title: '方向',
      dataIndex: 'angle',
      render: (angle) => `${angle}°`,
    },
  ]

  return (
    <div>
      <h2 style={{ marginBottom: 24 }}>历史轨迹</h2>

      <Card style={{ marginBottom: 24 }}>
        <Form
          form={form}
          layout="inline"
          onFinish={handleSearch}
        >
          <Form.Item
            name="device_id"
            label="设备"
            rules={[{ required: true }]}
          >
            <Select
              style={{ width: 200 }}
              placeholder="选择设备"
              showSearch
              optionFilterProp="children"
            >
              {devices.map((device) => (
                <Select.Option key={device.id} value={device.device_id}>
                  {device.name} ({device.device_id})
                </Select.Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="timeRange"
            label="时间范围"
            rules={[{ required: true }]}
          >
            <RangePicker
              showTime
              format="YYYY-MM-DD HH:mm"
              style={{ width: 380 }}
            />
          </Form.Item>

          <Form.Item>
            <Button
              type="primary"
              htmlType="submit"
              icon={<SearchOutlined />}
              loading={loading}
            >
              查询
            </Button>
          </Form.Item>

          <Form.Item>
            <Button icon={<PlayCircleOutlined />} disabled={positions.length === 0}>
              播放轨迹
            </Button>
          </Form.Item>
        </Form>
      </Card>

      <Card title={`查询结果 (${positions.length} 条记录)`}>
        <Table
          columns={columns}
          dataSource={positions}
          rowKey={(record, index) => index}
          loading={loading}
          pagination={{
            pageSize: 50,
            showSizeChanger: true,
            showTotal: (total) => `共 ${total} 条`,
          }}
          scroll={{ y: 400 }}
        />
      </Card>
    </div>
  )
}

export default History
