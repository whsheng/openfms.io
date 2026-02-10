import React, { useState, useEffect } from 'react'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { Layout as AntLayout, Menu, Avatar, Dropdown, Space, Badge } from 'antd'
import {
  DashboardOutlined,
  CarOutlined,
  GlobalOutlined,
  HistoryOutlined,
  SettingOutlined,
  UserOutlined,
  LogoutOutlined,
  EnvironmentOutlined,
  BorderOutlined,
  BellOutlined,
  TeamOutlined,
  SafetyOutlined,
} from '@ant-design/icons'
import { useAuthStore } from '../stores/auth'
import { alarmApi } from '../services/alarm'

const { Header, Sider, Content } = AntLayout

function Layout() {
  const navigate = useNavigate()
  const location = useLocation()
  const { user, logout } = useAuthStore()
  const [unreadCount, setUnreadCount] = useState(0)

  // Fetch unread count periodically
  useEffect(() => {
    const fetchUnreadCount = async () => {
      try {
        const response = await alarmApi.getUnreadCount()
        setUnreadCount(response.data.unread_count || 0)
      } catch (error) {
        // Ignore errors
      }
    }

    fetchUnreadCount()
    const interval = setInterval(fetchUnreadCount, 30000) // Every 30 seconds

    return () => clearInterval(interval)
  }, [])

  const menuItems = [
    {
      key: '/',
      icon: <DashboardOutlined />,
      label: '仪表盘',
    },
    {
      key: '/map',
      icon: <GlobalOutlined />,
      label: '实时地图',
    },
    {
      key: '/monitor',
      icon: <EnvironmentOutlined />,
      label: '车辆监控',
    },
    {
      key: '/devices',
      icon: <CarOutlined />,
      label: '设备管理',
    },
    {
      key: '/geofences',
      icon: <BorderOutlined />,
      label: '电子围栏',
    },
    {
      key: '/alarms',
      icon: (
        <Badge count={unreadCount} size="small" offset={[5, -5]}>
          <BellOutlined />
        </Badge>
      ),
      label: '报警中心',
    },
    {
      key: 'user-management',
      icon: <TeamOutlined />,
      label: '用户管理',
      children: [
        {
          key: '/users',
          icon: <UserOutlined />,
          label: '用户列表',
        },
        {
          key: '/roles',
          icon: <SafetyOutlined />,
          label: '角色权限',
        },
      ],
    },
    {
      key: '/history',
      icon: <HistoryOutlined />,
      label: '历史轨迹',
    },
    {
      key: '/settings',
      icon: <SettingOutlined />,
      label: '系统设置',
    },
  ]

  const userMenuItems = [
    {
      key: 'profile',
      icon: <UserOutlined />,
      label: '个人中心',
    },
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: '退出登录',
      danger: true,
    },
  ]

  const handleMenuClick = ({ key }) => {
    navigate(key)
  }

  const handleUserMenuClick = ({ key }) => {
    if (key === 'logout') {
      logout()
      navigate('/login')
    }
  }

  return (
    <AntLayout style={{ minHeight: '100vh' }}>
      <Sider theme="dark" width={200}>
        <div style={{ 
          height: 64, 
          display: 'flex', 
          alignItems: 'center', 
          justifyContent: 'center',
          color: '#fff',
          fontSize: 18,
          fontWeight: 'bold',
          borderBottom: '1px solid rgba(255,255,255,0.1)'
        }}>
          OpenFMS
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={handleMenuClick}
          style={{ borderRight: 0 }}
        />
      </Sider>
      <AntLayout>
        <Header style={{ 
          background: '#fff', 
          padding: '0 24px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'flex-end',
          boxShadow: '0 1px 4px rgba(0,0,0,0.1)'
        }}>
          <Dropdown
            menu={{ items: userMenuItems, onClick: handleUserMenuClick }}
            placement="bottomRight"
          >
            <Space style={{ cursor: 'pointer' }}>
              <Avatar icon={<UserOutlined />} />
              <span>{user?.username || '用户'}</span>
            </Space>
          </Dropdown>
        </Header>
        <Content style={{ 
          margin: 0, 
          padding: 0, 
          background: '#f0f2f5',
          minHeight: 280
        }}>
          <Outlet />
        </Content>
      </AntLayout>
    </AntLayout>
  )
}

export default Layout
