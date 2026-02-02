import axios from 'axios'
import { useAuthStore } from '../stores/auth'

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || '/api/v1',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor
api.interceptors.request.use(
  (config) => {
    const token = useAuthStore.getState().token
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Response interceptor
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      useAuthStore.getState().logout()
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

export default api

// Auth API
export const authApi = {
  login: (username, password) =>
    api.post('/auth/login', { username, password }),
  getMe: () =>
    api.get('/auth/me'),
}

// Device API
export const deviceApi = {
  getDevices: (params) =>
    api.get('/devices', { params }),
  getDevice: (id) =>
    api.get(`/devices/${id}`),
  createDevice: (data) =>
    api.post('/devices', data),
  updateDevice: (id, data) =>
    api.put(`/devices/${id}`, data),
  deleteDevice: (id) =>
    api.delete(`/devices/${id}`),
  getDeviceShadow: (deviceId) =>
    api.get(`/devices/${deviceId}/shadow`),
  sendCommand: (deviceId, type, params) =>
    api.post(`/devices/${deviceId}/commands`, { type, params }),
}

// Position API
export const positionApi = {
  getLatestPositions: () =>
    api.get('/positions/latest'),
  getDeviceHistory: (deviceId, start, end, limit = 1000) =>
    api.get(`/devices/${deviceId}/positions`, {
      params: { start, end, limit },
    }),
  getDeviceLatest: (deviceId) =>
    api.get(`/devices/${deviceId}/positions/latest`),
}
