import React, { useState, useEffect } from 'react';
import {
  Card,
  Row,
  Col,
  Statistic,
  DatePicker,
  Select,
  Button,
  Table,
  Tabs,
} from 'antd';
import {
  CarOutlined,
  AlertOutlined,
  RoadOutlined,
} from '@ant-design/icons';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  BarChart,
  Bar,
  PieChart,
  Pie,
  Cell,
  ResponsiveContainer,
} from 'recharts';
import { reportApi } from '../../services/api';

const { RangePicker } = DatePicker;
const { Option } = Select;
const { TabPane } = Tabs;

const COLORS = ['#0088FE', '#00C49F', '#FFBB28', '#FF8042', '#8884D8'];

const ReportDashboard: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [mileageData, setMileageData] = useState<any[]>([]);
  const [stats, setStats] = useState<any>({});

  useEffect(() => {
    fetchStats();
    // 模拟数据
    setMileageData([
      { date: '2026-01-01', mileage: 120.5 },
      { date: '2026-01-02', mileage: 98.3 },
      { date: '2026-01-03', mileage: 156.7 },
      { date: '2026-01-04', mileage: 203.4 },
      { date: '2026-01-05', mileage: 178.9 },
    ]);
  }, []);

  const fetchStats = async () => {
    try {
      const res = await reportApi.getDashboardStats();
      setStats(res.data);
    } catch (error) {
      console.error('获取统计失败:', error);
    }
  };

  const alarmTypeData = [
    { name: '超速', value: 12 },
    { name: '围栏', value: 8 },
    { name: '离线', value: 5 },
    { name: '低电量', value: 3 },
  ];

  return (
    <div style={{ padding: 16 }}>
      <Row gutter={[16, 16]}>
        <Col span={6}>
          <Card>
            <Statistic
              title="今日里程"
              value={stats.today?.mileage || 0}
              suffix="km"
              prefix={<RoadOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="在线设备"
              value={stats.online_devices || 0}
              suffix={`/ ${stats.total_devices || 0}`}
              prefix={<CarOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="今日报警"
              value={stats.today?.alarms || 0}
              prefix={<AlertOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="未处理报警"
              value={stats.unread_alarms || 0}
              valueStyle={{ color: '#cf1322' }}
              prefix={<AlertOutlined />}
            />
          </Card>
        </Col>
      </Row>

      <Card style={{ marginTop: 16 }}>
        <Tabs defaultActiveKey="mileage">
          <TabPane tab="里程报表" key="mileage">
            <Row gutter={[16, 16]}>
              <Col span={24}>
                <Space style={{ marginBottom: 16 }}>
                  <Select style={{ width: 200 }} placeholder="选择设备">
                    <Option value="">全部设备</Option>
                  </Select>
                  <RangePicker />
                  <Button type="primary">查询</Button>
                  <Button>导出Excel</Button>
                </Space>
              </Col>
              <Col span={16}>
                <ResponsiveContainer width="100%" height={300}>
                  <LineChart data={mileageData}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="date" />
                    <YAxis />
                    <Tooltip />
                    <Legend />
                    <Line type="monotone" dataKey="mileage" name="里程(km)" stroke="#8884d8" />
                  </LineChart>
                </ResponsiveContainer>
              </Col>
              <Col span={8}>
                <Table
                  size="small"
                  dataSource={mileageData}
                  columns={[
                    { title: '日期', dataIndex: 'date' },
                    { title: '里程(km)', dataIndex: 'mileage' },
                  ]}
                  pagination={false}
                />
              </Col>
            </Row>
          </TabPane>
          
          <TabPane tab="报警统计" key="alarm">
            <Row gutter={[16, 16]}>
              <Col span={12}>
                <ResponsiveContainer width="100%" height={300}>
                  <PieChart>
                    <Pie
                      data={alarmTypeData}
                      cx="50%"
                      cy="50%"
                      labelLine={false}
                      label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
                      outerRadius={80}
                      dataKey="value"
                    >
                      {alarmTypeData.map((entry, index) => (
                        <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                      ))}
                    </Pie>
                    <Tooltip />
                  </PieChart>
                </ResponsiveContainer>
              </Col>
              <Col span={12}>
                <ResponsiveContainer width="100%" height={300}>
                  <BarChart data={alarmTypeData}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="name" />
                    <YAxis />
                    <Tooltip />
                    <Bar dataKey="value" fill="#8884d8">
                      {alarmTypeData.map((entry, index) => (
                        <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                      ))}
                    </Bar>
                  </BarChart>
                </ResponsiveContainer>
              </Col>
            </Row>
          </TabPane>
        </Tabs>
      </Card>
    </div>
  );
};

// eslint-disable-next-line @typescript-eslint/no-unused-vars
const Space = ({ children, style }: { children: React.ReactNode; style?: React.CSSProperties }) => (
  <div style={{ display: 'flex', gap: 8, ...style }}>{children}</div>
);

export default ReportDashboard;
