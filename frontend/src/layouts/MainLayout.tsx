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
  KeyOutlined,
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
    { key: '/admin/git-credentials', icon: <KeyOutlined />, label: t('menu.gitCredentials') },
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
        width={240}
        style={{
          overflow: 'auto',
          height: '100vh',
          position: 'fixed',
          left: 0,
          top: 0,
          bottom: 0,
          background: 'linear-gradient(180deg, #0f172a 0%, #1e3a8a 100%)',
          borderRight: '1px solid rgba(255,255,255,0.05)'
        }}
      >
        <div style={{
          height: 80,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          padding: '0 20px',
        }}>
          <div style={{
            display: 'flex',
            alignItems: 'center',
            gap: 12,
            background: 'rgba(255,255,255,0.05)',
            padding: '8px 16px',
            borderRadius: 12,
            width: '100%',
            justifyContent: 'center',
            border: '1px solid rgba(255,255,255,0.1)'
          }}>
            <img src="/codesentry-icon.png" alt="Logo" style={{ width: 24, height: 24 }} />
            <span style={{
              color: '#fff',
              fontSize: 16,
              fontWeight: 600,
              letterSpacing: 0.5,
            }}>
              CodeSentry
            </span>
          </div>
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={handleMenuClick}
          style={{
            borderRight: 0,
            background: 'transparent',
            padding: '0 12px'
          }}
        />
        <div style={{
          position: 'absolute',
          bottom: 24,
          left: 0,
          right: 0,
          padding: '0 24px',
        }}>
          <div style={{
            padding: '16px',
            background: 'rgba(255,255,255,0.05)',
            borderRadius: 12,
            display: 'flex',
            alignItems: 'center',
            gap: 12,
            border: '1px solid rgba(255,255,255,0.05)',
            backdropFilter: 'blur(10px)'
          }}>
            <Avatar
              size={40}
              icon={<UserOutlined />}
              style={{
                backgroundColor: '#06b6d4',
                boxShadow: '0 4px 6px rgba(6, 182, 212, 0.2)'
              }}
            />
            <div style={{ flex: 1, overflow: 'hidden' }}>
              <div style={{
                color: '#fff',
                fontWeight: 600,
                fontSize: 14,
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
              }}>
                {user?.nickname || user?.username || 'User'}
              </div>
              <div style={{ color: 'rgba(255,255,255,0.6)', fontSize: 12 }}>
                {user?.role === 'admin' ? (i18n.language === 'zh' ? '管理员' : 'Admin') : (i18n.language === 'zh' ? '用户' : 'User')}
              </div>
            </div>
          </div>
        </div>
      </Sider>
      <Layout style={{ marginLeft: 240, background: '#f8fafc' }}>
        <Header style={{
          padding: '0 32px',
          background: 'rgba(255, 255, 255, 0.8)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          backdropFilter: 'blur(8px)',
          borderBottom: '1px solid #f1f5f9',
          position: 'sticky',
          top: 0,
          zIndex: 10,
          height: 64,
        }}>
          <Text strong style={{ fontSize: 20, color: '#0f172a' }}>
            {currentPageTitle}
          </Text>
          <Space size="large">
            <Select
              value={i18n.language?.startsWith('zh') ? 'zh' : 'en'}
              onChange={handleLanguageChange}
              style={{ width: 110 }}
              bordered={false}
              suffixIcon={<GlobalOutlined style={{ color: '#64748b' }} />}
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
                  style={{ backgroundColor: '#06b6d4' }}
                />
                <span style={{ color: '#334155', fontWeight: 500 }}>{user?.nickname || user?.username}</span>
              </Space>
            </Dropdown>
          </Space>
        </Header>
        <Content style={{
          margin: '32px 32px',
          minHeight: 280,
        }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
};

export default MainLayout;
