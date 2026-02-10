import React from 'react'
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom'
import { useAuthStore } from './stores/auth'
import Login from './pages/Login'
import Layout from './components/Layout'
import Dashboard from './pages/Dashboard'
import Devices from './pages/Devices'
import RealtimeMap from './pages/RealtimeMap'
import MonitorPage from './pages/Monitor'
import History from './pages/History'
import Settings from './pages/Settings'
import { GeofenceList } from './pages/Geofence'
import AlarmCenter from './pages/Alarm/AlarmCenter'
import { UserList, RoleList } from './pages/Users'

function PrivateRoute({ children }) {
  const token = useAuthStore((state) => state.token)
  return token ? children : <Navigate to="/login" />
}

function App() {
  return (
    <Router>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route
          path="/"
          element={
            <PrivateRoute>
              <Layout />
            </PrivateRoute>
          }
        >
          <Route index element={<Dashboard />} />
          <Route path="map" element={<RealtimeMap />} />
          <Route path="monitor" element={<MonitorPage />} />
          <Route path="devices" element={<Devices />} />
          <Route path="geofences" element={<GeofenceList />} />
          <Route path="alarms" element={<AlarmCenter />} />
          <Route path="users" element={<UserList />} />
          <Route path="roles" element={<RoleList />} />
          <Route path="history" element={<History />} />
          <Route path="settings" element={<Settings />} />
        </Route>
      </Routes>
    </Router>
  )
}

export default App
