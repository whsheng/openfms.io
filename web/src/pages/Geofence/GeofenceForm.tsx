import React, { useState, useEffect } from 'react'
import {
  Form,
  Input,
  Select,
  Radio,
  Button,
  Space,
  InputNumber,
  Card,
  Row,
  Col,
  message,
} from 'antd'
import GeofenceMap from './GeofenceMap'

const { TextArea } = Input
const { Option } = Select

interface GeofenceFormProps {
  initialValues?: any
  onSubmit: (values: any) => void
  onCancel: () => void
}

interface MapPoint {
  lat: number
  lon: number
}

const GeofenceForm: React.FC<GeofenceFormProps> = ({
  initialValues,
  onSubmit,
  onCancel,
}) => {
  const [form] = Form.useForm()
  const [geofenceType, setGeofenceType] = useState<'circle' | 'polygon'>('circle')
  const [center, setCenter] = useState<MapPoint>({ lat: 39.9042, lon: 116.4074 })
  const [radius, setRadius] = useState<number>(1000)
  const [polygonPoints, setPolygonPoints] = useState<MapPoint[]>([])
  const [mapMode, setMapMode] = useState<'view' | 'draw'>('view')

  useEffect(() => {
    if (initialValues) {
      form.setFieldsValue(initialValues)
      setGeofenceType(initialValues.type)
      
      if (initialValues.type === 'circle' && initialValues.coordinates) {
        setCenter(initialValues.coordinates.center)
        setRadius(initialValues.coordinates.radius)
      } else if (initialValues.type === 'polygon' && initialValues.coordinates) {
        setPolygonPoints(initialValues.coordinates.points || [])
      }
    }
  }, [initialValues, form])

  const handleTypeChange = (e: any) => {
    setGeofenceType(e.target.value)
    setPolygonPoints([])
  }

  const handleMapClick = (point: MapPoint) => {
    if (geofenceType === 'circle') {
      setCenter(point)
    } else if (geofenceType === 'polygon') {
      setPolygonPoints([...polygonPoints, point])
    }
  }

  const handleSubmit = () => {
    form.validateFields().then((values) => {
      let coordinates: any

      if (geofenceType === 'circle') {
        if (!center) {
          message.error('请在地图上选择圆心')
          return
        }
        coordinates = {
          center: center,
          radius: radius,
        }
      } else {
        if (polygonPoints.length < 3) {
          message.error('多边形至少需要3个点')
          return
        }
        coordinates = {
          points: polygonPoints,
        }
      }

      const submitData = {
        ...values,
        type: geofenceType,
        coordinates,
      }

      onSubmit(submitData)
    })
  }

  const clearPolygonPoints = () => {
    setPolygonPoints([])
  }

  const removeLastPoint = () => {
    setPolygonPoints(polygonPoints.slice(0, -1))
  }

  return (
    <div>
      <Form
        form={form}
        layout="vertical"
        initialValues={{
          type: 'circle',
          alert_type: 'both',
          status: 1,
        }}
      >
        <Row gutter={16}>
          <Col span={12}>
            <Form.Item
              name="name"
              label="围栏名称"
              rules={[{ required: true, message: '请输入围栏名称' }]}
            >
              <Input placeholder="例如：公司区域围栏" />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              name="alert_type"
              label="报警类型"
              rules={[{ required: true }]}
            >
              <Select>
                <Option value="enter">进入报警</Option>
                <Option value="exit">离开报警</Option>
                <Option value="both">进出都报警</Option>
              </Select>
            </Form.Item>
          </Col>
        </Row>

        <Form.Item name="description" label="描述">
          <TextArea rows={2} placeholder="围栏描述（可选）" />
        </Form.Item>

        <Form.Item label="围栏类型" required>
          <Radio.Group value={geofenceType} onChange={handleTypeChange}>
            <Radio.Button value="circle">圆形围栏</Radio.Button>
            <Radio.Button value="polygon">多边形围栏</Radio.Button>
          </Radio.Group>
        </Form.Item>

        {geofenceType === 'circle' && (
          <Card size="small" title="圆形围栏设置" style={{ marginBottom: 16 }}>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item label="圆心纬度">
                  <InputNumber
                    value={center.lat}
                    onChange={(val) => setCenter({ ...center, lat: val || 0 })}
                    precision={6}
                    style={{ width: '100%' }}
                    step={0.0001}
                  />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item label="圆心经度">
                  <InputNumber
                    value={center.lon}
                    onChange={(val) => setCenter({ ...center, lon: val || 0 })}
                    precision={6}
                    style={{ width: '100%' }}
                    step={0.0001}
                  />
                </Form.Item>
              </Col>
            </Row>
            <Form.Item label="半径（米）">
              <InputNumber
                value={radius}
                onChange={(val) => setRadius(val || 100)}
                min={10}
                max={100000}
                style={{ width: '100%' }}
                step={100}
              />
            </Form.Item>
            <div style={{ color: '#888', fontSize: 12 }}>
              提示：在地图上点击可设置圆心位置
            </div>
          </Card>
        )}

        {geofenceType === 'polygon' && (
          <Card
            size="small"
            title="多边形围栏设置"
            style={{ marginBottom: 16 }}
            extra={
              <Space>
                <span style={{ fontSize: 12 }}>
                  已选 {polygonPoints.length} 个点
                </span>
                <Button size="small" onClick={removeLastPoint} disabled={polygonPoints.length === 0}>
                  撤销
                </Button>
                <Button size="small" danger onClick={clearPolygonPoints} disabled={polygonPoints.length === 0}>
                  清空
                </Button>
              </Space>
            }
          >
            <div style={{ color: '#888', fontSize: 12, marginBottom: 8 }}>
              提示：在地图上点击添加多边形顶点，至少需要3个点
            </div>
            {polygonPoints.length > 0 && (
              <div style={{ maxHeight: 100, overflow: 'auto' }}>
                {polygonPoints.map((point, index) => (
                  <div key={index} style={{ fontSize: 12, color: '#666' }}>
                    点 {index + 1}: ({point.lat.toFixed(6)}, {point.lon.toFixed(6)})
                  </div>
                ))}
              </div>
            )}
          </Card>
        )}

        <Card
          size="small"
          title="地图绘制"
          extra={
            <Radio.Group
              value={mapMode}
              onChange={(e) => setMapMode(e.target.value)}
              size="small"
            >
              <Radio.Button value="view">查看</Radio.Button>
              <Radio.Button value="draw">绘制</Radio.Button>
            </Radio.Group>
          }
        >
          <div style={{ height: 300, borderRadius: 4, overflow: 'hidden' }}>
            <GeofenceMap
              type={geofenceType}
              center={center}
              radius={radius}
              polygonPoints={polygonPoints}
              onMapClick={mapMode === 'draw' ? handleMapClick : undefined}
              editable={mapMode === 'draw'}
            />
          </div>
        </Card>

        <Form.Item style={{ marginTop: 24, marginBottom: 0 }}>
          <Space style={{ float: 'right' }}>
            <Button onClick={onCancel}>取消</Button>
            <Button type="primary" onClick={handleSubmit}>
              {initialValues ? '更新' : '创建'}
            </Button>
          </Space>
        </Form.Item>
      </Form>
    </div>
  )
}

export default GeofenceForm
