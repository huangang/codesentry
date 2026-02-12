import React, { useState, useEffect } from 'react';
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
import { type DailyReportConfig, type ChunkedReviewConfig, type FileContextConfig } from '../services';
import type { LDAPConfig } from '../types';
import {
  useLDAPConfig,
  useDailyReportConfig,
  useChunkedReviewConfig,
  useFileContextConfig,
  useAuthSessionConfig,
  useActiveLLMConfigs,
  useActiveImBots,
  useUpdateLDAPConfig,
  useUpdateDailyReportConfig,
  useUpdateChunkedReviewConfig,
  useUpdateFileContextConfig,
  useUpdateAuthSessionConfig,
  useHolidayCountries,
} from '../hooks/queries';

const Settings: React.FC = () => {
  const { t } = useTranslation();
  const [ldapForm] = Form.useForm();
  const [dailyReportForm] = Form.useForm();
  const [chunkedReviewForm] = Form.useForm();
  const [fileContextForm] = Form.useForm();
  const [authSessionForm] = Form.useForm();
  const [ldapEnabled, setLdapEnabled] = useState(false);
  const [dailyReportEnabled, setDailyReportEnabled] = useState(false);
  const [chunkedReviewEnabled, setChunkedReviewEnabled] = useState(false);
  const [fileContextEnabled, setFileContextEnabled] = useState(false);

  // Queries
  const { data: ldapConfig, isLoading: ldapLoading } = useLDAPConfig();
  const { data: dailyReportConfig, isLoading: dailyReportLoading } = useDailyReportConfig();
  const { data: chunkedReviewConfig, isLoading: chunkedReviewLoading } = useChunkedReviewConfig();
  const { data: fileContextConfig, isLoading: fileContextLoading } = useFileContextConfig();
  const { data: authSessionConfig, isLoading: authSessionLoading } = useAuthSessionConfig();
  const { data: llmConfigs } = useActiveLLMConfigs();
  const { data: imBots } = useActiveImBots();
  const { data: holidayCountries } = useHolidayCountries();
  const [workdaysOnly, setWorkdaysOnly] = useState(true);

  // Mutations
  const updateLDAP = useUpdateLDAPConfig();
  const updateDailyReport = useUpdateDailyReportConfig();
  const updateChunkedReview = useUpdateChunkedReviewConfig();
  const updateFileContext = useUpdateFileContextConfig();
  const updateAuthSession = useUpdateAuthSessionConfig();

  const isLoading = ldapLoading || dailyReportLoading || chunkedReviewLoading || fileContextLoading || authSessionLoading;

  // Set form values when data loads
  useEffect(() => {
    if (ldapConfig) {
      ldapForm.setFieldsValue({ ...ldapConfig, port: ldapConfig.port || 389, user_filter: ldapConfig.user_filter || '(uid=%s)', bind_password: '' });
      setLdapEnabled(ldapConfig.enabled);
    }
  }, [ldapConfig, ldapForm]);

  useEffect(() => {
    if (dailyReportConfig) {
      dailyReportForm.setFieldsValue({
        enabled: dailyReportConfig.enabled,
        time: dailyReportConfig.time ? dayjs(dailyReportConfig.time, 'HH:mm') : dayjs('18:00', 'HH:mm'),
        timezone: dailyReportConfig.timezone || 'Asia/Shanghai',
        low_score: dailyReportConfig.low_score || 60,
        llm_config_id: dailyReportConfig.llm_config_id || undefined,
        im_bot_ids: dailyReportConfig.im_bot_ids || [],
        workdays_only: dailyReportConfig.workdays_only ?? true,
        holiday_country: dailyReportConfig.holiday_country || 'CN',
      });
      setDailyReportEnabled(dailyReportConfig.enabled);
      setWorkdaysOnly(dailyReportConfig.workdays_only ?? true);
    }
  }, [dailyReportConfig, dailyReportForm]);

  useEffect(() => {
    if (chunkedReviewConfig) {
      chunkedReviewForm.setFieldsValue({ enabled: chunkedReviewConfig.enabled, threshold: chunkedReviewConfig.threshold || 50000, max_tokens_per_batch: chunkedReviewConfig.max_tokens_per_batch || 30000 });
      setChunkedReviewEnabled(chunkedReviewConfig.enabled);
    }
  }, [chunkedReviewConfig, chunkedReviewForm]);

  useEffect(() => {
    if (fileContextConfig) {
      fileContextForm.setFieldsValue({ enabled: fileContextConfig.enabled, extract_functions: fileContextConfig.extract_functions, max_file_size: fileContextConfig.max_file_size || 102400, max_files: fileContextConfig.max_files || 10 });
      setFileContextEnabled(fileContextConfig.enabled);
    }
  }, [fileContextConfig, fileContextForm]);

  useEffect(() => {
    if (authSessionConfig) {
      authSessionForm.setFieldsValue({
        access_token_expire_hours: authSessionConfig.access_token_expire_hours,
        refresh_token_expire_hours: authSessionConfig.refresh_token_expire_hours,
      });
    }
  }, [authSessionConfig, authSessionForm]);

  const handleLdapSave = async () => {
    try {
      const values = await ldapForm.validateFields();
      const payload: Partial<LDAPConfig> = { ...values };
      if (!values.bind_password) delete payload.bind_password;
      await updateLDAP.mutateAsync(payload);
      message.success(t('settings.ldap.saveSuccess'));
      setLdapEnabled(values.enabled);
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: string } } };
      message.error(err.response?.data?.error || t('settings.ldap.saveFailed'));
    }
  };

  const handleDailyReportSave = async () => {
    try {
      const values = await dailyReportForm.validateFields();
      const payload: Partial<DailyReportConfig> = {
        enabled: values.enabled,
        time: values.time ? values.time.format('HH:mm') : '18:00',
        timezone: values.timezone || 'Asia/Shanghai',
        low_score: values.low_score,
        llm_config_id: values.llm_config_id || 0,
        im_bot_ids: values.im_bot_ids || [],
        workdays_only: values.workdays_only ?? true,
        holiday_country: values.holiday_country || 'CN',
      };
      await updateDailyReport.mutateAsync(payload);
      message.success(t('settings.dailyReport.saveSuccess'));
      setDailyReportEnabled(values.enabled);
      setWorkdaysOnly(values.workdays_only ?? true);
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: string } } };
      message.error(err.response?.data?.error || t('common.error'));
    }
  };

  const handleChunkedReviewSave = async () => {
    try {
      const values = await chunkedReviewForm.validateFields();
      const payload: Partial<ChunkedReviewConfig> = { enabled: values.enabled, threshold: values.threshold || 50000, max_tokens_per_batch: values.max_tokens_per_batch || 30000 };
      await updateChunkedReview.mutateAsync(payload);
      message.success(t('settings.chunkedReview.saveSuccess'));
      setChunkedReviewEnabled(values.enabled);
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: string } } };
      message.error(err.response?.data?.error || t('common.error'));
    }
  };

  const handleFileContextSave = async () => {
    try {
      const values = await fileContextForm.validateFields();
      const payload: Partial<FileContextConfig> = {
        enabled: values.enabled,
        max_file_size: values.max_file_size || 102400,
        max_files: values.max_files || 10,
        extract_functions: values.extract_functions ?? true,
      };
      await updateFileContext.mutateAsync(payload);
      message.success(t('settings.fileContext.saveSuccess'));
      setFileContextEnabled(values.enabled);
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: string } } };
      message.error(err.response?.data?.error || t('common.error'));
    }
  };

  const handleAuthSessionSave = async () => {
    try {
      const values = await authSessionForm.validateFields();
      await updateAuthSession.mutateAsync({
        access_token_expire_hours: values.access_token_expire_hours,
        refresh_token_expire_hours: values.refresh_token_expire_hours,
      });
      message.success(t('settings.authSession.saveSuccess'));
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: string } } };
      message.error(err.response?.data?.error || t('common.error'));
    }
  };

  if (isLoading) {
    return <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: 300 }}><Spin size="large" /></div>;
  }

  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      <Card title={t('settings.dailyReport.title')} extra={<Button type="primary" icon={<SaveOutlined />} loading={updateDailyReport.isPending} onClick={handleDailyReportSave}>{t('common.save')}</Button>}>
        <Form form={dailyReportForm} layout="vertical" style={{ maxWidth: 600 }}>
          <Form.Item name="enabled" label={t('settings.dailyReport.enabled')} valuePropName="checked"><Switch onChange={setDailyReportEnabled} /></Form.Item>
          <Row gutter={16}>
            <Col xs={24} sm={8}><Form.Item name="time" label={t('settings.dailyReport.time')}><TimePicker format="HH:mm" style={{ width: '100%' }} disabled={!dailyReportEnabled} /></Form.Item></Col>
            <Col xs={24} sm={8}>
              <Form.Item name="timezone" label={t('settings.dailyReport.timezone')}>
                <Select showSearch disabled={!dailyReportEnabled} options={[
                  { value: 'Asia/Shanghai', label: 'Asia/Shanghai (UTC+8)' }, { value: 'Asia/Tokyo', label: 'Asia/Tokyo (UTC+9)' }, { value: 'Asia/Singapore', label: 'Asia/Singapore (UTC+8)' },
                  { value: 'Asia/Hong_Kong', label: 'Asia/Hong_Kong (UTC+8)' }, { value: 'Europe/London', label: 'Europe/London (UTC+0/+1)' }, { value: 'America/New_York', label: 'America/New_York (UTC-5/-4)' }, { value: 'UTC', label: 'UTC (UTC+0)' },
                ]} />
              </Form.Item>
            </Col>
            <Col xs={24} sm={8}><Form.Item name="low_score" label={t('settings.dailyReport.lowScore')} extra={t('settings.dailyReport.lowScoreHint')}><InputNumber min={0} max={100} style={{ width: '100%' }} disabled={!dailyReportEnabled} /></Form.Item></Col>
          </Row>
          <Form.Item name="llm_config_id" label={t('settings.dailyReport.llmModel')} extra={t('settings.dailyReport.llmModelHint')}>
            <Select allowClear placeholder={t('settings.dailyReport.selectLLM')} disabled={!dailyReportEnabled} options={(llmConfigs || []).map((c) => ({ value: c.id, label: `${c.name} (${c.model})` }))} />
          </Form.Item>
          <Form.Item name="im_bot_ids" label={t('settings.dailyReport.imBots')} extra={t('settings.dailyReport.imBotsHint')}>
            <Select mode="multiple" allowClear placeholder={t('settings.dailyReport.selectIMBots')} disabled={!dailyReportEnabled} options={(imBots || []).filter((b: { daily_report_enabled: boolean }) => b.daily_report_enabled).map((b: { id: number; name: string; type: string }) => ({ value: b.id, label: `${b.name} (${b.type})` }))} />
          </Form.Item>
          <Row gutter={16}>
            <Col xs={24} sm={12}>
              <Form.Item name="workdays_only" label={t('settings.dailyReport.workdaysOnly')} valuePropName="checked" extra={t('settings.dailyReport.workdaysOnlyHint')}>
                <Switch disabled={!dailyReportEnabled} onChange={setWorkdaysOnly} />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item name="holiday_country" label={t('settings.dailyReport.holidayCountry')} extra={t('settings.dailyReport.holidayCountryHint')}>
                <Select showSearch disabled={!dailyReportEnabled || !workdaysOnly} options={(holidayCountries || []).map((c) => ({ value: c.code, label: `${c.name_zh} (${c.name})` }))} />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Card>

      <Card title={t('settings.chunkedReview.title')} extra={<Button type="primary" icon={<SaveOutlined />} loading={updateChunkedReview.isPending} onClick={handleChunkedReviewSave}>{t('common.save')}</Button>}>
        <Form form={chunkedReviewForm} layout="vertical" style={{ maxWidth: 600 }}>
          <Form.Item name="enabled" label={t('settings.chunkedReview.enabled')} valuePropName="checked" extra={t('settings.chunkedReview.enabledHint')}><Switch onChange={setChunkedReviewEnabled} /></Form.Item>
          <Row gutter={16}>
            <Col xs={24} sm={12}><Form.Item name="threshold" label={t('settings.chunkedReview.threshold')} extra={t('settings.chunkedReview.thresholdHint')}><InputNumber min={1000} max={500000} step={1000} style={{ width: '100%' }} disabled={!chunkedReviewEnabled} /></Form.Item></Col>
            <Col xs={24} sm={12}><Form.Item name="max_tokens_per_batch" label={t('settings.chunkedReview.maxTokensPerBatch')} extra={t('settings.chunkedReview.maxTokensPerBatchHint')}><InputNumber min={1000} max={200000} step={1000} style={{ width: '100%' }} disabled={!chunkedReviewEnabled} /></Form.Item></Col>
          </Row>
        </Form>
      </Card>

      <Card title={t('settings.fileContext.title')} extra={<Button type="primary" icon={<SaveOutlined />} loading={updateFileContext.isPending} onClick={handleFileContextSave}>{t('common.save')}</Button>}>
        <Form form={fileContextForm} layout="vertical" style={{ maxWidth: 600 }}>
          <Form.Item name="enabled" label={t('settings.fileContext.enabled')} valuePropName="checked" extra={t('settings.fileContext.enabledHint')}><Switch onChange={setFileContextEnabled} /></Form.Item>
          <Form.Item name="extract_functions" label={t('settings.fileContext.extractFunctions')} valuePropName="checked" extra={t('settings.fileContext.extractFunctionsHint')}><Switch disabled={!fileContextEnabled} /></Form.Item>
          <Row gutter={16}>
            <Col xs={24} sm={12}><Form.Item name="max_file_size" label={t('settings.fileContext.maxFileSize')} extra={t('settings.fileContext.maxFileSizeHint')}><InputNumber min={1024} max={1048576} step={1024} style={{ width: '100%' }} disabled={!fileContextEnabled} addonAfter="bytes" /></Form.Item></Col>
            <Col xs={24} sm={12}><Form.Item name="max_files" label={t('settings.fileContext.maxFiles')} extra={t('settings.fileContext.maxFilesHint')}><InputNumber min={1} max={50} style={{ width: '100%' }} disabled={!fileContextEnabled} /></Form.Item></Col>
          </Row>
        </Form>
      </Card>

      <Card title={t('settings.authSession.title')} extra={<Button type="primary" icon={<SaveOutlined />} loading={updateAuthSession.isPending} onClick={handleAuthSessionSave}>{t('common.save')}</Button>}>
        <Form form={authSessionForm} layout="vertical" style={{ maxWidth: 600 }}>
          <Row gutter={16}>
            <Col xs={24} sm={12}>
              <Form.Item
                name="access_token_expire_hours"
                label={t('settings.authSession.accessTokenExpireHours')}
                extra={t('settings.authSession.accessTokenExpireHoursHint')}
                rules={[{ required: true, message: t('settings.authSession.accessTokenRequired') }]}
              >
                <InputNumber min={1} max={168} style={{ width: '100%' }} addonAfter="hours" />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item
                name="refresh_token_expire_hours"
                label={t('settings.authSession.refreshTokenExpireHours')}
                extra={t('settings.authSession.refreshTokenExpireHoursHint')}
                rules={[{ required: true, message: t('settings.authSession.refreshTokenRequired') }]}
              >
                <InputNumber min={24} max={8760} style={{ width: '100%' }} addonAfter="hours" />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Card>

      <Card title={t('settings.ldap.title')} extra={<Button type="primary" icon={<SaveOutlined />} loading={updateLDAP.isPending} onClick={handleLdapSave}>{t('common.save')}</Button>}>
        <Form form={ldapForm} layout="vertical" style={{ maxWidth: 600 }}>
          <Form.Item name="enabled" label={t('settings.ldap.enabled')} valuePropName="checked"><Switch onChange={setLdapEnabled} /></Form.Item>
          <Row gutter={16}>
            <Col xs={24} sm={16}><Form.Item name="host" label={t('settings.ldap.host')} rules={[{ required: ldapEnabled, message: t('settings.ldap.pleaseInputHost') }]}><Input placeholder="ldap.example.com" disabled={!ldapEnabled} /></Form.Item></Col>
            <Col xs={24} sm={8}><Form.Item name="port" label={t('settings.ldap.port')} rules={[{ required: ldapEnabled, message: t('settings.ldap.pleaseInputPort') }]}><InputNumber min={1} max={65535} style={{ width: '100%' }} disabled={!ldapEnabled} /></Form.Item></Col>
          </Row>
          <Form.Item name="base_dn" label={t('settings.ldap.baseDn')} rules={[{ required: ldapEnabled, message: t('settings.ldap.pleaseInputBaseDn') }]}><Input placeholder="dc=example,dc=com" disabled={!ldapEnabled} /></Form.Item>
          <Form.Item name="bind_dn" label={t('settings.ldap.bindDn')} rules={[{ required: ldapEnabled, message: t('settings.ldap.pleaseInputBindDn') }]}><Input placeholder="cn=admin,dc=example,dc=com" disabled={!ldapEnabled} /></Form.Item>
          <Form.Item name="bind_password" label={t('settings.ldap.bindPassword')} extra={t('settings.ldap.passwordHint')}><Input.Password placeholder="••••••••" disabled={!ldapEnabled} /></Form.Item>
          <Form.Item name="user_filter" label={t('settings.ldap.userFilter')} rules={[{ required: ldapEnabled, message: t('settings.ldap.pleaseInputUserFilter') }]}><Input placeholder="(uid=%s)" disabled={!ldapEnabled} /></Form.Item>
          <Form.Item name="use_ssl" label={t('settings.ldap.useSsl')} valuePropName="checked"><Switch disabled={!ldapEnabled} /></Form.Item>
        </Form>
      </Card>
    </Space>
  );
};

export default Settings;
