import React, { useEffect, useState } from 'react'
import { Table, Button, Space, Tag, Modal, Form, Input, Select, message, Popconfirm } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons'
import { deviceApi } from '../services/api'

function Devices() {
  const [devices, setDevices] = useState([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [editingDevice, setEditingDevice] = useState(null)
  const [form] = Form.useForm()
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 20,
    total: 0,
  })

  useEffect(() => {
    fetchDevices()
  }, [pagination.current, pagination.pageSize])

  const fetchDevices = async () => {
    setLoading(true)
    try {
      const res = await deviceApi.getDevices({
        page: pagination.current,
        page_size: pagination.pageSize,
      })
      setDevices(res.data.data || [])
      setPagination(prev => ({ ...prev, total: res.data.total || 0 }))
    } catch (error) {
      message.error('获取设备列表失败')
    } finally {
      setLoading(false)
    }
  }

  const handleAdd = () => {
    setEditingDevice(null)
    form.resetFields()
    setModalVisible(true)
  }

  const handleEdit = (record) => {
    setEditingDevice(record)
    form.setFieldsValue(record)
    setModalVisible(true)
  }

  const handleDelete = async (id) => {
    try {
      await deviceApi.deleteDevice(id)
      message.success('删除成功')
      fetchDevices()
    } catch (error) {
      message.error('删除失败')
    }
  }

  const handleSubmit = async (values) => {
    try {
      if (editingDevice) {
        await deviceApi.updateDevice(editingDevice.id, values)
        message.success('更新成功')
      } else {
        await deviceApi.createDevice(values)
        message.success('创建成功')
      }
      setModalVisible(false)
      fetchDevices()
    } catch (error) {
      message.error(editingDevice ? '更新失败' : '创建失败')
    }
  }

  const columns = [
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
      render: (protocol) => <Tag>{protocol || 'JT808'}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      render: (status) => (
        <Tag color={status === 1 ? 'green' : 'red'}>
          {status === 1 ? '启用' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '最后在线',
      dataIndex: 'last_online',
    },
    {
      title: '操作',
      key: 'action',
      width: 150,
      render: (_, record) => (
        <Space>
          <Button
            type="text"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          />
          <Popconfirm
            title="确认删除"
            description="删除后无法恢复，是否继续？"
            onConfirm={() => handleDelete(record.id)}
            okText="确认"
            cancelText="取消"
          >
            <Button type="text" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
        <h2>设备管理</h2>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
          添加设备
        </Button>
      </div>

      <Table
        columns={columns}
        dataSource={devices}
        rowKey="id"
        loading={loading}
        pagination={pagination}
        onChange={(p) => setPagination(p)}
      />

      <Modal
        title={editingDevice ? '编辑设备' : '添加设备'}
        open={modalVisible}
        onOk={() => form.submit()}
        onCancel={() => setModalVisible(false)}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
        >
          <Form.Item
            name="device_id"
            label="设备ID"
            rules={[{ required: true, message: '请输入设备ID' }]}
          >
            <Input placeholder="例如: 13900000001" disabled={!!editingDevice} />
          </Form.Item>
          <Form.Item
            name="name"
            label="设备名称"
            rules={[{ required: true, message: '请输入设备名称' }]}
          >
            <Input placeholder="例如: 车辆01" />
          </Form.Item>
          <Form.Item
            name="protocol"
            label="协议类型"
            initialValue="JT808"
          >
            <Select>
              <Select.Option value="JT808">JT808</Select.Option>
              <Select.Option value="GT06">GT06</Select.Option>
              <Select.Option value="Wialon">Wialon</Select.Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default Devices
