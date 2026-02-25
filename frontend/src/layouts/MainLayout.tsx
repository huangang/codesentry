import React, { useMemo, useState, useEffect } from 'react';
import { Layout, Menu, Dropdown, Avatar, Space, Typography, Select, Modal, Form, Input, message, Drawer, Button, Tooltip } from 'antd';
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
  LockOutlined,
  ScheduleOutlined,
  BarChartOutlined,
  BugOutlined,
  SafetyOutlined,
  MenuOutlined,
  CloseOutlined,
  SunOutlined,
  MoonOutlined,
  GithubOutlined,
} from '@ant-design/icons';
import { useNavigate, useLocation, Outlet } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuthStore } from '../stores/authStore';
import { useThemeStore } from '../stores/themeStore';
import { usePermission } from '../hooks';
import { authApi } from '../services';
import { stopProactiveRefresh } from '../services/api';
import GlobalSearch from '../components/GlobalSearch';
import NotificationBell from '../components/NotificationBell';

const { Header, Sider, Content } = Layout;
const { Text } = Typography;

const MOBILE_BREAKPOINT = 768;

const MainLayout: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { user, logout } = useAuthStore();
  const { isDark, toggleTheme } = useThemeStore();
  const { t, i18n } = useTranslation();
  const { canAccess } = usePermission();
  const [passwordModalVisible, setPasswordModalVisible] = useState(false);
  const [passwordLoading, setPasswordLoading] = useState(false);
  const [passwordForm] = Form.useForm();
  const [mobileMenuVisible, setMobileMenuVisible] = useState(false);
  const [isMobile, setIsMobile] = useState(false);

  useEffect(() => {
    const checkMobile = () => {
      setIsMobile(window.innerWidth < MOBILE_BREAKPOINT);
    };
    checkMobile();
    window.addEventListener('resize', checkMobile);
    return () => window.removeEventListener('resize', checkMobile);
  }, []);

  useEffect(() => {
    setMobileMenuVisible(false);
  }, [location.pathname]);

  const allMenuItems = [
    { key: '/admin/dashboard', icon: <DashboardOutlined />, label: t('menu.dashboard') },
    { key: '/admin/review-logs', icon: <FileSearchOutlined />, label: t('menu.reviewLogs') },
    { key: '/admin/projects', icon: <ProjectOutlined />, label: t('menu.projects') },
    { key: '/admin/member-analysis', icon: <TeamOutlined />, label: t('menu.memberAnalysis') },
    { key: '/admin/llm-models', icon: <RobotOutlined />, label: t('menu.llmModels') },
    { key: '/admin/prompts', icon: <BookOutlined />, label: t('menu.prompts') },
    { key: '/admin/im-bots', icon: <NotificationOutlined />, label: t('menu.imBots') },
    { key: '/admin/daily-reports', icon: <ScheduleOutlined />, label: t('menu.dailyReports') },
    { key: '/admin/git-credentials', icon: <KeyOutlined />, label: t('menu.gitCredentials') },
    { key: '/admin/users', icon: <UserOutlined />, label: t('menu.users') },
    { key: '/admin/sys-logs', icon: <FileTextOutlined />, label: t('menu.systemLogs') },
    { key: '/admin/reports', icon: <BarChartOutlined />, label: t('menu.reports', 'Reports') },
    { key: '/admin/issue-trackers', icon: <BugOutlined />, label: t('menu.issueTrackers', 'Issue Trackers') },
    { key: '/admin/review-rules', icon: <SafetyOutlined />, label: t('menu.reviewRules', 'Review Rules') },
    { key: '/admin/settings', icon: <SettingOutlined />, label: t('menu.settings') },
  ];

  const menuItems = useMemo(
    () => allMenuItems.filter((item) => canAccess(item.key)),
    [canAccess, t]
  );

  const handleMenuClick = (info: { key: string }) => {
    navigate(info.key);
    if (isMobile) {
      setMobileMenuVisible(false);
    }
  };

  const handleLogout = async () => {
    try {
      await authApi.logout();
    } catch {
    } finally {
      stopProactiveRefresh();
      logout();
      navigate('/login');
    }
  };

  const handleLanguageChange = (lang: string) => {
    i18n.changeLanguage(lang);
  };

  const handleChangePassword = async (values: { oldPassword: string; newPassword: string }) => {
    setPasswordLoading(true);
    try {
      await authApi.changePassword(values.oldPassword, values.newPassword);
      message.success(t('auth.changePasswordSuccess', 'Password changed successfully'));
      setPasswordModalVisible(false);
      passwordForm.resetFields();
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: string } } };
      message.error(err.response?.data?.error || t('common.error'));
    } finally {
      setPasswordLoading(false);
    }
  };

  const isLocalUser = user?.auth_type === 'local';

  const userMenuItems = [
    {
      key: 'profile',
      icon: <SettingOutlined />,
      label: t('menu.settings'),
    },
    ...(isLocalUser ? [{
      key: 'changePassword',
      icon: <LockOutlined />,
      label: t('auth.changePassword', 'Change Password'),
      onClick: () => setPasswordModalVisible(true),
    }] : []),
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

  const userInfoCard = (
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
  );

  const sidebarContent = (forMobile: boolean) => (
    <>
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
      {forMobile ? (
        <div style={{ padding: '24px' }}>
          {userInfoCard}
        </div>
      ) : (
        <div style={{
          position: 'absolute',
          bottom: 24,
          left: 0,
          right: 0,
          padding: '0 24px',
        }}>
          {userInfoCard}
        </div>
      )}
    </>
  );

  return (
    <Layout style={{ minHeight: '100vh' }}>
      {!isMobile && (
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
          {sidebarContent(false)}
        </Sider>
      )}

      <Drawer
        placement="left"
        open={mobileMenuVisible}
        onClose={() => setMobileMenuVisible(false)}
        width={280}
        closable={false}
        styles={{
          body: {
            padding: 0,
            background: 'linear-gradient(180deg, #0f172a 0%, #1e3a8a 100%)',
          },
          header: {
            display: 'none',
          },
        }}
        className="mobile-menu-drawer"
      >
        <div style={{
          display: 'flex',
          justifyContent: 'flex-end',
          padding: '12px 16px',
        }}>
          <Button
            type="text"
            icon={<CloseOutlined style={{ color: '#fff', fontSize: 18 }} />}
            onClick={() => setMobileMenuVisible(false)}
          />
        </div>
        {sidebarContent(true)}
      </Drawer>

      <Layout
        className="main-layout"
        style={{
          marginLeft: isMobile ? 0 : 240,
          background: isDark ? '#0f172a' : '#f8fafc'
        }}
      >
        <Header
          className="main-header"
          style={{
            padding: isMobile ? '0 12px' : '0 32px',
            background: isDark ? 'rgba(30, 41, 59, 0.9)' : 'rgba(255, 255, 255, 0.8)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            backdropFilter: 'blur(8px)',
            borderBottom: isDark ? '1px solid #334155' : '1px solid #f1f5f9',
            position: 'sticky',
            top: 0,
            zIndex: 10,
            height: isMobile ? 56 : 64,
          }}
        >
          <Space>
            {isMobile && (
              <Button
                type="text"
                icon={<MenuOutlined style={{ fontSize: 20, color: isDark ? '#f1f5f9' : undefined }} />}
                onClick={() => setMobileMenuVisible(true)}
                style={{ marginRight: 8 }}
              />
            )}
            <Text
              strong
              className="header-title"
              style={{ fontSize: isMobile ? 16 : 20, color: isDark ? '#f1f5f9' : '#0f172a' }}
            >
              {currentPageTitle}
            </Text>
          </Space>
          {!isMobile && <GlobalSearch />}
          <Space size={isMobile ? 'small' : 'large'}>
            <NotificationBell />
            <Tooltip title="GitHub">
              <Button
                type="text"
                icon={<GithubOutlined style={{ fontSize: 18, color: isDark ? '#e2e8f0' : '#64748b' }} />}
                onClick={() => window.open('https://github.com/huangang/codesentry', '_blank')}
                style={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}
              />
            </Tooltip>
            <Tooltip title={isDark ? 'Light Mode' : 'Dark Mode'}>
              <Button
                type="text"
                icon={isDark ? <SunOutlined style={{ fontSize: 18, color: '#fbbf24' }} /> : <MoonOutlined style={{ fontSize: 18, color: '#64748b' }} />}
                onClick={toggleTheme}
                style={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}
              />
            </Tooltip>
            <Select
              value={i18n.language?.startsWith('zh') ? 'zh' : 'en'}
              onChange={handleLanguageChange}
              style={{ width: isMobile ? 80 : 110 }}
              bordered={false}
              className="header-language-select"
              suffixIcon={<GlobalOutlined style={{ color: isDark ? '#94a3b8' : '#64748b' }} />}
              options={[
                { value: 'en', label: isMobile ? 'EN' : 'English' },
                { value: 'zh', label: isMobile ? '中' : '中文' },
              ]}
            />
            <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
              <Space style={{ cursor: 'pointer' }} className="header-user-dropdown">
                <Avatar
                  size="small"
                  icon={<UserOutlined />}
                  style={{ backgroundColor: '#06b6d4' }}
                />
                {!isMobile && (
                  <span style={{ color: isDark ? '#f1f5f9' : '#334155', fontWeight: 500 }}>
                    {user?.nickname || user?.username}
                  </span>
                )}
              </Space>
            </Dropdown>
          </Space>
        </Header>
        <Content
          className="main-content"
          style={{
            margin: isMobile ? '16px 12px' : '32px 32px',
            minHeight: 280,
          }}
        >
          <Outlet />
        </Content>
      </Layout>

      <Modal
        title={t('auth.changePassword', 'Change Password')}
        open={passwordModalVisible}
        onCancel={() => {
          setPasswordModalVisible(false);
          passwordForm.resetFields();
        }}
        onOk={() => passwordForm.submit()}
        confirmLoading={passwordLoading}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
      >
        <Form
          form={passwordForm}
          layout="vertical"
          onFinish={handleChangePassword}
        >
          <Form.Item
            name="oldPassword"
            label={t('auth.oldPassword', 'Current Password')}
            rules={[{ required: true, message: t('auth.pleaseInputOldPassword', 'Please input current password') }]}
          >
            <Input.Password />
          </Form.Item>
          <Form.Item
            name="newPassword"
            label={t('auth.newPassword', 'New Password')}
            rules={[
              { required: true, message: t('auth.pleaseInputNewPassword', 'Please input new password') },
              { min: 6, message: t('auth.passwordMinLength', 'Password must be at least 6 characters') },
            ]}
          >
            <Input.Password />
          </Form.Item>
          <Form.Item
            name="confirmPassword"
            label={t('auth.confirmPassword', 'Confirm Password')}
            dependencies={['newPassword']}
            rules={[
              { required: true, message: t('auth.pleaseConfirmPassword', 'Please confirm password') },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('newPassword') === value) {
                    return Promise.resolve();
                  }
                  return Promise.reject(new Error(t('auth.passwordMismatch', 'Passwords do not match')));
                },
              }),
            ]}
          >
            <Input.Password />
          </Form.Item>
        </Form>
      </Modal>
    </Layout>
  );
};

export default MainLayout;
