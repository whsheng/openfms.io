import React, { useState } from 'react';
import {
  Card,
  Table,
  DatePicker,
  Select,
  Button,
  Space,
  Tag,
} from 'antd';
import { PlayCircleOutlined, DownloadOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import type { VideoRecord } from '../../types/video';

const { RangePicker } = DatePicker;
const { Option } = Select;

const VideoRecords: React.FC = () => {
  const [records, setRecords] = useState<VideoRecord[]>([]);
  const [loading, setLoading] = useState(false);

  const columns: ColumnsType<VideoRecord> = [
    {
      title: '设备',
      dataIndex: 'device_id',
    },
    {
      title: '通道',
      dataIndex: 'channel',
      width: 80,
    },
    {
      title: '开始时间',
      dataIndex: 'start_time',
      render: (time) => new Date(time).toLocaleString(),
    },
    {
      title: '结束时间',
      dataIndex: 'end_time',
      render: (time) => new Date(time).toLocaleString(),
    },
    {
      title: '时长',
      dataIndex: 'duration',
      render: (duration) => {
        const hours = Math.floor(duration / 3600);
        const minutes = Math.floor((duration % 3600) / 60);
        const seconds = duration % 60;
        return `${hours}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
      },
    },
    {
      title: '大小',
      dataIndex: 'file_size',
      render: (size) => {
        if (size < 1024) return `${size} B`;
        if (size < 1024 * 1024) return `${(size / 1024).toFixed(2)} KB`;
        if (size < 1024 * 1024 * 1024) return `${(size / 1024 / 1024).toFixed(2)} MB`;
        return `${(size / 1024 / 1024 / 1024).toFixed(2)} GB`;
      },
    },
    {
      title: '类型',
      dataIndex: 'record_type',
      render: (type) => {
        const colors: Record<string, string> = {
          auto: 'blue',
          alarm: 'red',
          manual: 'green',
        };
        const labels: Record<string, string> = {
          auto: '自动',
          alarm: '报警',
          manual: '手动',
        };
        return <Tag color={colors[type]}>{labels[type]}</Tag>;
      },
    },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Space>
          <Button
            type="primary"
            size="small"
            icon={<PlayCircleOutlined />}
          >
            播放
          </Button>
          <Button
            size="small"
            icon={<DownloadOutlined />}
          >
            下载
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 16 }}>
      <Card title="录像查询">
        <Space style={{ marginBottom: 16 }}>
          <Select style={{ width: 200 }} placeholder="选择设备">
            <Option value="">全部设备</Option>
          </Select>
          <Select style={{ width: 120 }} placeholder="通道" defaultValue={1}>
            {[1, 2, 3, 4, 5, 6, 7, 8].map((ch) => (
              <Option key={ch} value={ch}>通道 {ch}</Option>
            ))}
          </Select>
          <RangePicker showTime />
          <Button type="primary">查询</Button>
        </Space>

        <Table
          columns={columns}
          dataSource={records}
          rowKey="id"
          loading={loading}
        />
      </Card>
    </div>
  );
};

export default VideoRecords;
