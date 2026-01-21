import React, { useState, useEffect } from 'react';
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
import { imBotApi } from '../services';
import type { IMBot } from '../types';

const IMBots: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<IMBot[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [modalVisible, setModalVisible] = useState(false);
  const [currentBot, setCurrentBot] = useState<IMBot | null>(null);
  const [form] = Form.useForm();
  const { t, i18n } = useTranslation();

  // Filters
  const [searchName, setSearchName] = useState('');
  const [botType, setBotType] = useState<string>('');

  const getBotTypeLabel = (type: string) => {
    switch (type) {
      case 'wechat_work': return t('imBots.wecom');
      case 'dingtalk': return t('imBots.dingtalk');
      case 'feishu': return t('imBots.feishu');
      case 'slack': return t('imBots.slack');
      default: return type;
    }
  };

  // Check if the bot type requires a secret
  const needsSecret = (type: string) => {
    // DingTalk and Feishu support signing secret (optional)
    // WeChat Work and Slack don't need it (key is in webhook URL)
    return type === 'dingtalk' || type === 'feishu';
  };

  const getSecretHelpText = (type: string) => {
    const isZh = i18n.language?.startsWith('zh');
    switch (type) {
      case 'dingtalk':
        return isZh ? '钉钉加签密钥（可选，用于安全验证）' : 'DingTalk signing secret (optional, for security)';
      case 'feishu':
        return isZh ? '飞书签名密钥（可选，用于安全验证）' : 'Feishu signing secret (optional, for security)';
      default:
        return '';
    }
  };

  const getWebhookPlaceholder = (type: string) => {
    switch (type) {
      case 'wechat_work':
        return 'https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx';
      case 'dingtalk':
        return 'https://oapi.dingtalk.com/robot/send?access_token=xxx';
      case 'feishu':
        return 'https://open.feishu.cn/open-apis/bot/v2/hook/xxx';
      case 'slack':
        return 'https://hooks.slack.com/services/xxx/xxx/xxx';
      default:
        return 'https://...';
    }
  };

  const fetchData = async () => {
    setLoading(true);
    try {
      const params: any = { page, page_size: pageSize };
      if (searchName) params.name = searchName;
      if (botType) params.type = botType;
      const res = await imBotApi.list(params);
      setData(res.data.items);
      setTotal(res.data.total);
    } catch (error) {
      message.error(t('common.error'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, [page, pageSize]);

  const handleSearch = () => {
    setPage(1);
    fetchData();
  };

  const handleReset = () => {
    setSearchName('');
    setBotType('');
    setPage(1);
    setTimeout(fetchData, 0);
  };

  const showCreateModal = () => {
    setCurrentBot(null);
    form.resetFields();
    form.setFieldsValue({
      type: 'wechat_work',
      is_active: true,
    });
    setModalVisible(true);
  };

  const showEditModal = (record: IMBot) => {
    setCurrentBot(record);
    form.setFieldsValue(record);
    setModalVisible(true);
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (currentBot) {
        await imBotApi.update(currentBot.id, values);
        message.success(t('imBots.updateSuccess'));
      } else {
        await imBotApi.create(values);
        message.success(t('imBots.createSuccess'));
      }
      setModalVisible(false);
      fetchData();
    } catch (error: any) {
      message.error(error.response?.data?.error || t('common.error'));
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await imBotApi.delete(id);
      message.success(t('imBots.deleteSuccess'));
      fetchData();
    } catch (error) {
      message.error(t('common.error'));
    }
  };

  const columns: ColumnsType<IMBot> = [
    {
      title: t('imBots.botName'),
      dataIndex: 'name',
      key: 'name',
      width: 180,
    },
    {
      title: t('imBots.botType'),
      dataIndex: 'type',
      key: 'type',
      width: 120,
      render: (val: string) => <Tag color="blue">{getBotTypeLabel(val)}</Tag>,
    },
    {
      title: t('imBots.webhook'),
      dataIndex: 'webhook',
      key: 'webhook',
      ellipsis: true,
    },
    {
      title: t('imBots.isActive'),
      key: 'is_active',
      width: 100,
      render: (_, record) => (
        <Tag color={record.is_active ? 'success' : 'default'}>
          {record.is_active ? t('common.enabled') : t('common.disabled')}
        </Tag>
      ),
    },
    {
      title: t('common.createdAt'),
      dataIndex: 'created_at',
      key: 'created_at',
      width: 160,
      render: (val: string) => dayjs(val).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: t('common.actions'),
      key: 'action',
      width: 120,
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
          <Input
            placeholder={t('imBots.botName')}
            style={{ width: 180 }}
            value={searchName}
            onChange={(e) => setSearchName(e.target.value)}
            onPressEnter={handleSearch}
          />
          <Select
            placeholder={t('imBots.botType')}
            allowClear
            style={{ width: 120 }}
            value={botType || undefined}
            onChange={setBotType}
            options={[
              { value: 'wechat_work', label: t('imBots.wecom') },
              { value: 'dingtalk', label: t('imBots.dingtalk') },
              { value: 'feishu', label: t('imBots.feishu') },
              { value: 'slack', label: t('imBots.slack') },
            ]}
          />
          <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
            {t('common.search')}
          </Button>
          <Button icon={<ReloadOutlined />} onClick={handleReset}>
            {t('common.reset')}
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={showCreateModal}>
            {t('imBots.createBot')}
          </Button>
        </Space>

        <Table
          columns={columns}
          dataSource={data}
          rowKey="id"
          loading={loading}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            showTotal: (total) => `${t('common.total')} ${total}`,
            onChange: (p, ps) => {
              setPage(p);
              setPageSize(ps);
            },
          }}
        />
      </Card>

      <Modal
        title={currentBot ? t('imBots.editBot') : t('imBots.createBot')}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => setModalVisible(false)}
        width={520}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label={t('imBots.botName')} rules={[{ required: true, message: t('imBots.pleaseInputName') }]}>
            <Input placeholder="AI Code Review Bot" />
          </Form.Item>
          <Form.Item name="type" label={t('imBots.botType')} rules={[{ required: true, message: t('imBots.pleaseSelectType') }]}>
            <Select options={[
              { value: 'wechat_work', label: t('imBots.wecom') },
              { value: 'dingtalk', label: t('imBots.dingtalk') },
              { value: 'feishu', label: t('imBots.feishu') },
              { value: 'slack', label: t('imBots.slack') },
            ]} />
          </Form.Item>
          <Form.Item
            noStyle
            shouldUpdate={(prev, cur) => prev.type !== cur.type}
          >
            {({ getFieldValue }) => (
              <Form.Item 
                name="webhook" 
                label={t('imBots.webhook')} 
                rules={[{ required: true, message: t('imBots.pleaseInputWebhook') }]}
              >
                <Input placeholder={getWebhookPlaceholder(getFieldValue('type'))} />
              </Form.Item>
            )}
          </Form.Item>
          <Form.Item
            noStyle
            shouldUpdate={(prev, cur) => prev.type !== cur.type}
          >
            {({ getFieldValue }) => {
              const type = getFieldValue('type');
              if (!needsSecret(type)) return null;
              return (
                <Form.Item 
                  name="secret" 
                  label={t('imBots.secret')}
                  extra={getSecretHelpText(type)}
                >
                  <Input.Password placeholder={t('imBots.secretPlaceholder', 'SEC...')} />
                </Form.Item>
              );
            }}
          </Form.Item>
          <Form.Item name="is_active" label={t('imBots.isActive')} valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

export default IMBots;
