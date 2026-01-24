import React, { useState, useEffect } from 'react';
import { Form, Input, Button, Card, Tabs, message } from 'antd';
import { UserOutlined, LockOutlined, SafetyCertificateOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { authApi } from '../services';
import { useAuthStore } from '../stores/authStore';

const Login: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [ldapEnabled, setLdapEnabled] = useState(false);
  const [authType, setAuthType] = useState('local');
  const navigate = useNavigate();
  const { isAuthenticated, setAuth } = useAuthStore();
  const { t } = useTranslation();

  useEffect(() => {
    if (isAuthenticated) {
      navigate('/admin/dashboard');
    }
    
    // Check if LDAP is enabled
    authApi.getConfig().then(res => {
      setLdapEnabled(res.data.ldap_enabled);
    }).catch(() => {});
  }, [isAuthenticated, navigate]);

  const handleLogin = async (values: { username: string; password: string }) => {
    setLoading(true);
    try {
      const res = await authApi.login(values.username, values.password, authType);
      setAuth(res.data.token, res.data.user);
      message.success(t('auth.loginSuccess'));
      navigate('/admin/dashboard');
    } catch (error: any) {
      message.error(error.response?.data?.error || t('auth.loginFailed'));
    } finally {
      setLoading(false);
    }
  };

  const loginForm = (
    <Form
      name="login"
      onFinish={handleLogin}
      size="large"
      autoComplete="off"
    >
      <Form.Item
        name="username"
        rules={[{ required: true, message: t('auth.pleaseInputUsername') }]}
      >
        <Input 
          prefix={<UserOutlined />} 
          placeholder={t('auth.username')} 
        />
      </Form.Item>

      <Form.Item
        name="password"
        rules={[{ required: true, message: t('auth.pleaseInputPassword') }]}
      >
        <Input.Password 
          prefix={<LockOutlined />} 
          placeholder={t('auth.password')} 
        />
      </Form.Item>

      <Form.Item>
        <Button 
          type="primary" 
          htmlType="submit" 
          loading={loading}
          block
          style={{ height: 44 }}
        >
          {t('auth.login')}
        </Button>
      </Form.Item>
    </Form>
  );

  const tabItems = [
    {
      key: 'local',
      label: (
        <span>
          <UserOutlined />
          {t('auth.accountLogin')}
        </span>
      ),
      children: loginForm,
    },
  ];

  if (ldapEnabled) {
    tabItems.push({
      key: 'ldap',
      label: (
        <span>
          <SafetyCertificateOutlined />
          {t('auth.ldapLogin')}
        </span>
      ),
      children: loginForm,
    });
  }

  return (
    <div style={{
      minHeight: '100vh',
      display: 'flex',
      justifyContent: 'center',
      alignItems: 'center',
      background: 'linear-gradient(135deg, #1a1a2e 0%, #16213e 50%, #0f3460 100%)',
      padding: '16px',
    }}>
      <Card 
        className="login-card"
        style={{ 
          width: '100%',
          maxWidth: 420, 
          boxShadow: '0 8px 24px rgba(0,0,0,0.2)',
          borderRadius: 8,
        }}
      >
        <div style={{ textAlign: 'center', marginBottom: 32 }}>
          <div style={{ 
            fontSize: 32, 
            fontWeight: 700, 
            color: '#1890ff',
            marginBottom: 8 
          }}>
            <SafetyCertificateOutlined style={{ marginRight: 8 }} />
            CodeSentry
          </div>
          <div style={{ color: '#666', fontSize: 14 }}>
            {t('common.appDescription')}
          </div>
        </div>

        <Tabs 
          activeKey={authType}
          onChange={setAuthType}
          centered
          items={tabItems}
        />
      </Card>
    </div>
  );
};

export default Login;
