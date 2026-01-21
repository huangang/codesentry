import React from 'react';
import { Layout, Menu, Dropdown, Avatar, Space, Typography, Select } from 'antd';
import {
  DashboardOutlined,
  FileSearchOutlined,
  ProjectOutlined,
  TeamOutlined,
  RobotOutlined,
  NotificationOutlined,
  FileTextOutlined,
  LogoutOutlined,
  UserOutlined,
  SettingOutlined,
  GlobalOutlined,
  BookOutlined,
} from '@ant-design/icons';
import { useNavigate, useLocation, Outlet } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuthStore } from '../stores/authStore';

const { Header, Sider, Content } = Layout;
const { Text } = Typography;

const MainLayout: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { user, logout } = useAuthStore();
  const { t, i18n } = useTranslation();

  const menuItems = [
    { key: '/admin/dashboard', icon: <DashboardOutlined />, label: t('menu.dashboard') },
    { key: '/admin/review-logs', icon: <FileSearchOutlined />, label: t('menu.reviewLogs') },
    { key: '/admin/projects', icon: <ProjectOutlined />, label: t('menu.projects') },
    { key: '/admin/member-analysis', icon: <TeamOutlined />, label: t('menu.memberAnalysis') },
    { key: '/admin/llm-models', icon: <RobotOutlined />, label: t('menu.llmModels') },
    { key: '/admin/prompts', icon: <BookOutlined />, label: t('menu.prompts') },
    { key: '/admin/im-bots', icon: <NotificationOutlined />, label: t('menu.imBots') },
    { key: '/admin/sys-logs', icon: <FileTextOutlined />, label: t('menu.systemLogs') },
  ];

  const handleMenuClick = (info: { key: string }) => {
    navigate(info.key);
  };

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  const handleLanguageChange = (lang: string) => {
    i18n.changeLanguage(lang);
  };

  const userMenuItems = [
    {
      key: 'profile',
      icon: <SettingOutlined />,
      label: t('menu.settings'),
    },
    {
      type: 'divider' as const,
    },
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: t('auth.logout'),
      onClick: handleLogout,
    },
  ];

  const currentPageTitle = menuItems.find(item => item.key === location.pathname)?.label || 'CodeSentry';

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider
        width={220}
        theme="dark"
        style={{
          overflow: 'auto',
          height: '100vh',
          position: 'fixed',
          left: 0,
          top: 0,
          bottom: 0,
        }}
      >
        <div style={{
          height: 64,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          borderBottom: '1px solid rgba(255,255,255,0.1)',
        }}>
          <span style={{ 
            color: '#fff', 
            fontSize: 20, 
            fontWeight: 700,
            letterSpacing: 1,
          }}>
            CodeSentry
          </span>
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={handleMenuClick}
          style={{ borderRight: 0, marginTop: 8 }}
        />
        <div style={{
          position: 'absolute',
          bottom: 16,
          left: 0,
          right: 0,
          padding: '0 16px',
        }}>
          <div style={{
            padding: '12px',
            background: 'rgba(255,255,255,0.05)',
            borderRadius: 8,
            display: 'flex',
            alignItems: 'center',
            gap: 12,
          }}>
            <Avatar 
              size={36} 
              icon={<UserOutlined />}
              style={{ backgroundColor: '#1890ff' }}
            />
            <div style={{ flex: 1, overflow: 'hidden' }}>
              <div style={{ 
                color: '#fff', 
                fontWeight: 500,
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
              }}>
                {user?.nickname || user?.username || 'User'}
              </div>
              <div style={{ color: 'rgba(255,255,255,0.5)', fontSize: 12 }}>
                {user?.role === 'admin' ? (i18n.language === 'zh' ? '管理员' : 'Admin') : (i18n.language === 'zh' ? '用户' : 'User')}
              </div>
            </div>
          </div>
        </div>
      </Sider>
      <Layout style={{ marginLeft: 220 }}>
        <Header style={{
          padding: '0 24px',
          background: '#fff',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          boxShadow: '0 1px 4px rgba(0,0,0,0.08)',
          position: 'sticky',
          top: 0,
          zIndex: 10,
        }}>
          <Text strong style={{ fontSize: 18 }}>
            {currentPageTitle}
          </Text>
          <Space size="middle">
            <Select
              value={i18n.language?.startsWith('zh') ? 'zh' : 'en'}
              onChange={handleLanguageChange}
              style={{ width: 100 }}
              suffixIcon={<GlobalOutlined />}
              options={[
                { value: 'en', label: 'English' },
                { value: 'zh', label: '中文' },
              ]}
            />
            <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
              <Space style={{ cursor: 'pointer' }}>
                <Avatar 
                  size="small" 
                  icon={<UserOutlined />}
                  style={{ backgroundColor: '#1890ff' }}
                />
                <span>{user?.nickname || user?.username}</span>
              </Space>
            </Dropdown>
          </Space>
        </Header>
        <Content style={{
          margin: 24,
          minHeight: 280,
        }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
};

export default MainLayout;
