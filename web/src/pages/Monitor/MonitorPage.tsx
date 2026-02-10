import React, { useEffect, useState, useRef, useCallback, useMemo } from 'react';
import {
  Card,
  List,
  Badge,
  Space,
  Typography,
  Button,
  Tooltip,
  Segmented,
  Empty,
  Spin,
  Tag,
} from 'antd';
import {
  CarOutlined,
  ReloadOutlined,
  AimOutlined,
  GlobalOutlined,
  DashboardOutlined,
  EnvironmentOutlined,
} from '@ant-design/icons';
import { MapboxMap, MapboxMapRef, VehicleData } from '../../components/MapboxMap';
import { positionApi } from '../../services/api';
import styles from './monitor.module.css';

const { Text } = Typography;

// 车辆状态类型
interface VehicleStatus {
  key: 'all' | 'moving' | 'idle' | 'offline';
  label: string;
  color: string;
}

const STATUS_FILTERS: VehicleStatus[] = [
  { key: 'all', label: '全部', color: '#1890ff' },
  { key: 'moving', label: '行驶中', color: '#1890ff' },
  { key: 'idle', label: '静止', color: '#faad14' },
  { key: 'offline', label: '离线', color: '#bfbfbf' },
];

// 状态映射
const getVehicleStatus = (speed: number, lastTime: string): VehicleData['status'] => {
  const lastUpdate = new Date(lastTime).getTime();
  const now = Date.now();
  const diffMinutes = (now - lastUpdate) / 1000 / 60;

  if (diffMinutes > 30) return 'offline';
  if (speed > 0) return 'moving';
  return 'idle';
};

// 转换 API 数据为 VehicleData
const transformPositionToVehicle = (position: any): VehicleData => ({
  deviceId: position.device_id,
  lat: position.lat,
  lon: position.lon,
  speed: position.speed || 0,
  heading: position.heading || 0,
  status: getVehicleStatus(position.speed || 0, position.created_at),
  lastReportTime: position.created_at,
  plateNumber: position.plate_number || position.device_id,
  driverName: position.driver_name,
});

const MonitorPage: React.FC = () => {
  const mapRef = useRef<MapboxMapRef>(null);
  const [vehicles, setVehicles] = useState<VehicleData[]>([]);
  const [selectedVehicleId, setSelectedVehicleId] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [statusFilter, setStatusFilter] = useState<VehicleStatus['key']>('all');
  const [mapType, setMapType] = useState<'vector' | 'satellite'>('vector');
  const [showTraffic, setShowTraffic] = useState(false);

  // 获取车辆数据
  const fetchVehicles = useCallback(async () => {
    setLoading(true);
    try {
      const res = await positionApi.getLatestPositions();
      const positions = res.data?.data || [];
      const vehicleData = positions.map(transformPositionToVehicle);
      setVehicles(vehicleData);
    } catch (error) {
      console.error('Failed to fetch vehicles:', error);
    } finally {
      setLoading(false);
    }
  }, []);

  // 初始加载和定时刷新
  useEffect(() => {
    fetchVehicles();
    const interval = setInterval(fetchVehicles, 10000); // 每10秒刷新
    return () => clearInterval(interval);
  }, [fetchVehicles]);

  // 过滤后的车辆列表
  const filteredVehicles = useMemo(() => {
    if (statusFilter === 'all') return vehicles;
    return vehicles.filter(v => v.status === statusFilter);
  }, [vehicles, statusFilter]);

  // 选中的车辆
  const selectedVehicle = useMemo(() => {
    return vehicles.find(v => v.deviceId === selectedVehicleId) || null;
  }, [vehicles, selectedVehicleId]);

  // 统计信息
  const stats = useMemo(() => {
    const total = vehicles.length;
    const moving = vehicles.filter(v => v.status === 'moving').length;
    const idle = vehicles.filter(v => v.status === 'idle').length;
    const offline = vehicles.filter(v => v.status === 'offline').length;
    return { total, moving, idle, offline };
  }, [vehicles]);

  // 处理车辆点击
  const handleVehicleClick = useCallback((vehicle: VehicleData) => {
    setSelectedVehicleId(vehicle.deviceId);
    // 地图飞移到选中车辆
    mapRef.current?.flyToVehicle(vehicle, 16);
  }, []);

  // 处理列表项点击
  const handleListItemClick = useCallback((vehicle: VehicleData) => {
    setSelectedVehicleId(vehicle.deviceId);
    mapRef.current?.flyToVehicle(vehicle, 16);
  }, []);

  // 定位到所有车辆
  const handleFitAll = useCallback(() => {
    if (vehicles.length === 0) return;
    
    const bounds = vehicles.reduce(
      (acc, v) => ({
        minLon: Math.min(acc.minLon, v.lon),
        maxLon: Math.max(acc.maxLon, v.lon),
        minLat: Math.min(acc.minLat, v.lat),
        maxLat: Math.max(acc.maxLat, v.lat),
      }),
      {
        minLon: vehicles[0].lon,
        maxLon: vehicles[0].lon,
        minLat: vehicles[0].lat,
        maxLat: vehicles[0].lat,
      }
    );

    mapRef.current?.fitBounds(
      [
        [bounds.minLon, bounds.minLat],
        [bounds.maxLon, bounds.maxLat],
      ],
      { padding: 100 }
    );
  }, [vehicles]);

  // 获取状态样式
  const getStatusIconClass = (status: VehicleData['status']) => {
    switch (status) {
      case 'moving':
        return styles.deviceIconMoving;
      case 'idle':
        return styles.deviceIconIdle;
      case 'offline':
        return styles.deviceIconOffline;
      default:
        return styles.deviceIconOnline;
    }
  };

  // 获取状态颜色
  const getStatusColor = (status: VehicleData['status']) => {
    switch (status) {
      case 'moving':
        return '#1890ff';
      case 'idle':
        return '#faad14';
      case 'offline':
        return '#bfbfbf';
      default:
        return '#52c41a';
    }
  };

  // 获取状态文本
  const getStatusText = (status: VehicleData['status']) => {
    switch (status) {
      case 'moving':
        return '行驶中';
      case 'idle':
        return '静止';
      case 'offline':
        return '离线';
      default:
        return '在线';
    }
  };

  return (
    <div className={styles.monitorPage}>
      {/* 侧边栏 */}
      <div className={styles.sidebar}>
        {/* 统计卡片 */}
        <Card className={styles.statCard} size="small">
          <div className={styles.statRow}>
            <div className={styles.statItem}>
              <div className={styles.statValue} style={{ color: '#1890ff' }}>
                {stats.total}
              </div>
              <div className={styles.statLabel}>总车辆</div>
            </div>
            <div className={styles.statItem}>
              <div className={styles.statValue} style={{ color: '#52c41a' }}>
                {stats.moving}
              </div>
              <div className={styles.statLabel}>行驶中</div>
            </div>
            <div className={styles.statItem}>
              <div className={styles.statValue} style={{ color: '#faad14' }}>
                {stats.idle}
              </div>
              <div className={styles.statLabel}>静止</div>
            </div>
            <div className={styles.statItem}>
              <div className={styles.statValue} style={{ color: '#bfbfbf' }}>
                {stats.offline}
              </div>
              <div className={styles.statLabel}>离线</div>
            </div>
          </div>
        </Card>

        {/* 设备列表 */}
        <Card
          className={styles.deviceList}
          title={
            <div className={styles.listHeader}>
              <span className={styles.listTitle}>车辆列表</span>
              <Space>
                <Tooltip title="刷新">
                  <Button
                    type="text"
                    icon={<ReloadOutlined spin={loading} />}
                    onClick={fetchVehicles}
                    size="small"
                  />
                </Tooltip>
              </Space>
            </div>
          }
          extra={
            <Segmented
              size="small"
              options={STATUS_FILTERS.map(s => ({
                label: s.label,
                value: s.key,
              }))}
              value={statusFilter}
              onChange={setStatusFilter}
            />
          }
          bodyStyle={{ padding: 0, height: 'calc(100% - 57px)', overflow: 'auto' }}
        >
          {loading && vehicles.length === 0 ? (
            <div style={{ padding: 40, textAlign: 'center' }}>
              <Spin />
            </div>
          ) : filteredVehicles.length === 0 ? (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description="暂无车辆数据"
              style={{ marginTop: 40 }}
            />
          ) : (
            <List
              dataSource={filteredVehicles}
              renderItem={(item) => (
                <List.Item
                  className={selectedVehicleId === item.deviceId ? 'selected' : ''}
                  onClick={() => handleListItemClick(item)}
                  style={{
                    backgroundColor:
                      selectedVehicleId === item.deviceId ? '#e6f7ff' : 'transparent',
                  }}
                >
                  <div className={styles.deviceItem}>
                    <div
                      className={`${styles.deviceIcon} ${getStatusIconClass(
                        item.status
                      )}`}
                    >
                      <CarOutlined style={{ fontSize: 20 }} />
                    </div>
                    <div className={styles.deviceInfo}>
                      <div className={styles.deviceName}>
                        {item.plateNumber || item.deviceId}
                      </div>
                      <div className={styles.deviceMeta}>
                        <Space size={8}>
                          <span>{item.speed} km/h</span>
                          <span>·</span>
                          <span>
                            {new Date(item.lastReportTime).toLocaleTimeString('zh-CN', {
                              hour: '2-digit',
                              minute: '2-digit',
                            })}
                          </span>
                        </Space>
                      </div>
                    </div>
                    <div className={styles.deviceStatus}>
                      <Badge
                        color={getStatusColor(item.status)}
                        text={
                          <span style={{ fontSize: 12 }}>
                            {getStatusText(item.status)}
                          </span>
                        }
                      />
                    </div>
                  </div>
                </List.Item>
              )}
            />
          )}
        </Card>
      </div>

      {/* 地图容器 */}
      <div className={styles.mapContainer}>
        {/* 地图工具栏 */}
        <div className={styles.mapToolbar}>
          <Tooltip title="刷新数据">
            <Button
              icon={<ReloadOutlined spin={loading} />}
              onClick={fetchVehicles}
              size="small"
            />
          </Tooltip>
          <Tooltip title="适应所有车辆">
            <Button icon={<AimOutlined />} onClick={handleFitAll} size="small" />
          </Tooltip>
          <Tooltip title={mapType === 'satellite' ? '切换矢量图' : '切换卫星图'}>
            <Button
              icon={<GlobalOutlined />}
              type={mapType === 'satellite' ? 'primary' : 'default'}
              onClick={() =>
                setMapType(prev => (prev === 'vector' ? 'satellite' : 'vector'))
              }
              size="small"
            >
              {mapType === 'satellite' ? '卫星' : '矢量'}
            </Button>
          </Tooltip>
          <Tooltip title="显示交通">
            <Button
              icon={<DashboardOutlined />}
              type={showTraffic ? 'primary' : 'default'}
              onClick={() => setShowTraffic(prev => !prev)}
              size="small"
            >
              交通
            </Button>
          </Tooltip>
        </div>

        {/* 地图图例 */}
        <div className={styles.mapLegend}>
          <div className={styles.legendTitle}>车辆状态</div>
          <div className={styles.legendItem}>
            <span
              className={styles.legendDot}
              style={{ backgroundColor: '#1890ff' }}
            />
            <span>行驶中</span>
          </div>
          <div className={styles.legendItem}>
            <span
              className={styles.legendDot}
              style={{ backgroundColor: '#faad14' }}
            />
            <span>静止</span>
          </div>
          <div className={styles.legendItem}>
            <span
              className={styles.legendDot}
              style={{ backgroundColor: '#bfbfbf' }}
            />
            <span>离线</span>
          </div>
        </div>

        {/* Mapbox 地图 */}
        <MapboxMap
          ref={mapRef}
          vehicles={vehicles}
          selectedVehicleId={selectedVehicleId}
          onVehicleClick={handleVehicleClick}
          showTraffic={showTraffic}
          showSatellite={mapType === 'satellite'}
          style={{ width: '100%', height: '100%' }}
        />
      </div>
    </div>
  );
};

export default MonitorPage;
