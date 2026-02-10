import React, { useState, useRef } from 'react'
import {
  Modal,
  Upload,
  Button,
  Steps,
  Table,
  Progress,
  Alert,
  Space,
  Typography,
  message,
  Card,
  Row,
  Col,
  Statistic,
  Tag,
} from 'antd'
import {
  UploadOutlined,
  DownloadOutlined,
  FileExcelOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  LoadingOutlined,
} from '@ant-design/icons'
import { deviceApi } from '../services/api'

const { Step } = Steps
const { Text, Title } = Typography
const { Dragger } = Upload

// 导入状态常量
const IMPORT_STATUS = {
  PENDING: 'pending',
  PROCESSING: 'processing',
  COMPLETED: 'completed',
  FAILED: 'failed',
}

const DeviceImportModal = ({ visible, onCancel, onSuccess }) => {
  const [currentStep, setCurrentStep] = useState(0)
  const [file, setFile] = useState(null)
  const [previewData, setPreviewData] = useState(null)
  const [importTask, setImportTask] = useState(null)
  const [importStatus, setImportStatus] = useState(null)
  const [loading, setLoading] = useState(false)
  const [pollingTimer, setPollingTimer] = useState(null)
  const fileInputRef = useRef(null)

  // 下载模板
  const handleDownloadTemplate = async () => {
    try {
      setLoading(true)
      const response = await deviceApi.downloadImportTemplate()
      const blob = new Blob([response.data], {
        type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
      })
      const url = window.URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = '设备导入模板.xlsx'
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(url)
      message.success('模板下载成功')
    } catch (error) {
      message.error('模板下载失败')
    } finally {
      setLoading(false)
    }
  }

  // 文件上传配置
  const uploadProps = {
    accept: '.xlsx,.xls',
    maxCount: 1,
    beforeUpload: (file) => {
      const isExcel =
        file.type === 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' ||
        file.type === 'application/vnd.ms-excel' ||
        file.name.endsWith('.xlsx') ||
        file.name.endsWith('.xls')
      if (!isExcel) {
        message.error('请上传Excel文件(.xlsx或.xls)')
        return Upload.LIST_IGNORE
      }
      setFile(file)
      return false
    },
    onRemove: () => {
      setFile(null)
    },
  }

  // 预览导入数据
  const handlePreview = async () => {
    if (!file) {
      message.warning('请先选择文件')
      return
    }

    try {
      setLoading(true)
      const formData = new FormData()
      formData.append('file', file)

      const response = await deviceApi.previewImport(formData)
      setPreviewData(response.data)
      setCurrentStep(1)
    } catch (error) {
      message.error(error.response?.data?.error || '预览失败')
    } finally {
      setLoading(false)
    }
  }

  // 开始导入
  const handleImport = async () => {
    if (!file) {
      message.warning('请先选择文件')
      return
    }

    try {
      setLoading(true)
      const formData = new FormData()
      formData.append('file', file)

      const response = await deviceApi.importDevices(formData)
      setImportTask(response.data)
      setCurrentStep(2)

      // 开始轮询状态
      startPolling(response.data.task_id)
    } catch (error) {
      message.error(error.response?.data?.error || '导入失败')
    } finally {
      setLoading(false)
    }
  }

  // 轮询导入状态
  const startPolling = (taskId) => {
    const timer = setInterval(async () => {
      try {
        const response = await deviceApi.getImportStatus(taskId)
        const status = response.data
        setImportStatus(status)

        if (status.status === IMPORT_STATUS.COMPLETED || status.status === IMPORT_STATUS.FAILED) {
          clearInterval(timer)
          setPollingTimer(null)
          if (status.status === IMPORT_STATUS.COMPLETED && status.error_count === 0) {
            message.success('导入成功')
            onSuccess?.()
          }
        }
      } catch (error) {
        console.error('获取导入状态失败:', error)
      }
    }, 1000)

    setPollingTimer(timer)
  }

  // 关闭模态框
  const handleClose = () => {
    if (pollingTimer) {
      clearInterval(pollingTimer)
      setPollingTimer(null)
    }
    resetState()
    onCancel()
  }

  // 重置状态
  const resetState = () => {
    setCurrentStep(0)
    setFile(null)
    setPreviewData(null)
    setImportTask(null)
    setImportStatus(null)
    setLoading(false)
  }

  // 预览表格列
  const previewColumns = [
    {
      title: '行号',
      dataIndex: 'row_num',
      width: 70,
    },
    {
      title: '设备号',
      dataIndex: 'device_id',
    },
    {
      title: '设备名称',
      dataIndex: 'name',
    },
    {
      title: '协议类型',
      dataIndex: 'protocol',
      render: (protocol) => <Tag>{protocol}</Tag>,
    },
    {
      title: '绑定车牌',
      dataIndex: 'vehicle_plate',
      render: (plate) => plate || '-',
    },
    {
      title: '状态',
      dataIndex: 'valid',
      width: 100,
      render: (valid, record) =>
        valid ? (
          <Tag color="success" icon={<CheckCircleOutlined />}>
            有效
          </Tag>
        ) : (
          <Tag color="error" icon={<CloseCircleOutlined />}>
            错误
          </Tag>
        ),
    },
    {
      title: '错误信息',
      dataIndex: 'error',
      render: (error) => error || '-',
    },
  ]

  // 步骤内容
  const renderStepContent = () => {
    switch (currentStep) {
      case 0:
        return (
          <Space direction="vertical" style={{ width: '100%' }} size="large">
            <Alert
              message="导入说明"
              description={
                <ul style={{ margin: 0, paddingLeft: 20 }}>
                  <li>请使用提供的模板填写设备信息</li>
                  <li>设备号、设备名称、协议类型为必填项</li>
                  <li>支持批量导入，单次最多导入10000条记录</li>
                  <li>重复的设备号将被跳过</li>
                </ul>
              }
              type="info"
              showIcon
            />

            <Card>
              <Space direction="vertical" style={{ width: '100%' }}>
                <Title level={5}>
                  <FileExcelOutlined /> 1. 下载导入模板
                </Title>
                <Text type="secondary">下载Excel模板，按格式填写设备信息</Text>
                <Button
                  type="primary"
                  icon={<DownloadOutlined />}
                  onClick={handleDownloadTemplate}
                  loading={loading}
                >
                  下载模板
                </Button>
              </Space>
            </Card>

            <Card>
              <Space direction="vertical" style={{ width: '100%' }}>
                <Title level={5}>
                  <UploadOutlined /> 2. 上传文件
                </Title>
                <Text type="secondary">选择填写好的Excel文件进行上传</Text>
                <Dragger {...uploadProps}>
                  <p className="ant-upload-drag-icon">
                    <UploadOutlined />
                  </p>
                  <p className="ant-upload-text">点击或拖拽文件到此区域上传</p>
                  <p className="ant-upload-hint">支持 .xlsx, .xls 格式的Excel文件</p>
                </Dragger>
              </Space>
            </Card>

            <div style={{ textAlign: 'center' }}>
              <Button
                type="primary"
                size="large"
                onClick={handlePreview}
                loading={loading}
                disabled={!file}
              >
                下一步：预览数据
              </Button>
            </div>
          </Space>
        )

      case 1:
        return (
          <Space direction="vertical" style={{ width: '100%' }} size="large">
            {previewData && (
              <>
                <Row gutter={16}>
                  <Col span={8}>
                    <Statistic
                      title="总记录数"
                      value={previewData.total}
                      valueStyle={{ color: '#1890ff' }}
                    />
                  </Col>
                  <Col span={8}>
                    <Statistic
                      title="有效记录"
                      value={previewData.valid_count}
                      valueStyle={{ color: '#52c41a' }}
                    />
                  </Col>
                  <Col span={8}>
                    <Statistic
                      title="错误记录"
                      value={previewData.invalid_count}
                      valueStyle={{ color: previewData.invalid_count > 0 ? '#ff4d4f' : '#52c41a' }}
                    />
                  </Col>
                </Row>

                {previewData.invalid_count > 0 && (
                  <Alert
                    message={`发现 ${previewData.invalid_count} 条错误记录，请修正后重新上传`}
                    type="warning"
                    showIcon
                  />
                )}

                <Table
                  columns={previewColumns}
                  dataSource={previewData.preview}
                  rowKey="row_num"
                  pagination={false}
                  size="small"
                  scroll={{ y: 300 }}
                />

                <div style={{ textAlign: 'center' }}>
                  <Space>
                    <Button onClick={() => setCurrentStep(0)}>上一步</Button>
                    <Button
                      type="primary"
                      onClick={handleImport}
                      loading={loading}
                      disabled={previewData.invalid_count > 0}
                    >
                      开始导入
                    </Button>
                  </Space>
                </div>
              </>
            )}
          </Space>
        )

      case 2:
        return (
          <Space direction="vertical" style={{ width: '100%' }} size="large">
            {importStatus ? (
              <>
                <div style={{ textAlign: 'center', padding: '20px 0' }}>
                  {importStatus.status === IMPORT_STATUS.PROCESSING ? (
                    <LoadingOutlined style={{ fontSize: 48, color: '#1890ff' }} />
                  ) : importStatus.error_count === 0 ? (
                    <CheckCircleOutlined style={{ fontSize: 48, color: '#52c41a' }} />
                  ) : (
                    <CloseCircleOutlined style={{ fontSize: 48, color: '#ff4d4f' }} />
                  )}
                  <Title level={4} style={{ marginTop: 16 }}>
                    {importStatus.status === IMPORT_STATUS.PROCESSING
                      ? '导入中...'
                      : importStatus.error_count === 0
                      ? '导入完成'
                      : '导入完成（部分失败）'}
                  </Title>
                </div>

                <Progress
                  percent={importStatus.progress}
                  status={
                    importStatus.status === IMPORT_STATUS.PROCESSING
                      ? 'active'
                      : importStatus.error_count === 0
                      ? 'success'
                      : 'exception'
                  }
                />

                <Row gutter={16}>
                  <Col span={8}>
                    <Statistic title="总记录数" value={importStatus.total_count} />
                  </Col>
                  <Col span={8}>
                    <Statistic
                      title="成功导入"
                      value={importStatus.success_count}
                      valueStyle={{ color: '#52c41a' }}
                    />
                  </Col>
                  <Col span={8}>
                    <Statistic
                      title="导入失败"
                      value={importStatus.error_count}
                      valueStyle={{ color: importStatus.error_count > 0 ? '#ff4d4f' : '#52c41a' }}
                    />
                  </Col>
                </Row>

                {importStatus.errors && importStatus.errors.length > 0 && (
                  <Alert
                    message="错误详情"
                    description={
                      <ul style={{ margin: 0, paddingLeft: 20, maxHeight: 200, overflow: 'auto' }}>
                        {importStatus.errors.slice(0, 10).map((err, index) => (
                          <li key={index}>
                            第{err.row_num}行: {err.error}
                          </li>
                        ))}
                        {importStatus.errors.length > 10 && (
                          <li>...还有 {importStatus.errors.length - 10} 条错误</li>
                        )}
                      </ul>
                    }
                    type="error"
                  />
                )}

                <div style={{ textAlign: 'center' }}>
                  <Button type="primary" onClick={handleClose}>
                    完成
                  </Button>
                </div>
              </>
            ) : (
              <div style={{ textAlign: 'center', padding: '40px 0' }}>
                <LoadingOutlined style={{ fontSize: 48, color: '#1890ff' }} />
                <p style={{ marginTop: 16 }}>正在启动导入任务...</p>
              </div>
            )}
          </Space>
        )

      default:
        return null
    }
  }

  return (
    <Modal
      title="批量导入设备"
      open={visible}
      onCancel={handleClose}
      footer={null}
      width={800}
      destroyOnClose
    >
      <Steps current={currentStep} style={{ marginBottom: 24 }}>
        <Step title="上传文件" />
        <Step title="预览数据" />
        <Step title="导入进度" />
      </Steps>

      <div style={{ minHeight: 400 }}>{renderStepContent()}</div>
    </Modal>
  )
}

export default DeviceImportModal
