import React, { useEffect, useState } from 'react';
import {
  Card,
  Form,
  Input,
  InputNumber,
  Switch,
  Button,
  message,
  Spin,
  Row,
  Col,
} from 'antd';
import { SaveOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { systemConfigApi } from '../services';
import type { LDAPConfig } from '../types';

const Settings: React.FC = () => {
  const { t } = useTranslation();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [ldapEnabled, setLdapEnabled] = useState(false);

  const fetchConfig = async () => {
    try {
      setLoading(true);
      const res = await systemConfigApi.getLDAPConfig();
      const config = res.data;
      form.setFieldsValue({
        ...config,
        port: config.port || 389,
        user_filter: config.user_filter || '(uid=%s)',
        bind_password: '',
      });
      setLdapEnabled(config.enabled);
    } catch (error) {
      form.setFieldsValue({
        enabled: false,
        port: 389,
        user_filter: '(uid=%s)',
      });
      message.error(t('common.error'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchConfig();
  }, []);

  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      setSaving(true);
      
      const payload: Partial<LDAPConfig> = { ...values };
      if (!values.bind_password) {
        delete payload.bind_password;
      }
      
      await systemConfigApi.updateLDAPConfig(payload);
      message.success(t('settings.ldap.saveSuccess'));
      setLdapEnabled(values.enabled);
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: string } } };
      message.error(err.response?.data?.error || t('settings.ldap.saveFailed'));
    } finally {
      setSaving(false);
    }
  };

  const handleEnabledChange = (checked: boolean) => {
    setLdapEnabled(checked);
  };

  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: 300 }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <Card
      title={t('settings.ldap.title')}
      extra={
        <Button type="primary" icon={<SaveOutlined />} loading={saving} onClick={handleSave}>
          {t('common.save')}
        </Button>
      }
    >
      <Form form={form} layout="vertical" style={{ maxWidth: 600 }}>
        <Form.Item name="enabled" label={t('settings.ldap.enabled')} valuePropName="checked">
          <Switch onChange={handleEnabledChange} />
        </Form.Item>

        <Row gutter={16}>
          <Col span={16}>
            <Form.Item
              name="host"
              label={t('settings.ldap.host')}
              rules={[{ required: ldapEnabled, message: t('settings.ldap.pleaseInputHost') }]}
            >
              <Input placeholder="ldap.example.com" disabled={!ldapEnabled} />
            </Form.Item>
          </Col>
          <Col span={8}>
            <Form.Item
              name="port"
              label={t('settings.ldap.port')}
              rules={[{ required: ldapEnabled, message: t('settings.ldap.pleaseInputPort') }]}
            >
              <InputNumber min={1} max={65535} style={{ width: '100%' }} disabled={!ldapEnabled} />
            </Form.Item>
          </Col>
        </Row>

        <Form.Item
          name="base_dn"
          label={t('settings.ldap.baseDn')}
          rules={[{ required: ldapEnabled, message: t('settings.ldap.pleaseInputBaseDn') }]}
        >
          <Input placeholder="dc=example,dc=com" disabled={!ldapEnabled} />
        </Form.Item>

        <Form.Item
          name="bind_dn"
          label={t('settings.ldap.bindDn')}
          rules={[{ required: ldapEnabled, message: t('settings.ldap.pleaseInputBindDn') }]}
        >
          <Input placeholder="cn=admin,dc=example,dc=com" disabled={!ldapEnabled} />
        </Form.Item>

        <Form.Item
          name="bind_password"
          label={t('settings.ldap.bindPassword')}
          extra={t('settings.ldap.passwordHint')}
        >
          <Input.Password placeholder="••••••••" disabled={!ldapEnabled} />
        </Form.Item>

        <Form.Item
          name="user_filter"
          label={t('settings.ldap.userFilter')}
          rules={[{ required: ldapEnabled, message: t('settings.ldap.pleaseInputUserFilter') }]}
        >
          <Input placeholder="(uid=%s)" disabled={!ldapEnabled} />
        </Form.Item>

        <Form.Item name="use_ssl" label={t('settings.ldap.useSsl')} valuePropName="checked">
          <Switch disabled={!ldapEnabled} />
        </Form.Item>
      </Form>
    </Card>
  );
};

export default Settings;
