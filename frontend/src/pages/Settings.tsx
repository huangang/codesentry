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
  TimePicker,
  Space,
  Select,
} from 'antd';
import { SaveOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import dayjs from 'dayjs';
import { systemConfigApi, llmConfigApi, imBotApi, type DailyReportConfig, type ChunkedReviewConfig } from '../services';
import type { LDAPConfig, LLMConfig, IMBot } from '../types';

const Settings: React.FC = () => {
  const { t } = useTranslation();
  const [ldapForm] = Form.useForm();
  const [dailyReportForm] = Form.useForm();
  const [chunkedReviewForm] = Form.useForm();
  const [loading, setLoading] = useState(true);
  const [ldapSaving, setLdapSaving] = useState(false);
  const [dailyReportSaving, setDailyReportSaving] = useState(false);
  const [chunkedReviewSaving, setChunkedReviewSaving] = useState(false);
  const [ldapEnabled, setLdapEnabled] = useState(false);
  const [dailyReportEnabled, setDailyReportEnabled] = useState(false);
  const [chunkedReviewEnabled, setChunkedReviewEnabled] = useState(false);
  const [llmConfigs, setLlmConfigs] = useState<LLMConfig[]>([]);
  const [imBots, setImBots] = useState<IMBot[]>([]);

  const fetchConfig = async () => {
    try {
      setLoading(true);
      const [ldapRes, dailyReportRes, chunkedReviewRes, llmRes, imBotRes] = await Promise.all([
        systemConfigApi.getLDAPConfig(),
        systemConfigApi.getDailyReportConfig(),
        systemConfigApi.getChunkedReviewConfig(),
        llmConfigApi.getActive(),
        imBotApi.getActive(),
      ]);

      const ldapConfig = ldapRes.data;
      ldapForm.setFieldsValue({
        ...ldapConfig,
        port: ldapConfig.port || 389,
        user_filter: ldapConfig.user_filter || '(uid=%s)',
        bind_password: '',
      });
      setLdapEnabled(ldapConfig.enabled);

      const dailyReportConfig = dailyReportRes.data;
      dailyReportForm.setFieldsValue({
        enabled: dailyReportConfig.enabled,
        time: dailyReportConfig.time ? dayjs(dailyReportConfig.time, 'HH:mm') : dayjs('18:00', 'HH:mm'),
        timezone: dailyReportConfig.timezone || 'Asia/Shanghai',
        low_score: dailyReportConfig.low_score || 60,
        llm_config_id: dailyReportConfig.llm_config_id || undefined,
        im_bot_ids: dailyReportConfig.im_bot_ids || [],
      });
      setDailyReportEnabled(dailyReportConfig.enabled);

      const chunkedReviewConfig = chunkedReviewRes.data;
      chunkedReviewForm.setFieldsValue({
        enabled: chunkedReviewConfig.enabled,
        threshold: chunkedReviewConfig.threshold || 50000,
        max_tokens_per_batch: chunkedReviewConfig.max_tokens_per_batch || 30000,
      });
      setChunkedReviewEnabled(chunkedReviewConfig.enabled);

      setLlmConfigs(llmRes.data || []);
      setImBots(imBotRes.data || []);
    } catch {
      ldapForm.setFieldsValue({
        enabled: false,
        port: 389,
        user_filter: '(uid=%s)',
      });
      dailyReportForm.setFieldsValue({
        enabled: false,
        time: dayjs('18:00', 'HH:mm'),
        timezone: 'Asia/Shanghai',
        low_score: 60,
      });
      chunkedReviewForm.setFieldsValue({
        enabled: true,
        threshold: 50000,
        max_tokens_per_batch: 30000,
      });
      message.error(t('common.error'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchConfig();
  }, []);

  const handleLdapSave = async () => {
    try {
      const values = await ldapForm.validateFields();
      setLdapSaving(true);

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
      setLdapSaving(false);
    }
  };

  const handleDailyReportSave = async () => {
    try {
      const values = await dailyReportForm.validateFields();
      setDailyReportSaving(true);

      const payload: Partial<DailyReportConfig> = {
        enabled: values.enabled,
        time: values.time ? values.time.format('HH:mm') : '18:00',
        timezone: values.timezone || 'Asia/Shanghai',
        low_score: values.low_score,
        llm_config_id: values.llm_config_id || 0,
        im_bot_ids: values.im_bot_ids || [],
      };

      await systemConfigApi.updateDailyReportConfig(payload);
      message.success(t('settings.dailyReport.saveSuccess'));
      setDailyReportEnabled(values.enabled);
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: string } } };
      message.error(err.response?.data?.error || t('common.error'));
    } finally {
      setDailyReportSaving(false);
    }
  };

  const handleLdapEnabledChange = (checked: boolean) => {
    setLdapEnabled(checked);
  };

  const handleDailyReportEnabledChange = (checked: boolean) => {
    setDailyReportEnabled(checked);
  };

  const handleChunkedReviewSave = async () => {
    try {
      const values = await chunkedReviewForm.validateFields();
      setChunkedReviewSaving(true);

      const payload: Partial<ChunkedReviewConfig> = {
        enabled: values.enabled,
        threshold: values.threshold || 50000,
        max_tokens_per_batch: values.max_tokens_per_batch || 30000,
      };

      await systemConfigApi.updateChunkedReviewConfig(payload);
      message.success(t('settings.chunkedReview.saveSuccess'));
      setChunkedReviewEnabled(values.enabled);
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: string } } };
      message.error(err.response?.data?.error || t('common.error'));
    } finally {
      setChunkedReviewSaving(false);
    }
  };

  const handleChunkedReviewEnabledChange = (checked: boolean) => {
    setChunkedReviewEnabled(checked);
  };

  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: 300 }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      <Card
        title={t('settings.dailyReport.title')}
        extra={
          <Button type="primary" icon={<SaveOutlined />} loading={dailyReportSaving} onClick={handleDailyReportSave}>
            {t('common.save')}
          </Button>
        }
      >
        <Form form={dailyReportForm} layout="vertical" style={{ maxWidth: 600 }}>
          <Form.Item name="enabled" label={t('settings.dailyReport.enabled')} valuePropName="checked">
            <Switch onChange={handleDailyReportEnabledChange} />
          </Form.Item>

          <Row gutter={16}>
            <Col xs={24} sm={8}>
              <Form.Item
                name="time"
                label={t('settings.dailyReport.time')}
              >
                <TimePicker format="HH:mm" style={{ width: '100%' }} disabled={!dailyReportEnabled} />
              </Form.Item>
            </Col>
            <Col xs={24} sm={8}>
              <Form.Item
                name="timezone"
                label={t('settings.dailyReport.timezone')}
              >
                <Select
                  showSearch
                  disabled={!dailyReportEnabled}
                  options={[
                    { value: 'Asia/Shanghai', label: 'Asia/Shanghai (UTC+8)' },
                    { value: 'Asia/Tokyo', label: 'Asia/Tokyo (UTC+9)' },
                    { value: 'Asia/Singapore', label: 'Asia/Singapore (UTC+8)' },
                    { value: 'Asia/Hong_Kong', label: 'Asia/Hong_Kong (UTC+8)' },
                    { value: 'Asia/Seoul', label: 'Asia/Seoul (UTC+9)' },
                    { value: 'Europe/London', label: 'Europe/London (UTC+0/+1)' },
                    { value: 'Europe/Paris', label: 'Europe/Paris (UTC+1/+2)' },
                    { value: 'Europe/Berlin', label: 'Europe/Berlin (UTC+1/+2)' },
                    { value: 'America/New_York', label: 'America/New_York (UTC-5/-4)' },
                    { value: 'America/Los_Angeles', label: 'America/Los_Angeles (UTC-8/-7)' },
                    { value: 'America/Chicago', label: 'America/Chicago (UTC-6/-5)' },
                    { value: 'UTC', label: 'UTC (UTC+0)' },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col xs={24} sm={8}>
              <Form.Item
                name="low_score"
                label={t('settings.dailyReport.lowScore')}
                extra={t('settings.dailyReport.lowScoreHint')}
              >
                <InputNumber min={0} max={100} style={{ width: '100%' }} disabled={!dailyReportEnabled} />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name="llm_config_id"
            label={t('settings.dailyReport.llmModel')}
            extra={t('settings.dailyReport.llmModelHint')}
          >
            <Select
              allowClear
              placeholder={t('settings.dailyReport.selectLLM')}
              disabled={!dailyReportEnabled}
              options={llmConfigs.map((c) => ({
                value: c.id,
                label: `${c.name} (${c.model})`,
              }))}
            />
          </Form.Item>

          <Form.Item
            name="im_bot_ids"
            label={t('settings.dailyReport.imBots')}
            extra={t('settings.dailyReport.imBotsHint')}
          >
            <Select
              mode="multiple"
              allowClear
              placeholder={t('settings.dailyReport.selectIMBots')}
              disabled={!dailyReportEnabled}
              options={imBots
                .filter((b) => b.daily_report_enabled)
                .map((b) => ({
                  value: b.id,
                  label: `${b.name} (${b.type})`,
                }))}
            />
          </Form.Item>
        </Form>
      </Card>

      <Card
        title={t('settings.chunkedReview.title')}
        extra={
          <Button type="primary" icon={<SaveOutlined />} loading={chunkedReviewSaving} onClick={handleChunkedReviewSave}>
            {t('common.save')}
          </Button>
        }
      >
        <Form form={chunkedReviewForm} layout="vertical" style={{ maxWidth: 600 }}>
          <Form.Item
            name="enabled"
            label={t('settings.chunkedReview.enabled')}
            valuePropName="checked"
            extra={t('settings.chunkedReview.enabledHint')}
          >
            <Switch onChange={handleChunkedReviewEnabledChange} />
          </Form.Item>

          <Row gutter={16}>
            <Col xs={24} sm={12}>
              <Form.Item
                name="threshold"
                label={t('settings.chunkedReview.threshold')}
                extra={t('settings.chunkedReview.thresholdHint')}
              >
                <InputNumber min={1000} max={500000} step={1000} style={{ width: '100%' }} disabled={!chunkedReviewEnabled} />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item
                name="max_tokens_per_batch"
                label={t('settings.chunkedReview.maxTokensPerBatch')}
                extra={t('settings.chunkedReview.maxTokensPerBatchHint')}
              >
                <InputNumber min={1000} max={200000} step={1000} style={{ width: '100%' }} disabled={!chunkedReviewEnabled} />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Card>

      <Card
        title={t('settings.ldap.title')}
        extra={
          <Button type="primary" icon={<SaveOutlined />} loading={ldapSaving} onClick={handleLdapSave}>
            {t('common.save')}
          </Button>
        }
      >
        <Form form={ldapForm} layout="vertical" style={{ maxWidth: 600 }}>
          <Form.Item name="enabled" label={t('settings.ldap.enabled')} valuePropName="checked">
            <Switch onChange={handleLdapEnabledChange} />
          </Form.Item>

          <Row gutter={16}>
            <Col xs={24} sm={16}>
              <Form.Item
                name="host"
                label={t('settings.ldap.host')}
                rules={[{ required: ldapEnabled, message: t('settings.ldap.pleaseInputHost') }]}
              >
                <Input placeholder="ldap.example.com" disabled={!ldapEnabled} />
              </Form.Item>
            </Col>
            <Col xs={24} sm={8}>
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
    </Space>
  );
};

export default Settings;
