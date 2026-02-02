import React, { useState } from 'react';
import {
  Card,
  Table,
  Button,
  Space,
  Input,
  Select,
  Tag,
  Modal,
  Form,
  Switch,
  message,
  Popconfirm,
} from 'antd';
import { PlusOutlined, SearchOutlined, ReloadOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import type { IMBot } from '../types';
import { useModal } from '../hooks';
import {
  useIMBots,
  useCreateIMBot,
  useUpdateIMBot,
  useDeleteIMBot,
  type IMBotFilters,
} from '../hooks/queries';
import { IM_BOT_TYPES } from '../constants';

const IMBots: React.FC = () => {
  const { t, i18n } = useTranslation();
  const [form] = Form.useForm();
  const [searchName, setSearchName] = useState('');
  const [botType, setBotType] = useState<string>('');
  const [filters, setFilters] = useState<IMBotFilters>({ page: 1, page_size: 10 });

  const modal = useModal<IMBot>();

  const { data: botsData, isLoading } = useIMBots(filters);
  const createBot = useCreateIMBot();
  const updateBot = useUpdateIMBot();
  const deleteBot = useDeleteIMBot();

  const getBotTypeLabel = (type: string) => {
    switch (type) {
      case IM_BOT_TYPES.WECHAT_WORK: return t('imBots.wecom');
      case IM_BOT_TYPES.DINGTALK: return t('imBots.dingtalk');
      case IM_BOT_TYPES.FEISHU: return t('imBots.feishu');
      case IM_BOT_TYPES.SLACK: return t('imBots.slack');
      case IM_BOT_TYPES.DISCORD: return t('imBots.discord');
      case IM_BOT_TYPES.TEAMS: return t('imBots.teams');
      case IM_BOT_TYPES.TELEGRAM: return t('imBots.telegram');
      default: return type;
    }
  };

  const needsSecret = (type: string) => type === IM_BOT_TYPES.DINGTALK || type === IM_BOT_TYPES.FEISHU;
  const needsExtra = (type: string) => type === IM_BOT_TYPES.TELEGRAM;

  const getSecretHelpText = (type: string) => {
    const isZh = i18n.language?.startsWith('zh');
    switch (type) {
      case IM_BOT_TYPES.DINGTALK: return isZh ? '钉钉加签密钥（可选，用于安全验证）' : 'DingTalk signing secret (optional)';
      case IM_BOT_TYPES.FEISHU: return isZh ? '飞书签名密钥（可选，用于安全验证）' : 'Feishu signing secret (optional)';
      default: return '';
    }
  };

  const getWebhookPlaceholder = (type: string) => {
    switch (type) {
      case IM_BOT_TYPES.WECHAT_WORK: return 'https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx';
      case IM_BOT_TYPES.DINGTALK: return 'https://oapi.dingtalk.com/robot/send?access_token=xxx';
      case IM_BOT_TYPES.FEISHU: return 'https://open.feishu.cn/open-apis/bot/v2/hook/xxx';
      case IM_BOT_TYPES.SLACK: return 'https://hooks.slack.com/services/xxx/xxx/xxx';
      case IM_BOT_TYPES.DISCORD: return 'https://discord.com/api/webhooks/xxx/xxx';
      case IM_BOT_TYPES.TEAMS: return 'https://xxx.webhook.office.com/webhookb2/xxx';
      case IM_BOT_TYPES.TELEGRAM: return 'https://api.telegram.org/botXXX:YYY/sendMessage';
      default: return 'https://...';
    }
  };

  const botTypeOptions = [
    { value: IM_BOT_TYPES.WECHAT_WORK, label: t('imBots.wecom') },
    { value: IM_BOT_TYPES.DINGTALK, label: t('imBots.dingtalk') },
    { value: IM_BOT_TYPES.FEISHU, label: t('imBots.feishu') },
    { value: IM_BOT_TYPES.SLACK, label: t('imBots.slack') },
    { value: IM_BOT_TYPES.DISCORD, label: t('imBots.discord') },
    { value: IM_BOT_TYPES.TEAMS, label: t('imBots.teams') },
    { value: IM_BOT_TYPES.TELEGRAM, label: t('imBots.telegram') },
  ];

  const handleSearch = () => {
    const newFilters: IMBotFilters = { page: 1, page_size: filters.page_size };
    if (searchName) newFilters.name = searchName;
    if (botType) newFilters.type = botType;
    setFilters(newFilters);
  };

  const handleReset = () => {
    setSearchName('');
    setBotType('');
    setFilters({ page: 1, page_size: 10 });
  };

  const handlePageChange = (page: number, pageSize: number) => {
    setFilters(prev => ({ ...prev, page, page_size: pageSize }));
  };

  const showCreateModal = () => {
    modal.open();
    form.resetFields();
    form.setFieldsValue({ type: IM_BOT_TYPES.WECHAT_WORK, is_active: true, error_notify: false, daily_report_enabled: false });
  };

  const showEditModal = (record: IMBot) => {
    modal.open(record);
    form.setFieldsValue(record);
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (modal.current) {
        await updateBot.mutateAsync({ id: modal.current.id, data: values });
        message.success(t('imBots.updateSuccess'));
      } else {
        await createBot.mutateAsync(values);
        message.success(t('imBots.createSuccess'));
      }
      modal.close();
    } catch (error: any) {
      message.error(error.response?.data?.error || t('common.error'));
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await deleteBot.mutateAsync(id);
      message.success(t('imBots.deleteSuccess'));
    } catch (error) {
      message.error(t('common.error'));
    }
  };

  const columns: ColumnsType<IMBot> = [
    { title: t('imBots.botName'), dataIndex: 'name', key: 'name', width: 180 },
    { title: t('imBots.botType'), dataIndex: 'type', key: 'type', width: 120, render: (val: string) => <Tag color="blue">{getBotTypeLabel(val)}</Tag> },
    { title: t('imBots.webhook'), dataIndex: 'webhook', key: 'webhook', ellipsis: true },
    { title: t('imBots.isActive'), key: 'is_active', width: 100, render: (_, record) => <Tag color={record.is_active ? 'success' : 'default'}>{record.is_active ? t('common.enabled') : t('common.disabled')}</Tag> },
    { title: t('imBots.errorNotify'), key: 'error_notify', width: 120, render: (_, record) => <Tag color={record.error_notify ? 'warning' : 'default'}>{record.error_notify ? t('common.enabled') : t('common.disabled')}</Tag> },
    { title: t('imBots.dailyReportEnabled'), key: 'daily_report_enabled', width: 100, render: (_, record) => <Tag color={record.daily_report_enabled ? 'processing' : 'default'}>{record.daily_report_enabled ? t('common.enabled') : t('common.disabled')}</Tag> },
    { title: t('common.createdAt'), dataIndex: 'created_at', key: 'created_at', width: 160, render: (val: string) => dayjs(val).format('YYYY-MM-DD HH:mm') },
    {
      title: t('common.actions'), key: 'action', width: 120,
      render: (_, record) => (
        <Space>
          <Button type="link" icon={<EditOutlined />} onClick={() => showEditModal(record)} />
          <Popconfirm title={t('imBots.deleteConfirm')} onConfirm={() => handleDelete(record.id)}>
            <Button type="link" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <>
      <Card>
        <Space style={{ marginBottom: 16 }} wrap>
          <Input placeholder={t('imBots.botName')} style={{ width: 180 }} value={searchName} onChange={(e) => setSearchName(e.target.value)} onPressEnter={handleSearch} />
          <Select placeholder={t('imBots.botType')} allowClear style={{ width: 120 }} value={botType || undefined} onChange={setBotType} options={botTypeOptions} />
          <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>{t('common.search')}</Button>
          <Button icon={<ReloadOutlined />} onClick={handleReset}>{t('common.reset')}</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={showCreateModal}>{t('imBots.createBot')}</Button>
        </Space>

        <Table columns={columns} dataSource={botsData?.items ?? []} rowKey="id" loading={isLoading} scroll={{ x: 900 }}
          pagination={{ current: filters.page, pageSize: filters.page_size, total: botsData?.total ?? 0, showSizeChanger: true, showTotal: (total) => `${t('common.total')} ${total}`, onChange: handlePageChange }} />
      </Card>

      <Modal title={modal.isEdit ? t('imBots.editBot') : t('imBots.createBot')} open={modal.visible} onOk={handleSubmit} onCancel={modal.close} confirmLoading={createBot.isPending || updateBot.isPending} width={520}>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label={t('imBots.botName')} rules={[{ required: true, message: t('imBots.pleaseInputName') }]}><Input placeholder="AI Code Review Bot" /></Form.Item>
          <Form.Item name="type" label={t('imBots.botType')} rules={[{ required: true, message: t('imBots.pleaseSelectType') }]}><Select options={botTypeOptions} /></Form.Item>
          <Form.Item noStyle shouldUpdate={(prev, cur) => prev.type !== cur.type}>
            {({ getFieldValue }) => (
              <Form.Item name="webhook" label={t('imBots.webhook')} rules={[{ required: true, message: t('imBots.pleaseInputWebhook') }]}>
                <Input placeholder={getWebhookPlaceholder(getFieldValue('type'))} />
              </Form.Item>
            )}
          </Form.Item>
          <Form.Item noStyle shouldUpdate={(prev, cur) => prev.type !== cur.type}>
            {({ getFieldValue }) => {
              const type = getFieldValue('type');
              if (!needsSecret(type)) return null;
              return <Form.Item name="secret" label={t('imBots.secret')} extra={getSecretHelpText(type)}><Input.Password placeholder={t('imBots.secretPlaceholder', 'SEC...')} /></Form.Item>;
            }}
          </Form.Item>
          <Form.Item noStyle shouldUpdate={(prev, cur) => prev.type !== cur.type}>
            {({ getFieldValue }) => {
              const type = getFieldValue('type');
              if (!needsExtra(type)) return null;
              return <Form.Item name="extra" label={t('imBots.extra')} rules={[{ required: true, message: t('imBots.pleaseInputExtra') }]} extra={t('imBots.extraHelp')}><Input placeholder="-123456789" /></Form.Item>;
            }}
          </Form.Item>
          <Form.Item name="is_active" label={t('imBots.isActive')} valuePropName="checked"><Switch /></Form.Item>
          <Form.Item name="error_notify" label={t('imBots.errorNotify')} valuePropName="checked" extra={t('imBots.errorNotifyHelp')}><Switch /></Form.Item>
          <Form.Item name="daily_report_enabled" label={t('imBots.dailyReportEnabled')} valuePropName="checked" extra={t('imBots.dailyReportHelp')}><Switch /></Form.Item>
        </Form>
      </Modal>
    </>
  );
};

export default IMBots;
