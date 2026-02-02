import React from 'react'
import { Card, Form, Input, Button, message, Tabs } from 'antd'

function Settings() {
  const [form] = Form.useForm()

  const handleSave = (values) => {
    message.success('设置已保存')
  }

  const items = [
    {
      key: 'general',
      label: '基本设置',
      children: (
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSave}
          initialValues={{
            companyName: '我的公司',
            timezone: 'Asia/Shanghai',
          }}
        >
          <Form.Item
            name="companyName"
            label="公司名称"
          >
            <Input />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit">
              保存设置
            </Button>
          </Form.Item>
        </Form>
      ),
    },
    {
      key: 'map',
      label: '地图设置',
      children: (
        <Form layout="vertical">
          <Form.Item label="地图提供商">
            <Input placeholder="Mapbox / OpenStreetMap" defaultValue="OpenStreetMap" />
          </Form.Item>
          <Form.Item label="Mapbox Token">
            <Input.Password placeholder="pk.xxxxxx" />
          </Form.Item>
          <Form.Item>
            <Button type="primary">保存设置</Button>
          </Form.Item>
        </Form>
      ),
    },
    {
      key: 'notifications',
      label: '通知设置',
      children: (
        <div>
          <p>通知设置功能开发中...</p>
        </div>
      ),
    },
  ]

  return (
    <div>
      <h2 style={{ marginBottom: 24 }}>系统设置</h2>
      <Card>
        <Tabs items={items} />
      </Card>
    </div>
  )
}

export default Settings
