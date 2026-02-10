import React from 'react';
import { Card, Row, Col, Statistic, Progress } from 'antd';
import {
  PieChart,
  Pie,
  Cell,
  ResponsiveContainer,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
} from 'recharts';
import type { AlarmStats as AlarmStatsType } from '../../types/alarm';
import styles from './alarm.module.css';

interface AlarmStatsProps {
  stats: AlarmStatsType;
}

const COLORS = ['#ff4d4f', '#faad14', '#52c41a', '#1890ff'];

const AlarmStats: React.FC<AlarmStatsProps> = ({ stats }) => {
  // 状态分布数据
  const statusData = [
    { name: '未读', value: stats.unread },
    { name: '已读', value: stats.read },
    { name: '已处理', value: stats.resolved },
  ].filter(item => item.value > 0);

  // 级别分布数据
  const levelData = [
    { name: '严重', value: stats.critical },
    { name: '警告', value: stats.warning },
    { name: '信息', value: stats.info },
  ].filter(item => item.value > 0);

  // 处理率
  const resolveRate = stats.total > 0 
    ? Math.round((stats.resolved / stats.total) * 100) 
    : 0;

  return (
    <div className={styles.alarmStats}>
      <Row gutter={[16, 16]}>
        <Col span={8}>
          <Card title="处理率">
            <Progress
              type="circle"
              percent={resolveRate}
              strokeColor={{
                '0%': '#108ee9',
                '100%': '#87d068',
              }}
            />
            <div style={{ textAlign: 'center', marginTop: 16 }}>
              <Statistic
                title="已处理 / 总数"
                value={`${stats.resolved} / ${stats.total}`}
              />
            </div>
          </Card>
        </Col>
        <Col span={8}>
          <Card title="状态分布">
            <ResponsiveContainer width="100%" height={250}>
              <PieChart>
                <Pie
                  data={statusData}
                  cx="50%"
                  cy="50%"
                  innerRadius={60}
                  outerRadius={80}
                  paddingAngle={5}
                  dataKey="value"
                  label
                >
                  {statusData.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip />
                <Legend />
              </PieChart>
            </ResponsiveContainer>
          </Card>
        </Col>
        <Col span={8}>
          <Card title="级别分布">
            <ResponsiveContainer width="100%" height={250}>
              <BarChart data={levelData} layout="vertical">
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis type="number" />
                <YAxis dataKey="name" type="category" width={60} />
                <Tooltip />
                <Bar dataKey="value" fill="#8884d8">
                  {levelData.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default AlarmStats;
