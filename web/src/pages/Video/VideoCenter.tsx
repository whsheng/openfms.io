import React, { useState, useEffect } from 'react';
import {
  Card,
  Row,
  Col,
  Select,
  Button,
  message,
  List,
  Tag,
  Badge,
} from 'antd';
import {
  VideoCameraOutlined,
  PlayCircleOutlined,
  StopOutlined,
  HistoryOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import { videoApi } from '../../services/api';
import VideoPlayer from './VideoPlayer';
import type { VideoStream } from '../../types/video';

const { Option } = Select;

interface VideoCenterProps {
  deviceId?: string;
}

const VideoCenter: React.FC<VideoCenterProps> = ({ deviceId: propDeviceId }) => {
  const [deviceId, setDeviceId] = useState<string>(propDeviceId || '');
  const [channel, setChannel] = useState<number>(1);
  const [activeStreams, setActiveStreams] = useState<VideoStream[]>([]);
  const [loading, setLoading] = useState(false);
  const [currentStream, setCurrentStream] = useState<VideoStream | null>(null);

  // 获取活跃流
  const fetchActiveStreams = async () => {
    try {
      const res = await videoApi.getActiveStreams(deviceId);
      setActiveStreams(res.data);
    } catch (error) {
      console.error('获取活跃流失败:', error);
    }
  };

  useEffect(() => {
    fetchActiveStreams();
    const interval = setInterval(fetchActiveStreams, 5000);
    return () => clearInterval(interval);
  }, [deviceId]);

  // 开始实时视频
  const startRealtime = async () => {
    if (!deviceId) {
      message.warning('请选择设备');
      return;
    }
    setLoading(true);
    try {
      const res = await videoApi.startRealtime(deviceId, channel);
      setCurrentStream(res.data);
      message.success('视频流已启动');
      fetchActiveStreams();
    } catch (error: any) {
      message.error(error.response?.data?.error || '启动失败');
    } finally {
      setLoading(false);
    }
  };

  // 停止视频
  const stopVideo = async (streamId: number) => {
    try {
      await videoApi.stopVideo(streamId);
      message.success('视频已停止');
      if (currentStream?.stream_id === streamId) {
        setCurrentStream(null);
      }
      fetchActiveStreams();
    } catch (error) {
      message.error('停止失败');
    }
  };

  return (
    <div style={{ padding: 16 }}>
      <Row gutter={[16, 16]}>
        <Col span={16}>
          <Card
            title={
              <span>
                <VideoCameraOutlined /> 视频播放
              </span>
            }
          >
            {currentStream ? (
              <VideoPlayer
                stream={currentStream}
                onStop={() => stopVideo(currentStream.stream_id)}
              />
            ) : (
              <div style={{ textAlign: 'center', padding: '100px 0', color: '#999' }}>
                <VideoCameraOutlined style={{ fontSize: 64, marginBottom: 16 }} />
                <p>请选择设备并点击"开始播放"</p>
              </div>
            )}
          </Card>
        </Col>
        <Col span={8}>
          <Card
            title="控制面板"
            extra={
              <Badge count={activeStreams.length} showZero>
                <span style={{ marginRight: 8 }}>活跃流</span>
              </Badge>
            }
          >
            <div style={{ marginBottom: 16 }}>
              <div style={{ marginBottom: 8 }}>设备</div>
              <Select
                style={{ width: '100%' }}
                placeholder="选择设备"
                value={deviceId || undefined}
                onChange={setDeviceId}
              >
                <Option value="13912345678">设备 13912345678</Option>
              </Select>
            </div>

            <div style={{ marginBottom: 16 }}>
              <div style={{ marginBottom: 8 }}>通道</div>
              <Select
                style={{ width: '100%' }}
                value={channel}
                onChange={setChannel}
              >
                {[1, 2, 3, 4, 5, 6, 7, 8].map((ch) => (
                  <Option key={ch} value={ch}>通道 {ch}</Option>
                ))}
              </Select>
            </div>

            <Button
              type="primary"
              icon={<PlayCircleOutlined />}
              block
              loading={loading}
              onClick={startRealtime}
              style={{ marginBottom: 8 }}
            >
              开始播放
            </Button>

            <Button
              icon={<HistoryOutlined />}
              block
              style={{ marginBottom: 8 }}
            >
              录像回放
            </Button>

            <Button
              icon={<SettingOutlined />}
              block
            >
              设备配置
            </Button>
          </Card>

          <Card title="活跃流列表" style={{ marginTop: 16 }}>
            <List
              size="small"
              dataSource={activeStreams}
              renderItem={(stream) => (
                <List.Item
                  actions={[
                    <Button
                      type="text"
                      danger
                      size="small"
                      icon={<StopOutlined />}
                      onClick={() => stopVideo(stream.stream_id)}
                    >
                      停止
                    </Button>,
                  ]}
                >
                  <List.Item.Meta
                    title={
                      <span>
                        {stream.device_id} - 通道{stream.channel}
                        <Tag color="green" style={{ marginLeft: 8 }}>播放中</Tag>
                      </span>
                    }
                    description={new Date().toLocaleTimeString()}
                  />
                </List.Item>
              )}
            />
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default VideoCenter;
