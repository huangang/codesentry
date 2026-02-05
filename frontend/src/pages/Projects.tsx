import React, { useState, useEffect, useRef } from 'react';
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
  Radio,
  Divider,
  InputNumber,
  DatePicker,
} from 'antd';
import {
  PlusOutlined,
  SearchOutlined,
  ReloadOutlined,
  EditOutlined,
  DeleteOutlined,
  SettingOutlined,
  CopyOutlined,
  UploadOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import type { Project } from '../types';
import { useModal, usePermission, getResponsiveWidth } from '../hooks';
import {
  useProjects,
  useCreateProject,
  useUpdateProject,
  useDeleteProject,
  useDefaultPrompt,
  useActiveImBots,
  useActivePromptTemplates,
  useActiveLLMConfigs,
  type ProjectFilters,
} from '../hooks/queries';
import { PLATFORMS } from '../constants';
import { reviewLogApi } from '../services';

const { TextArea } = Input;

type PromptMode = 'default' | 'template' | 'custom';

const Projects: React.FC = () => {
  const { t, i18n } = useTranslation();
  const [form] = Form.useForm();
  const [promptForm] = Form.useForm();
  const { isAdmin } = usePermission();

  // Query hooks for data fetching
  const [filters, setFilters] = useState<ProjectFilters>({ page: 1, page_size: 10 });
  const [searchName, setSearchName] = useState('');

  const { data: projectsData, isLoading } = useProjects(filters);
  const { data: imBots = [] } = useActiveImBots();
  const { data: promptTemplates = [] } = useActivePromptTemplates();
  const { data: llmConfigs = [] } = useActiveLLMConfigs();
  const { data: defaultPrompt = '' } = useDefaultPrompt();

  // Mutations
  const createProject = useCreateProject();
  const updateProject = useUpdateProject();
  const deleteProject = useDeleteProject();

  const modal = useModal<Project>();
  const [promptDrawerVisible, setPromptDrawerVisible] = useState(false);
  const [currentProjectForPrompt, setCurrentProjectForPrompt] = useState<Project | null>(null);
  const [manualModalVisible, setManualModalVisible] = useState(false);
  const [manualProjectId, setManualProjectId] = useState<number | null>(null);
  const [manualForm] = Form.useForm();
  const [manualLoading, setManualLoading] = useState(false);

  // SSE subscription for import events
  const eventSourceRef = useRef<EventSource | null>(null);

  useEffect(() => {
    const token = localStorage.getItem('token');
    if (!token) return;

    const eventSource = new EventSource(`/api/events/imports?token=${token}`);
    eventSourceRef.current = eventSource;

    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.error) {
          message.error(t('projects.importFailed', 'Import failed for {{name}}: {{error}}')
            .replace('{{name}}', data.project_name)
            .replace('{{error}}', data.error));
        } else {
          message.success(t('projects.importSuccess', 'Imported {{imported}} commits, skipped {{skipped}} existing')
            .replace('{{imported}}', String(data.imported))
            .replace('{{skipped}}', String(data.skipped)));
        }
      } catch {
        // Ignore parse errors
      }
    };

    eventSource.onerror = () => {
      // Silently reconnect on error
    };

    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
    };
  }, [t]);

  const getWebhookUrl = (record: Project) => {
    const baseUrl = window.location.origin;
    return `${baseUrl}/api/webhook/${record.platform}/${record.id}`;
  };

  const handleSearch = () => {
    setFilters(prev => ({ ...prev, page: 1, name: searchName || undefined }));
  };

  const handleReset = () => {
    setSearchName('');
    setFilters({ page: 1, page_size: 10 });
  };

  const handlePageChange = (page: number, pageSize: number) => {
    setFilters(prev => ({ ...prev, page, page_size: pageSize }));
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
      min_score: 0,
    });
  };

  const showEditModal = (record: Project) => {
    modal.open(record);
    form.setFieldsValue(record);
  };

  const getPromptMode = (project: Project): PromptMode => {
    if (project.ai_prompt && project.ai_prompt.length > 0) return 'custom';
    if (project.ai_prompt_id !== null && project.ai_prompt_id !== undefined) return 'template';
    return 'default';
  };

  const showPromptDrawer = (record: Project) => {
    setCurrentProjectForPrompt(record);
    const mode = getPromptMode(record);
    promptForm.setFieldsValue({
      prompt_mode: mode,
      ai_prompt_id: record.ai_prompt_id,
      ai_prompt: record.ai_prompt || defaultPrompt,
    });
    setPromptDrawerVisible(true);
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (modal.current) {
        await updateProject.mutateAsync({ id: modal.current.id, data: values });
        message.success(t('projects.updateSuccess'));
      } else {
        await createProject.mutateAsync(values);
        message.success(t('projects.createSuccess'));
      }
      modal.close();
    } catch (error: any) {
      message.error(error.response?.data?.error || t('common.error'));
    }
  };

  const handlePromptSubmit = async () => {
    try {
      const values = await promptForm.validateFields();
      if (currentProjectForPrompt) {
        const updateData: Partial<Project> = {};

        switch (values.prompt_mode) {
          case 'default':
            updateData.ai_prompt = '';
            updateData.ai_prompt_id = null;
            break;
          case 'template':
            updateData.ai_prompt = '';
            updateData.ai_prompt_id = values.ai_prompt_id;
            break;
          case 'custom':
            updateData.ai_prompt = values.ai_prompt;
            updateData.ai_prompt_id = null;
            break;
        }

        await updateProject.mutateAsync({ id: currentProjectForPrompt.id, data: updateData });
        message.success(t('common.success'));
        setPromptDrawerVisible(false);
      }
    } catch (error: any) {
      message.error(error.response?.data?.error || t('common.error'));
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await deleteProject.mutateAsync(id);
      message.success(t('projects.deleteSuccess'));
    } catch (error) {
      message.error(t('common.error'));
    }
  };

  const copyWebhookUrl = (record: Project) => {
    const url = getWebhookUrl(record);
    navigator.clipboard.writeText(url);
    message.success(t('common.copied') + ': ' + url);
  };

  const showManualModal = (projectId: number) => {
    setManualProjectId(projectId);
    setManualModalVisible(true);
    manualForm.resetFields();
  };

  const handleManualSubmit = async () => {
    if (!manualProjectId) return;
    try {
      const values = await manualForm.validateFields();
      if (!values.date_range || values.date_range.length !== 2) {
        message.error(t('projects.pleaseSelectDateRange', 'Please select date range'));
        return;
      }
      setManualLoading(true);
      await reviewLogApi.importCommits({
        project_id: manualProjectId,
        start_date: values.date_range[0].format('YYYY-MM-DD'),
        end_date: values.date_range[1].format('YYYY-MM-DD'),
      });
      setManualModalVisible(false);
      manualForm.resetFields();
      message.success(t('projects.importStarted', 'Import task started, you will be notified when complete'), 5);
    } catch (error: any) {
      message.error(error.response?.data?.error || t('common.error'));
    } finally {
      setManualLoading(false);
    }
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
        <Tag color={platform === PLATFORMS.GITHUB ? 'geekblue' : platform === PLATFORMS.BITBUCKET ? 'cyan' : 'orange'}>
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
      title: t('projects.minScore', 'Min Score'),
      dataIndex: 'min_score',
      key: 'min_score',
      width: 100,
      render: (score: number) => score > 0 ? score : t('common.default', 'Default'),
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
          {isAdmin && (
            <Tooltip title={t('common.edit')}>
              <Button type="link" size="small" icon={<EditOutlined />} onClick={() => showEditModal(record)} />
            </Tooltip>
          )}
          {isAdmin && (
            <Tooltip title={t('projects.aiPrompt')}>
              <Button type="link" size="small" icon={<SettingOutlined />} onClick={() => showPromptDrawer(record)} />
            </Tooltip>
          )}
          <Tooltip title={t('projects.copyWebhookUrl')}>
            <Button type="link" size="small" icon={<CopyOutlined />} onClick={() => copyWebhookUrl(record)} />
          </Tooltip>
          {isAdmin && (
            <Tooltip title={t('projects.importCommits', 'Import Commits')}>
              <Button type="link" size="small" icon={<UploadOutlined />} onClick={() => showManualModal(record.id)} />
            </Tooltip>
          )}
          {isAdmin && (
            <Popconfirm title={t('projects.deleteConfirm')} onConfirm={() => handleDelete(record.id)}>
              <Button type="link" size="small" danger icon={<DeleteOutlined />} />
            </Popconfirm>
          )}
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
          {isAdmin && (
            <Button type="primary" icon={<PlusOutlined />} onClick={showCreateModal}>
              {t('projects.createProject')}
            </Button>
          )}
        </Space>

        <Table
          columns={columns}
          dataSource={projectsData?.items ?? []}
          rowKey="id"
          loading={isLoading}
          scroll={{ x: 1000 }}
          pagination={{
            current: filters.page,
            pageSize: filters.page_size,
            total: projectsData?.total ?? 0,
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
        confirmLoading={createProject.isPending || updateProject.isPending}
        width={getResponsiveWidth(640)}
        styles={{ body: { maxHeight: '70vh', overflowY: 'auto' } }}
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
              { value: PLATFORMS.BITBUCKET, label: 'Bitbucket' },
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
          <Form.Item
            name="branch_filter"
            label={t('projects.branchFilter')}
            extra={i18n.language?.startsWith('zh') ? '忽略的分支，逗号分隔。支持通配符（如：main,master,release/*）' : 'Branches to ignore, comma-separated. Supports wildcards (e.g., main,master,release/*)'}
          >
            <Input placeholder="main,master,release/*" />
          </Form.Item>
          <Form.Item name="ai_enabled" label={t('projects.aiEnabled')} valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item
            name="llm_config_id"
            label={t('projects.llmConfig', 'LLM Model')}
            extra={i18n.language?.startsWith('zh') ? '选择用于此项目的 AI 模型（不选则使用默认模型）' : 'Select AI model for this project (uses default if not set)'}
          >
            <Select
              allowClear
              placeholder={t('projects.selectLLM', 'Select LLM Model')}
              options={llmConfigs.map(c => ({
                value: c.id,
                label: `${c.name} (${c.model})${c.is_default ? ' ★' : ''}`
              }))}
            />
          </Form.Item>
          <Form.Item
            name="min_score"
            label={t('projects.minScore', 'Min Score')}
            extra={i18n.language?.startsWith('zh') ? 'CI 流水线阻塞阈值（0 表示使用系统默认值）' : 'CI blocking threshold (0 means use system default)'}
          >
            <InputNumber min={0} max={100} style={{ width: '100%' }} />
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
        width={getResponsiveWidth(640)}
        open={promptDrawerVisible}
        onClose={() => setPromptDrawerVisible(false)}
        extra={
          <Button type="primary" onClick={handlePromptSubmit} loading={updateProject.isPending}>{t('common.save')}</Button>
        }
      >
        <Form form={promptForm} layout="vertical">
          <Form.Item
            name="prompt_mode"
            label={t('projects.promptMode', 'Prompt Mode')}
          >
            <Radio.Group>
              <Radio.Button value="default">{t('projects.useSystemDefault', 'System Default')}</Radio.Button>
              <Radio.Button value="template">{t('projects.useTemplate', 'Use Template')}</Radio.Button>
              <Radio.Button value="custom">{t('projects.customPrompt', 'Custom')}</Radio.Button>
            </Radio.Group>
          </Form.Item>

          <Form.Item
            noStyle
            shouldUpdate={(prev, cur) => prev.prompt_mode !== cur.prompt_mode}
          >
            {({ getFieldValue }) => {
              const mode = getFieldValue('prompt_mode');

              const scoringHint = (
                <div style={{ padding: 12, background: '#fffbe6', border: '1px solid #ffe58f', borderRadius: 4, marginBottom: 16 }}>
                  <p style={{ margin: 0, fontSize: 12, color: '#ad8b00' }}>
                    {i18n.language?.startsWith('zh')
                      ? '💡 提示：如果您的提示词没有包含打分指令（如"总分"、"评分"、"score"等），系统将自动追加打分要求。'
                      : '💡 Hint: If your prompt doesn\'t include scoring instructions (like "total score", "rating", etc.), the system will auto-append scoring requirements.'}
                  </p>
                </div>
              );

              if (mode === 'template') {
                return (
                  <>
                    {scoringHint}
                    <Form.Item
                      name="ai_prompt_id"
                      label={t('projects.selectTemplate', 'Select Template')}
                      rules={[{ required: true, message: t('projects.pleaseSelectTemplate', 'Please select a template') }]}
                    >
                      <Select
                        placeholder={t('projects.selectTemplate', 'Select Template')}
                        options={promptTemplates.map(p => ({
                          value: p.id,
                          label: `${p.name}${p.is_default ? ' ★' : ''}${p.is_system ? ` (${t('prompts.system')})` : ''}`
                        }))}
                      />
                    </Form.Item>
                  </>
                );
              }

              if (mode === 'custom') {
                return (
                  <>
                    {scoringHint}
                    <Form.Item
                      name="ai_prompt"
                      label={t('projects.customPrompt', 'Custom Prompt')}
                      rules={[{ required: true, message: t('projects.pleaseInputPrompt', 'Please input prompt') }]}
                      extra={i18n.language?.startsWith('zh') ? '使用 {{diffs}} 和 {{commits}} 作为占位符' : 'Use {{diffs}} and {{commits}} as placeholders'}
                    >
                      <TextArea
                        rows={20}
                        placeholder="{{diffs}}, {{commits}}"
                      />
                    </Form.Item>
                  </>
                );
              }

              return (
                <div style={{ padding: 16, background: '#f5f5f5', borderRadius: 4 }}>
                  <p style={{ margin: 0, color: '#666' }}>
                    {i18n.language?.startsWith('zh')
                      ? '使用系统默认提示词。您可以在「提示词模板」页面中修改默认提示词。'
                      : 'Using system default prompt. You can modify the default prompt in the "Prompt Templates" page.'}
                  </p>
                </div>
              );
            }}
          </Form.Item>

          <Divider />

          <div style={{ fontSize: 12, color: '#999' }}>
            <p><strong>{t('projects.promptPriority', 'Prompt Priority')}:</strong></p>
            <ol style={{ paddingLeft: 20, margin: 0 }}>
              <li>{i18n.language?.startsWith('zh') ? '项目自定义提示词' : 'Project custom prompt'}</li>
              <li>{i18n.language?.startsWith('zh') ? '项目关联的模板' : 'Project linked template'}</li>
              <li>{i18n.language?.startsWith('zh') ? '系统默认提示词' : 'System default prompt'}</li>
            </ol>
          </div>
        </Form>
      </Drawer>

      <Modal
        title={t('projects.importCommits', 'Import Commits')}
        open={manualModalVisible}
        onOk={handleManualSubmit}
        onCancel={() => setManualModalVisible(false)}
        confirmLoading={manualLoading}
        width={getResponsiveWidth(450)}
      >
        <Form form={manualForm} layout="vertical">
          <Form.Item 
            name="date_range" 
            label={t('projects.dateRange', 'Date Range')} 
            rules={[{ required: true, message: t('projects.pleaseSelectDateRange', 'Please select date range') }]}
            extra={i18n.language?.startsWith('zh') 
              ? '系统将自动从 Git 平台拉取该时间范围内的所有提交' 
              : 'The system will fetch all commits within this date range from the Git platform'}
          >
            <DatePicker.RangePicker style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

export default Projects;
