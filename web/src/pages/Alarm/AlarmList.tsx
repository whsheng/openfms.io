import React, { useState, useEffect, useCallback } from 'react';
import {
  Table,
  Tag,
  Button,
  Space,
  Modal,
  message,
  Dropdown,
  Select,
  DatePicker,
  Input,
  Badge,
  Tooltip,
} from 'antd';
import {
  CheckOutlined,
  EyeOutlined,
  DeleteOutlined,
  MoreOutlined,
  ReloadOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { alarmApi } from '../../services/api';
import type { Alarm, AlarmType, AlarmLevel, AlarmStatus } from '../../types/alarm';
import styles from './alarm.module.css';

const { RangePicker } = DatePicker;
const { Option } = Select;

interface AlarmListProps {
  refreshKey: number;
  onStatsChange: () => void;
}

const AlarmList: React.FC<AlarmListProps> = ({ refreshKey, onStatsChange }) => {
  const [alarms, setAlarms] = useState<Alarm[]>([]);
  const [loading, setLoading] = useState(false);
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 20,
    total: 0,
  });
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [filters, setFilters] = useState({
    type: undefined as AlarmType | undefined,
    level: undefined as AlarmLevel | undefined,
    status: undefined as AlarmStatus | undefined,
    dateRange: null as [moment.Moment, moment.Moment] | null,
    search: '',
  });
  const [alarmTypes, setAlarmTypes] = useState<{ value: string; label: string; level: string }[]>([]);

  // 获取报警类型
  useEffect(() => {
    alarmApi.getTypes().then(res => {
      setAlarmTypes(res.data);
    });
  }, []);

  // 获取报警列表
  const fetchAlarms = useCallback(async () => {
    setLoading(true);
    try {
      const params: any = {
        page: pagination.current,
        page_size: pagination.pageSize,
      };
      if (filters.type) params.type = filters.type;
      if (filters.level) params.level = filters.level;
      if (filters.status) params.status = filters.status;
      if (filters.search) params.search = filters.search;
      if (filters.dateRange) {
        params.start_time = filters.dateRange[0].format('YYYY-MM-DD HH:mm:ss');
        params.end_time = filters.dateRange[1].format('YYYY-MM-DD HH:mm:ss');
      }

      const res = await alarmApi.getList(params);
      setAlarms(res.data.list);
      setPagination(prev => ({
        ...prev,
        total: res.data.total,
      }));
    } catch (error) {
      message.error('获取报警列表失败');
    } finally {
      setLoading(false);
    }
  }, [pagination.current, pagination.pageSize, filters, refreshKey]);

  useEffect(() => {
    fetchAlarms();
  }, [fetchAlarms]);

  // 标记已读
  const handleMarkAsRead = async (id: number) => {
    try {
      await alarmApi.markAsRead(id);
      message.success('已标记为已读');
      fetchAlarms();
      onStatsChange();
    } catch (error) {
      message.error('操作失败');
    }
  };

  // 处理报警
  const handleResolve = async (id: number) => {
    Modal.confirm({
      title: '处理报警',
      content: (
        <Input.TextArea
          id="resolve-note"
          placeholder="请输入处理备注（可选）"
          rows={3}
        />
      ),
      onOk: async () => {
        const note = (document.getElementById('resolve-note') as HTMLTextAreaElement)?.value;
        try {
          await alarmApi.resolve(id, note);
          message.success('报警已处理');
          fetchAlarms();
          onStatsChange();
        } catch (error) {
          message.error('操作失败');
        }
      },
    });
  };

  // 批量处理
  const handleBatchResolve = () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请先选择报警记录');
      return;
    }
    Modal.confirm({
      title: '批量处理',
      content: `确定要处理选中的 ${selectedRowKeys.length} 条报警吗？`,
      onOk: async () => {
        try {
          await alarmApi.batchResolve(selectedRowKeys as number[]);
          message.success('批量处理成功');
          setSelectedRowKeys([]);
          fetchAlarms();
          onStatsChange();
        } catch (error) {
          message.error('操作失败');
        }
      },
    });
  };

  // 批量标记已读
  const handleBatchRead = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请先选择报警记录');
      return;
    }
    try {
      await alarmApi.batchRead(selectedRowKeys as number[]);
      message.success('批量标记已读成功');
      setSelectedRowKeys([]);
      fetchAlarms();
      onStatsChange();
    } catch (error) {
      message.error('操作失败');
    }
  };

  // 查看详情
  const handleViewDetail = (alarm: Alarm) => {
    Modal.info({
      title: '报警详情',
      width: 600,
      content: (
        <div className={styles.alarmDetail}>
          <p><strong>报警类型：</strong>{getAlarmTypeLabel(alarm.type)}</p>
          <p><strong>报警级别：</strong>{getLevelTag(alarm.level)}</p>
          <p><strong>设备名称：</strong>{alarm.device_name || alarm.device_id}</p>
          <p><strong>报警标题：</strong>{alarm.title}</p>
          <p><strong>报警内容：</strong>{alarm.content}</p>
          {alarm.speed && <p><strong>当前速度：</strong>{alarm.speed} km/h</p>}
          {alarm.geofence_name && <p><strong>关联围栏：</strong>{alarm.geofence_name}</p>}
          {alarm.lat && alarm.lon && (
            <p><strong>位置：</strong>{alarm.lat.toFixed(6)}, {alarm.lon.toFixed(6)}</p>
          )}
          <p><strong>状态：</strong>{getStatusTag(alarm.status)}</p>
          <p><strong>创建时间：</strong>{new Date(alarm.created_at).toLocaleString()}</p>
          {alarm.resolved_at && (
            <>
              <p><strong>处理时间：</strong>{new Date(alarm.resolved_at).toLocaleString()}</p>
              {alarm.resolve_note && <p><strong>处理备注：</strong>{alarm.resolve_note}</p>}
            </>
          )}
        </div>
      ),
    });
  };

  // 获取报警类型标签
  const getAlarmTypeLabel = (type: AlarmType) => {
    const found = alarmTypes.find(t => t.value === type);
    return found?.label || type;
  };

  // 获取级别标签
  const getLevelTag = (level: AlarmLevel) => {
    const colors = {
      critical: 'red',
      warning: 'orange',
      info: 'blue',
    };
    const labels = {
      critical: '严重',
      warning: '警告',
      info: '信息',
    };
    return <Tag color={colors[level]}>{labels[level]}</Tag>;
  };

  // 获取状态标签
  const getStatusTag = (status: AlarmStatus) => {
    const colors = {
      unread: 'red',
      read: 'blue',
      resolved: 'green',
    };
    const labels = {
      unread: '未读',
      read: '已读',
      resolved: '已处理',
    };
    return <Badge status={status === 'unread' ? 'error' : status === 'read' ? 'processing' : 'success'} text={labels[status]} />;
  };

  const columns: ColumnsType<Alarm> = [
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: getStatusTag,
    },
    {
      title: '级别',
      dataIndex: 'level',
      width: 100,
      render: getLevelTag,
    },
    {
      title: '类型',
      dataIndex: 'type',
      width: 120,
      render: (type) => getAlarmTypeLabel(type),
    },
    {
      title: '设备',
      dataIndex: 'device_name',
      width: 150,
      render: (name, record) => name || record.device_id,
    },
    {
      title: '报警内容',
      dataIndex: 'content',
      ellipsis: true,
      render: (content, record) => (
        <Tooltip title={content}>
          <span>{record.title}: {content}</span>
        </Tooltip>
      ),
    },
    {
      title: '时间',
      dataIndex: 'created_at',
      width: 180,
      render: (time) => new Date(time).toLocaleString(),
    },
    {
      title: '操作',
      key: 'action',
      width: 150,
      render: (_, record) => (
        <Space size="small">
          <Button
            type="text"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => handleViewDetail(record)}
          />
          {record.status === 'unread' && (
            <Button
              type="text"
              size="small"
              icon={<CheckOutlined />}
              onClick={() => handleMarkAsRead(record.id)}
              title="标记已读"
            />
          )}
          {record.status !== 'resolved' && (
            <Button
              type="text"
              size="small"
              icon={<ExclamationCircleOutlined />}
              onClick={() => handleResolve(record.id)}
              title="处理"
            />
          )}
        </Space>
      ),
    },
  ];

  const rowSelection = {
    selectedRowKeys,
    onChange: (newSelectedRowKeys: React.Key[]) => {
      setSelectedRowKeys(newSelectedRowKeys);
    },
  };

  return (
    <div className={styles.alarmList}>
      <div className={styles.filterBar}>
        <Space wrap>
          <Select
            placeholder="报警类型"
            allowClear
            style={{ width: 140 }}
            value={filters.type}
            onChange={(value) => setFilters(prev => ({ ...prev, type: value }))}
          >
            {alarmTypes.map(type => (
              <Option key={type.value} value={type.value}>{type.label}</Option>
            ))}
          </Select>
          <Select
            placeholder="报警级别"
            allowClear
            style={{ width: 120 }}
            value={filters.level}
            onChange={(value) => setFilters(prev => ({ ...prev, level: value }))}
          >
            <Option value="critical">严重</Option>
            <Option value="warning">警告</Option>
            <Option value="info">信息</Option>
          </Select>
          <Select
            placeholder="处理状态"
            allowClear
            style={{ width: 120 }}
            value={filters.status}
            onChange={(value) => setFilters(prev => ({ ...prev, status: value }))}
          >
            <Option value="unread">未读</Option>
            <Option value="read">已读</Option>
            <Option value="resolved">已处理</Option>
          </Select>
          <RangePicker
            showTime
            value={filters.dateRange}
            onChange={(dates) => setFilters(prev => ({ ...prev, dateRange: dates as any }))}
          />
          <Button icon={<ReloadOutlined />} onClick={fetchAlarms}>
            刷新
          </Button>
        </Space>
        
        {selectedRowKeys.length > 0 && (
          <Space style={{ marginLeft: 16 }}>
            <span>已选择 {selectedRowKeys.length} 项</span>
            <Button onClick={handleBatchRead}>批量已读</Button>
            <Button type="primary" onClick={handleBatchResolve}>批量处理</Button>
          </Space>
        )}
      </div>

      <Table
        rowSelection={rowSelection}
        columns={columns}
        dataSource={alarms}
        rowKey="id"
        loading={loading}
        pagination={{
          ...pagination,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (total) => `共 ${total} 条`,
        }}
        onChange={(newPagination) => {
          setPagination({
            current: newPagination.current || 1,
            pageSize: newPagination.pageSize || 20,
            total: pagination.total,
          });
        }}
      />
    </div>
  );
};

export default AlarmList;
