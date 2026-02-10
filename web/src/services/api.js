import axios from 'axios';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:3000/api/v1';

// 创建 axios 实例
const apiClient = axios.create({
  baseURL: API_BASE_URL,
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// 请求拦截器 - 添加 token
apiClient.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// 响应拦截器 - 处理错误
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

// 认证相关
export const authApi = {
  login: (data) => apiClient.post('/auth/login', data),
  register: (data) => apiClient.post('/auth/register', data),
  logout: () => apiClient.post('/auth/logout'),
  getProfile: () => apiClient.get('/auth/profile'),
};

// 设备相关
export const deviceApi = {
  getList: (params) => apiClient.get('/devices', { params }),
  getDetail: (id) => apiClient.get(`/devices/${id}`),
  create: (data) => apiClient.post('/devices', data),
  update: (id, data) => apiClient.put(`/devices/${id}`, data),
  delete: (id) => apiClient.delete(`/devices/${id}`),
  getLatestPositions: () => apiClient.get('/devices/positions/latest'),
  getHistory: (id, params) => apiClient.get(`/devices/${id}/history`, { params }),
  sendCommand: (id, data) => apiClient.post(`/devices/${id}/commands`, data),
  // 设备导入
  downloadImportTemplate: () => apiClient.get('/devices/import-template', {
    responseType: 'blob',
  }),
  previewImport: (formData) => apiClient.post('/devices/import-preview', formData, {
    headers: {
      'Content-Type': 'multipart/form-data',
    },
  }),
  importDevices: (formData) => apiClient.post('/devices/import', formData, {
    headers: {
      'Content-Type': 'multipart/form-data',
    },
  }),
  getImportStatus: (taskId) => apiClient.get(`/devices/import/${taskId}/status`),
  downloadImportErrors: (taskId) => apiClient.get(`/devices/import/${taskId}/errors`, {
    responseType: 'blob',
  }),
};

// 位置相关
export const positionApi = {
  getLatest: (deviceId) => apiClient.get(`/positions/${deviceId}/latest`),
  getHistory: (deviceId, params) => apiClient.get(`/positions/${deviceId}/history`, { params }),
};

// 围栏相关
export const geofenceApi = {
  getList: () => apiClient.get('/geofences'),
  getDetail: (id) => apiClient.get(`/geofences/${id}`),
  create: (data) => apiClient.post('/geofences', data),
  update: (id, data) => apiClient.put(`/geofences/${id}`, data),
  delete: (id) => apiClient.delete(`/geofences/${id}`),
  bindDevices: (id, deviceIds) => apiClient.post(`/geofences/${id}/bind`, { device_ids: deviceIds }),
  unbindDevices: (id, deviceIds) => apiClient.post(`/geofences/${id}/unbind`, { device_ids: deviceIds }),
  getBoundDevices: (id) => apiClient.get(`/geofences/${id}/devices`),
  getEvents: (id, params) => apiClient.get(`/geofences/${id}/events`, { params }),
};

// 报警相关
export const alarmApi = {
  getList: (params) => apiClient.get('/alarms', { params }),
  getDetail: (id) => apiClient.get(`/alarms/${id}`),
  getStats: () => apiClient.get('/alarms/stats'),
  getTypes: () => apiClient.get('/alarms/types'),
  markAsRead: (id) => apiClient.post(`/alarms/${id}/read`),
  resolve: (id, note) => apiClient.post(`/alarms/${id}/resolve`, { resolve_note: note }),
  batchRead: (ids) => apiClient.post('/alarms/batch-read', { ids }),
  batchResolve: (ids, note) => apiClient.post('/alarms/batch-resolve', { ids, resolve_note: note }),
  getRules: () => apiClient.get('/alarms/rules'),
  getRule: (id) => apiClient.get(`/alarms/rules/${id}`),
  updateRule: (id, data) => apiClient.put(`/alarms/rules/${id}`, data),
  toggleRule: (id) => apiClient.post(`/alarms/rules/${id}/toggle`),
};

// 视频相关
export const videoApi = {
  startRealtime: (deviceId, channel) => apiClient.post('/videos/start', { device_id: deviceId, channel }),
  stopVideo: (streamId) => apiClient.post('/videos/stop', { stream_id: streamId }),
  getActiveStreams: (deviceId) => apiClient.get('/videos/streams', { params: { device_id: deviceId } }),
  getStreamStatus: (id) => apiClient.get(`/videos/streams/${id}`),
  startPlayback: (data) => apiClient.post('/videos/playback/start', data),
  controlPlayback: (data) => apiClient.post('/videos/playback/control', data),
  queryRecords: (params) => apiClient.get('/videos/records', { params }),
  takeSnapshot: (deviceId, channel) => apiClient.post('/videos/snapshot', { device_id: deviceId, channel }),
  getDeviceConfig: (deviceId) => apiClient.get(`/videos/devices/${deviceId}/config`),
  updateDeviceConfig: (deviceId, data) => apiClient.put(`/videos/devices/${deviceId}/config`, data),
};

// 用户和权限相关
export const userApi = {
  // 用户管理
  getList: () => apiClient.get('/users'),
  getDetail: (id) => apiClient.get(`/users/${id}`),
  create: (data) => apiClient.post('/users', data),
  update: (id, data) => apiClient.put(`/users/${id}`, data),
  delete: (id) => apiClient.delete(`/users/${id}`),
  assignRole: (id, roleId) => apiClient.post(`/users/${id}/role`, { role_id: roleId }),
  
  // 角色管理
  getRoles: () => apiClient.get('/roles'),
  getRole: (id) => apiClient.get(`/roles/${id}`),
  createRole: (data) => apiClient.post('/roles', data),
  updateRole: (id, data) => apiClient.put(`/roles/${id}`, data),
  deleteRole: (id) => apiClient.delete(`/roles/${id}`),
  getRolePermissions: (id) => apiClient.get(`/roles/${id}/permissions`),
  
  // 权限管理
  getPermissions: () => apiClient.get('/permissions'),
  getPermissionGroups: () => apiClient.get('/permissions/groups'),
  
  // 当前用户
  getMyPermissions: () => apiClient.get('/me/permissions'),
  checkPermission: (code) => apiClient.post('/me/check-permission', { permission_code: code }),
};

// 报表相关
export const reportApi = {
  getDashboardStats: () => apiClient.get('/reports/dashboard'),
  getMileageReport: (params) => apiClient.get('/reports/mileage', { params }),
  getStopReport: (params) => apiClient.get('/reports/stops', { params }),
  getDrivingBehavior: (params) => apiClient.get('/reports/driving', { params }),
  generateReport: (data) => apiClient.post('/reports/generate', data),
};

export default apiClient;
