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
  Tooltip,
} from 'antd';
import {
  PlusOutlined,
  SearchOutlined,
  ReloadOutlined,
  EditOutlined,
  DeleteOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import { gitCredentialApi } from '../services';
import type { GitCredential } from '../types';
import { usePaginatedList, useModal } from '../hooks';
import { PLATFORMS } from '../constants';

interface GitCredentialFilters {
  name?: string;
  platform?: string;
}

const GitCredentials: React.FC = () => {
  const { t, i18n } = useTranslation();
  const [form] = Form.useForm();

  const [searchName, setSearchName] = useState('');
  const [searchPlatform, setSearchPlatform] = useState<string | undefined>();

  const modal = useModal<GitCredential>();

  const {
    loading,
    data,
    total,
    page,
    pageSize,
    setPage,
    fetchData,
    handlePageChange,
  } = usePaginatedList<GitCredential, GitCredentialFilters>({
    fetchApi: gitCredentialApi.list,
    onError: () => message.error(t('common.error')),
  });

  useEffect(() => {
    fetchData({ name: searchName || undefined, platform: searchPlatform });
  }, [page, pageSize]);

  const handleSearch = () => {
    setPage(1);
    fetchData({ name: searchName || undefined, platform: searchPlatform });
  };

  const handleReset = () => {
    setSearchName('');
    setSearchPlatform(undefined);
    setPage(1);
    fetchData({});
  };

  const showCreateModal = () => {
    modal.open();
    form.resetFields();
    form.setFieldsValue({
      platform: PLATFORMS.GITLAB,
      auto_create: true,
      default_enabled: true,
      is_active: true,
      file_extensions: '.go,.js,.ts,.jsx,.tsx,.py,.java,.c,.cpp,.h,.hpp,.cs,.rb,.php,.swift,.kt,.rs,.vue,.svelte',
      review_events: 'push,merge_request',
    });
  };

  const showEditModal = (record: GitCredential) => {
    modal.open(record);
    form.setFieldsValue({
      ...record,
      access_token: '',
      webhook_secret: '',
    });
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      
      const submitData = { ...values };
      if (!submitData.access_token) {
        delete submitData.access_token;
      }
      if (!submitData.webhook_secret) {
        delete submitData.webhook_secret;
      }

      if (modal.current) {
        await gitCredentialApi.update(modal.current.id, submitData);
        message.success(t('gitCredentials.updateSuccess'));
      } else {
        await gitCredentialApi.create(submitData);
        message.success(t('gitCredentials.createSuccess'));
      }
      modal.close();
      fetchData({ name: searchName || undefined, platform: searchPlatform });
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: string } } };
      message.error(err.response?.data?.error || t('common.error'));
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await gitCredentialApi.delete(id);
      message.success(t('gitCredentials.deleteSuccess'));
      fetchData({ name: searchName || undefined, platform: searchPlatform });
    } catch {
      message.error(t('common.error'));
    }
  };

  const columns: ColumnsType<GitCredential> = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 60,
    },
    {
      title: t('gitCredentials.name'),
      dataIndex: 'name',
      key: 'name',
      width: 150,
      ellipsis: true,
    },
    {
      title: t('gitCredentials.platform'),
      dataIndex: 'platform',
      key: 'platform',
      width: 100,
      render: (platform: string) => (
        <Tag color={platform === PLATFORMS.GITHUB ? 'geekblue' : platform === PLATFORMS.BITBUCKET ? 'cyan' : 'orange'}>
          {platform.toUpperCase()}
        </Tag>
      ),
    },
    {
      title: t('gitCredentials.baseUrl'),
      dataIndex: 'base_url',
      key: 'base_url',
      width: 200,
      ellipsis: true,
      render: (url: string) => url || '-',
    },
    {
      title: t('gitCredentials.autoCreate'),
      key: 'auto_create',
      width: 100,
      render: (_, record) => (
        <Tag color={record.auto_create ? 'success' : 'default'}>
          {record.auto_create ? t('common.yes') : t('common.no')}
        </Tag>
      ),
    },
    {
      title: t('gitCredentials.defaultEnabled'),
      key: 'default_enabled',
      width: 100,
      render: (_, record) => (
        <Tag color={record.default_enabled ? 'success' : 'default'}>
          {record.default_enabled ? t('common.yes') : t('common.no')}
        </Tag>
      ),
    },
    {
      title: t('gitCredentials.isActive'),
      key: 'is_active',
      width: 80,
      render: (_, record) => (
        <Tag color={record.is_active ? 'success' : 'default'}>
          {record.is_active ? t('common.yes') : t('common.no')}
        </Tag>
      ),
    },
    {
      title: t('common.actions'),
      key: 'action',
      width: 120,
      render: (_, record) => (
        <Space>
          <Tooltip title={t('common.edit')}>
            <Button type="link" size="small" icon={<EditOutlined />} onClick={() => showEditModal(record)} />
          </Tooltip>
          <Popconfirm title={t('gitCredentials.deleteConfirm')} onConfirm={() => handleDelete(record.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />} />
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
            placeholder={t('gitCredentials.name')}
            style={{ width: 200 }}
            value={searchName}
            onChange={(e) => setSearchName(e.target.value)}
            onPressEnter={handleSearch}
          />
          <Select
            placeholder={t('gitCredentials.platform')}
            style={{ width: 120 }}
            allowClear
            value={searchPlatform}
            onChange={setSearchPlatform}
            options={[
              { value: PLATFORMS.GITHUB, label: 'GitHub' },
              { value: PLATFORMS.GITLAB, label: 'GitLab' },
              { value: PLATFORMS.BITBUCKET, label: 'Bitbucket' },
            ]}
          />
          <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
            {t('common.search')}
          </Button>
          <Button icon={<ReloadOutlined />} onClick={handleReset}>
            {t('common.reset')}
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={showCreateModal}>
            {t('gitCredentials.createCredential')}
          </Button>
        </Space>

        <Table
          columns={columns}
          dataSource={data}
          rowKey="id"
          loading={loading}
          scroll={{ x: 1000 }}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            showTotal: (total) => `${t('common.total')} ${total}`,
            onChange: handlePageChange,
          }}
        />
      </Card>

      <Modal
        title={modal.isEdit ? t('gitCredentials.editCredential') : t('gitCredentials.createCredential')}
        open={modal.visible}
        onOk={handleSubmit}
        onCancel={modal.close}
        width={640}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label={t('gitCredentials.name')} rules={[{ required: true, message: t('gitCredentials.pleaseInputName') }]}>
            <Input placeholder={t('gitCredentials.pleaseInputName')} />
          </Form.Item>
          <Form.Item name="platform" label={t('gitCredentials.platform')} rules={[{ required: true, message: t('gitCredentials.pleaseSelectPlatform') }]}>
            <Select options={[
              { value: PLATFORMS.GITHUB, label: 'GitHub' },
              { value: PLATFORMS.GITLAB, label: 'GitLab' },
              { value: PLATFORMS.BITBUCKET, label: 'Bitbucket' },
            ]} />
          </Form.Item>
          <Form.Item
            name="base_url"
            label={t('gitCredentials.baseUrl')}
            extra={i18n.language?.startsWith('zh') ? '用于自托管 GitLab/GitHub/Bitbucket 企业版，如 https://gitlab.example.com' : 'For self-hosted GitLab/GitHub/Bitbucket Server, e.g., https://gitlab.example.com'}
          >
            <Input placeholder="https://gitlab.example.com" />
          </Form.Item>
          <Form.Item
            name="access_token"
            label={t('gitCredentials.accessToken')}
            extra={modal.isEdit 
              ? (i18n.language?.startsWith('zh') ? '留空则不修改现有令牌' : 'Leave empty to keep existing token')
              : (i18n.language?.startsWith('zh') ? '用于获取代码差异，需要有仓库读取权限' : 'Used to fetch code diff, requires repo read access')
            }
          >
            <Input.Password placeholder={t('gitCredentials.accessToken')} />
          </Form.Item>
          <Form.Item
            name="webhook_secret"
            label={t('gitCredentials.webhookSecret')}
            extra={modal.isEdit
              ? (i18n.language?.startsWith('zh') ? '留空则不修改现有密钥' : 'Leave empty to keep existing secret')
              : (i18n.language?.startsWith('zh') ? '用于验证 Webhook 请求签名' : 'Used to verify webhook signature')
            }
          >
            <Input.Password placeholder={t('gitCredentials.webhookSecret')} />
          </Form.Item>
          <Form.Item
            name="auto_create"
            label={t('gitCredentials.autoCreate')}
            valuePropName="checked"
            extra={i18n.language?.startsWith('zh') ? '收到 webhook 时自动创建项目进行代码审查' : 'Auto-create projects for code review when webhook received'}
          >
            <Switch />
          </Form.Item>
          <Form.Item
            name="default_enabled"
            label={t('gitCredentials.defaultEnabled')}
            valuePropName="checked"
            extra={i18n.language?.startsWith('zh') ? '自动创建的项目默认启用 AI 审查' : 'Enable AI review by default for auto-created projects'}
          >
            <Switch />
          </Form.Item>
          <Form.Item name="file_extensions" label={t('gitCredentials.fileExtensions')}>
            <Input placeholder=".go,.js,.ts,.jsx,.tsx,.py" />
          </Form.Item>
          <Form.Item name="review_events" label={t('gitCredentials.reviewEvents')}>
            <Input placeholder="push,merge_request" />
          </Form.Item>
          <Form.Item
            name="ignore_patterns"
            label={t('gitCredentials.ignorePatterns')}
            extra={i18n.language?.startsWith('zh') ? '忽略的文件路径，逗号分隔' : 'File paths to ignore, comma-separated'}
          >
            <Input placeholder="vendor/,node_modules/,*.min.js" />
          </Form.Item>
          <Form.Item name="is_active" label={t('gitCredentials.isActive')} valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

export default GitCredentials;
