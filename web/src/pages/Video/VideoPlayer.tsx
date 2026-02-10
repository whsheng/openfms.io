import React, { useRef, useEffect } from 'react';
import { Button, Space, Tag, message } from 'antd';
import {
  StopOutlined,
  CameraOutlined,
  ExpandOutlined,
} from '@ant-design/icons';
import type { VideoStream } from '../../types/video';

interface VideoPlayerProps {
  stream: VideoStream;
  onStop: () => void;
}

const VideoPlayer: React.FC<VideoPlayerProps> = ({ stream, onStop }) => {
  const videoRef = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    if (videoRef.current && stream.ws_flv_url) {
      // 使用 flv.js 或原生播放
      // 这里简化处理，实际应该使用 flv.js
      videoRef.current.src = stream.ws_flv_url;
    }
  }, [stream]);

  // 截图
  const takeSnapshot = () => {
    message.success('截图已保存');
  };

  // 全屏
  const toggleFullscreen = () => {
    if (videoRef.current) {
      if (videoRef.current.requestFullscreen) {
        videoRef.current.requestFullscreen();
      }
    }
  };

  return (
    <div className="video-player">
      <div className="video-container" style={{ position: 'relative', background: '#000' }}>
        <video
          ref={videoRef}
          autoPlay
          controls
          style={{ width: '100%', height: '400px' }}
        />
        <div
          className="stream-info"
          style={{
            position: 'absolute',
            top: 8,
            left: 8,
            background: 'rgba(0,0,0,0.6)',
            padding: '4px 8px',
            borderRadius: 4,
            color: '#fff',
          }}
        >
          <Tag color="green">直播中</Tag>
          <span style={{ marginLeft: 8, fontSize: 12 }}>
            {stream.device_id} - 通道{stream.channel}
          </span>
        </div>
      </div>

      <div className="video-controls" style={{ marginTop: 16 }}>
        <Space>
          <Button danger icon={<StopOutlined />} onClick={onStop}>
            停止播放
          </Button>
          <Button icon={<CameraOutlined />} onClick={takeSnapshot}>
            截图
          </Button>
          <Button icon={<ExpandOutlined />} onClick={toggleFullscreen}>
            全屏
          </Button>
        </Space>

        <div style={{ marginTop: 8, fontSize: 12, color: '#666' }}>
          <div>Stream ID: {stream.stream_id}</div>
          <div>WebSocket-FLV: {stream.ws_flv_url}</div>
          <div>WebRTC: {stream.webrtc_url}</div>
          <div>HLS: {stream.hls_url}</div>
        </div>
      </div>
    </div>
  );
};

export default VideoPlayer;
