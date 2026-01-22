import React, { useState, useEffect, useCallback } from 'react';
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
  Drawer,
  Tooltip,
} from 'antd';
import {
  PlusOutlined,
  SearchOutlined,
  ReloadOutlined,
  EditOutlined,
  DeleteOutlined,
  SettingOutlined,
  CopyOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import { projectApi, imBotApi } from '../services';
import type { Project, IMBot } from '../types';
import { usePaginatedList, useModal } from '../hooks';
import { PLATFORMS } from '../constants';

const { TextArea } = Input;

interface ProjectFilters {
  name?: string;
}

const Projects: React.FC = () => {
  const { t, i18n } = useTranslation();
  const [form] = Form.useForm();
  const [promptForm] = Form.useForm();

  const [imBots, setImBots] = useState<IMBot[]>([]);
  const [defaultPrompt, setDefaultPrompt] = useState('');
  const [searchName, setSearchName] = useState('');

  const modal = useModal<Project>();
  const [promptDrawerVisible, setPromptDrawerVisible] = useState(false);
  const [currentProjectForPrompt, setCurrentProjectForPrompt] = useState<Project | null>(null);

  const {
    loading,
    data,
    total,
    page,
    pageSize,
    setPage,
    fetchData,
    handlePageChange,
  } = usePaginatedList<Project, ProjectFilters>({
    fetchApi: projectApi.list,
    onError: () => message.error(t('common.error')),
  });

  const getWebhookUrl = (record: Project) => {
    const baseUrl = window.location.origin;
    return `${baseUrl}/api/webhook/${record.platform}/${record.id}`;
  };

  const fetchImBots = useCallback(async () => {
    try {
      const res = await imBotApi.getActive();
      setImBots(res.data);
    } catch (error) {
      console.error('Failed to fetch IM bots:', error);
    }
  }, []);

  const fetchDefaultPrompt = useCallback(async () => {
    try {
      const res = await projectApi.getDefaultPrompt();
      setDefaultPrompt(res.data.prompt);
    } catch (error) {
      console.error('Failed to fetch default prompt:', error);
    }
  }, []);

  useEffect(() => {
    fetchData({ name: searchName || undefined });
  }, [page, pageSize]);

  useEffect(() => {
    fetchImBots();
    fetchDefaultPrompt();
  }, [fetchImBots, fetchDefaultPrompt]);

  const handleSearch = () => {
    setPage(1);
    fetchData({ name: searchName || undefined });
  };

  const handleReset = () => {
    setSearchName('');
    setPage(1);
    fetchData({});
  };

  const showCreateModal = () => {
    modal.open();
    form.resetFields();
    form.setFieldsValue({
      platform: PLATFORMS.GITLAB,
      ai_enabled: true,
      im_enabled: false,
      file_extensions: '.go,.js,.ts,.jsx,.tsx,.py,.java,.c,.cpp,.h,.hpp,.cs,.rb,.php,.swift,.kt,.rs,.vue,.svelte',
      review_events: 'push,merge_request',
    });
  };

  const showEditModal = (record: Project) => {
    modal.open(record);
    form.setFieldsValue(record);
  };

  const showPromptDrawer = (record: Project) => {
    setCurrentProjectForPrompt(record);
    promptForm.setFieldsValue({
      ai_prompt: record.ai_prompt || defaultPrompt,
      use_default: !record.ai_prompt,
    });
    setPromptDrawerVisible(true);
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (modal.current) {
        await projectApi.update(modal.current.id, values);
        message.success(t('projects.updateSuccess'));
      } else {
        await projectApi.create(values);
        message.success(t('projects.createSuccess'));
      }
      modal.close();
      fetchData({ name: searchName || undefined });
    } catch (error: any) {
      message.error(error.response?.data?.error || t('common.error'));
    }
  };

  const handlePromptSubmit = async () => {
    try {
      const values = await promptForm.validateFields();
      if (currentProjectForPrompt) {
        await projectApi.update(currentProjectForPrompt.id, {
          ai_prompt: values.use_default ? '' : values.ai_prompt,
        });
        message.success(t('common.success'));
        setPromptDrawerVisible(false);
        fetchData({ name: searchName || undefined });
      }
    } catch (error: any) {
      message.error(error.response?.data?.error || t('common.error'));
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await projectApi.delete(id);
      message.success(t('projects.deleteSuccess'));
      fetchData({ name: searchName || undefined });
    } catch (error) {
      message.error(t('common.error'));
    }
  };

  const copyWebhookUrl = (record: Project) => {
    const url = getWebhookUrl(record);
    navigator.clipboard.writeText(url);
    message.success(t('common.copied') + ': ' + url);
  };

  const columns: ColumnsType<Project> = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 60,
    },
    {
      title: t('projects.projectName'),
      dataIndex: 'name',
      key: 'name',
      width: 150,
      ellipsis: true,
    },
    {
      title: t('projects.platform'),
      dataIndex: 'platform',
      key: 'platform',
      width: 100,
      render: (platform: string) => (
        <Tag color={platform === PLATFORMS.GITHUB ? 'geekblue' : 'orange'}>
          {platform.toUpperCase()}
        </Tag>
      ),
    },
    {
      title: t('projects.aiEnabled'),
      key: 'ai_enabled',
      width: 80,
      render: (_, record) => (
        <Tag color={record.ai_enabled ? 'success' : 'default'}>
          {record.ai_enabled ? t('common.yes') : t('common.no')}
        </Tag>
      ),
    },
    {
      title: t('projects.imEnabled'),
      key: 'im_enabled',
      width: 80,
      render: (_, record) => (
        <Tag color={record.im_enabled ? 'success' : 'default'}>
          {record.im_enabled ? t('common.yes') : t('common.no')}
        </Tag>
      ),
    },
    {
      title: t('common.actions'),
      key: 'action',
      width: 160,
      render: (_, record) => (
        <Space>
          <Tooltip title={t('common.edit')}>
            <Button type="link" size="small" icon={<EditOutlined />} onClick={() => showEditModal(record)} />
          </Tooltip>
          <Tooltip title={t('projects.aiPrompt')}>
            <Button type="link" size="small" icon={<SettingOutlined />} onClick={() => showPromptDrawer(record)} />
          </Tooltip>
          <Tooltip title={t('projects.copyWebhookUrl')}>
            <Button type="link" size="small" icon={<CopyOutlined />} onClick={() => copyWebhookUrl(record)} />
          </Tooltip>
          <Popconfirm title={t('projects.deleteConfirm')} onConfirm={() => handleDelete(record.id)}>
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
            placeholder={t('projects.projectName')}
            style={{ width: 200 }}
            value={searchName}
            onChange={(e) => setSearchName(e.target.value)}
            onPressEnter={handleSearch}
          />
          <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
            {t('common.search')}
          </Button>
          <Button icon={<ReloadOutlined />} onClick={handleReset}>
            {t('common.reset')}
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={showCreateModal}>
            {t('projects.createProject')}
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
        title={modal.isEdit ? t('projects.editProject') : t('projects.createProject')}
        open={modal.visible}
        onOk={handleSubmit}
        onCancel={modal.close}
        width={640}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label={t('projects.projectName')} rules={[{ required: true, message: t('projects.pleaseInputName') }]}>
            <Input placeholder={t('projects.pleaseInputName')} />
          </Form.Item>
          <Form.Item name="url" label={t('projects.projectUrl')} rules={[{ required: true, message: t('projects.pleaseInputUrl') }]}>
            <Input placeholder="https://github.com/user/repo" />
          </Form.Item>
          <Form.Item name="platform" label={t('projects.platform')} rules={[{ required: true, message: t('projects.pleaseSelectPlatform') }]}>
            <Select options={[
              { value: PLATFORMS.GITHUB, label: 'GitHub' },
              { value: PLATFORMS.GITLAB, label: 'GitLab' },
            ]} />
          </Form.Item>
          <Form.Item 
            name="access_token" 
            label={t('projects.accessToken')}
            extra={i18n.language?.startsWith('zh') ? '用于获取代码差异，需要有仓库读取权限' : 'Used to fetch code diff, requires repo read access'}
          >
            <Input.Password placeholder={t('projects.accessToken')} />
          </Form.Item>
          <Form.Item 
            name="webhook_secret" 
            label={t('projects.webhookSecret')}
            extra={i18n.language?.startsWith('zh') ? '用于验证 Webhook 请求签名（可选）' : 'Used to verify webhook signature (optional)'}
          >
            <Input.Password placeholder={t('projects.webhookSecret')} />
          </Form.Item>
          <Form.Item name="file_extensions" label={t('projects.fileExtensions')}>
            <Input placeholder={t('projects.fileExtensionsPlaceholder')} />
          </Form.Item>
          <Form.Item 
            name="ignore_patterns" 
            label={t('projects.ignorePatterns')}
            extra={i18n.language?.startsWith('zh') ? '忽略的文件路径，逗号分隔（如：vendor/,node_modules/,*.min.js）' : 'File paths to ignore, comma-separated (e.g., vendor/,node_modules/,*.min.js)'}
          >
            <Input placeholder="vendor/,node_modules/,*.min.js,*.lock" />
          </Form.Item>
          <Form.Item name="review_events" label={t('projects.reviewEvents')}>
            <Input placeholder="push,merge_request" />
          </Form.Item>
          <Form.Item name="ai_enabled" label={t('projects.aiEnabled')} valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item 
            name="comment_enabled" 
            label={t('projects.commentEnabled')} 
            valuePropName="checked"
            extra={i18n.language?.startsWith('zh') ? '审查完成后自动评论到 MR/PR' : 'Auto-comment review result to MR/PR'}
          >
            <Switch />
          </Form.Item>
          <Form.Item name="im_enabled" label={t('projects.imEnabled')} valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="im_bot_id" label={t('projects.imBot')}>
            <Select
              allowClear
              placeholder={t('projects.imBot')}
              options={imBots.map(bot => ({ value: bot.id, label: `${bot.name} (${bot.type})` }))}
            />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer
        title={t('projects.aiPrompt')}
        width={640}
        open={promptDrawerVisible}
        onClose={() => setPromptDrawerVisible(false)}
        extra={
          <Button type="primary" onClick={handlePromptSubmit}>{t('common.save')}</Button>
        }
      >
        <Form form={promptForm} layout="vertical">
          <Form.Item name="use_default" label={t('projects.useDefaultPrompt', 'Use Default Prompt')} valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item
            noStyle
            shouldUpdate={(prev, cur) => prev.use_default !== cur.use_default}
          >
            {({ getFieldValue }) => (
              <Form.Item name="ai_prompt" label={t('projects.customPrompt', 'Custom Prompt')}>
                <TextArea
                  rows={20}
                  disabled={getFieldValue('use_default')}
                  placeholder="{{diffs}}, {{commits}}"
                />
              </Form.Item>
            )}
          </Form.Item>
        </Form>
      </Drawer>
    </>
  );
};

export default Projects;
