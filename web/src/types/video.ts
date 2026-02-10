// 视频流
type VideoStreamStatus = 'pending' | 'streaming' | 'stopped' | 'error';
type VideoStreamType = 'realtime' | 'playback';

export interface VideoStream {
  stream_id: number;
  device_id: string;
  channel: number;
  status: VideoStreamStatus;
  stream_url?: string;
  ws_flv_url?: string;
  webrtc_url?: string;
  hls_url?: string;
}

// 录像记录
export interface VideoRecord {
  id: number;
  device_id: string;
  channel: number;
  start_time: string;
  end_time: string;
  duration: number;
  file_size: number;
  file_path: string;
  record_type: 'auto' | 'alarm' | 'manual';
}

// 设备视频配置
export interface VideoDeviceConfig {
  device_id: string;
  channel_count: number;
  video_codec: string;
  audio_codec: string;
  resolution: string;
  frame_rate: number;
  bit_rate: number;
}
