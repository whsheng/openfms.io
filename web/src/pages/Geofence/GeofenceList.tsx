import React, { useEffect, useState } from 'react'
import {
  Table,
  Button,
  Space,
  Tag,
  Modal,
  message,
  Popconfirm,
  Card,
  Row,
  Col,
  Statistic,
} from 'antd'
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  EyeOutlined,
  EnvironmentOutlined,
} from '@ant-design/icons'
import { geofenceApi } from '../../services/api'
import GeofenceForm from './GeofenceForm'
import GeofenceDetail from './GeofenceDetail'

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

const GeofenceList: React.FC = () => {
  const [geofences, setGeofences] = useState<Geofence[]>([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [detailVisible, setDetailVisible] = useState(false)
  const [editingGeofence, setEditingGeofence] = useState<Geofence | null>(null)
  const [selectedGeofence, setSelectedGeofence] = useState<Geofence | null>(null)
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 10,
    total: 0,
  })

  useEffect(() => {
    fetchGeofences()
  }, [pagination.current, pagination.pageSize])

  const fetchGeofences = async () => {
    setLoading(true)
    try {
      const res = await geofenceApi.getGeofences({
        page: pagination.current,
        page_size: pagination.pageSize,
      })
      setGeofences(res.data.data || [])
      setPagination((prev) => ({ ...prev, total: res.data.total || 0 }))
    } catch (error) {
      message.error('获取围栏列表失败')
    } finally {
      setLoading(false)
    }
  }

  const handleAdd = () => {
    setEditingGeofence(null)
    setModalVisible(true)
  }

  const handleEdit = (record: Geofence) => {
    setEditingGeofence(record)
    setModalVisible(true)
  }

  const handleView = (record: Geofence) => {
    setSelectedGeofence(record)
    setDetailVisible(true)
  }

  const handleDelete = async (id: number) => {
    try {
      await geofenceApi.deleteGeofence(id)
      message.success('删除成功')
      fetchGeofences()
    } catch (error) {
      message.error('删除失败')
    }
  }

  const handleFormSubmit = async (values: any) => {
    try {
      if (editingGeofence) {
        await geofenceApi.updateGeofence(editingGeofence.id, values)
        message.success('更新成功')
      } else {
        await geofenceApi.createGeofence(values)
        message.success('创建成功')
      }
      setModalVisible(false)
      fetchGeofences()
    } catch (error) {
      message.error(editingGeofence ? '更新失败' : '创建失败')
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
        return <Tag color="purple">进出</Tag>
      default:
        return <Tag>{type}</Tag>
    }
  }

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 80,
    },
    {
      title: '名称',
      dataIndex: 'name',
    },
    {
      title: '类型',
      dataIndex: 'type',
      render: (type: string) => getTypeTag(type),
    },
    {
      title: '报警类型',
      dataIndex: 'alert_type',
      render: (type: string) => getAlertTypeTag(type),
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
      title: '创建时间',
      dataIndex: 'created_at',
      render: (time: string) => new Date(time).toLocaleString(),
    },
    {
      title: '操作',
      key: 'action',
      width: 200,
      render: (_: any, record: Geofence) => (
        <Space>
          <Button
            type="text"
            icon={<EyeOutlined />}
            onClick={() => handleView(record)}
            title="查看详情"
          />
          <Button
            type="text"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
            title="编辑"
          />
          <Popconfirm
            title="确认删除"
            description="删除后无法恢复，是否继续？"
            onConfirm={() => handleDelete(record.id)}
            okText="确认"
            cancelText="取消"
          >
            <Button type="text" danger icon={<DeleteOutlined />} title="删除" />
          </Popconfirm>
        </Space>
      ),
    },
  ]

  const activeCount = geofences.filter((g) => g.status === 1).length
  const circleCount = geofences.filter((g) => g.type === 'circle').length
  const polygonCount = geofences.filter((g) => g.type === 'polygon').length

  return (
    <div>
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="围栏总数"
              value={geofences.length}
              prefix={<EnvironmentOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="启用中"
              value={activeCount}
              valueStyle={{ color: '#3f8600' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="圆形围栏" value={circleCount} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="多边形围栏" value={polygonCount} />
          </Card>
        </Col>
      </Row>

      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
        <h2>电子围栏管理</h2>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
          创建围栏
        </Button>
      </div>

      <Table
        columns={columns}
        dataSource={geofences}
        rowKey="id"
        loading={loading}
        pagination={pagination}
        onChange={(p) => setPagination(p)}
      />

      <Modal
        title={editingGeofence ? '编辑围栏' : '创建围栏'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={800}
        destroyOnClose
      >
        <GeofenceForm
          initialValues={editingGeofence}
          onSubmit={handleFormSubmit}
          onCancel={() => setModalVisible(false)}
        />
      </Modal>

      <Modal
        title="围栏详情"
        open={detailVisible}
        onCancel={() => setDetailVisible(false)}
        footer={null}
        width={900}
        destroyOnClose
      >
        {selectedGeofence && (
          <GeofenceDetail
            geofence={selectedGeofence}
            onRefresh={fetchGeofences}
          />
        )}
      </Modal>
    </div>
  )
}

export default GeofenceList
