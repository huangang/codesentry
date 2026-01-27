import React, { useEffect } from 'react';
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
  InputNumber,
  Slider,
} from 'antd';
import { PlusOutlined, SearchOutlined, ReloadOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import { llmConfigApi } from '../services';
import type { LLMConfig } from '../types';
import { usePaginatedList, useModal } from '../hooks';
import { LLM_PROVIDERS } from '../constants';

interface LLMConfigFilters {
  name?: string;
  provider?: string;
}

const LLMModels: React.FC = () => {
  const { t } = useTranslation();
  const [form] = Form.useForm();
  const [searchName, setSearchName] = React.useState('');
  const [provider, setProvider] = React.useState<string>('');
  const [selectedProvider, setSelectedProvider] = React.useState<string>('');

  const modal = useModal<LLMConfig>();

  const {
    loading,
    data,
    total,
    page,
    pageSize,
    setPage,
    fetchData,
    handlePageChange,
  } = usePaginatedList<LLMConfig, LLMConfigFilters>({
    fetchApi: llmConfigApi.list,
    onError: () => message.error(t('common.error')),
  });

  const buildFilters = (): LLMConfigFilters => {
    const filters: LLMConfigFilters = {};
    if (searchName) filters.name = searchName;
    if (provider) filters.provider = provider;
    return filters;
  };

  useEffect(() => {
    fetchData(buildFilters());
  }, [page, pageSize]);

  const handleSearch = () => {
    setPage(1);
    fetchData(buildFilters());
  };

  const handleReset = () => {
    setSearchName('');
    setProvider('');
    setPage(1);
    fetchData({});
  };

  const showCreateModal = () => {
    modal.open();
    form.resetFields();
    form.setFieldsValue({
      provider: LLM_PROVIDERS.OPENAI,
      max_tokens: 4096,
      temperature: 0.3,
      is_active: true,
      is_default: false,
    });
  };

  const showEditModal = (record: LLMConfig) => {
    modal.open(record);
    form.setFieldsValue({
      ...record,
      api_key: '',
    });
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (modal.current && !values.api_key) {
        delete values.api_key;
      }

      if (modal.current) {
        await llmConfigApi.update(modal.current.id, values);
        message.success(t('llmModels.updateSuccess'));
      } else {
        await llmConfigApi.create(values);
        message.success(t('llmModels.createSuccess'));
      }
      modal.close();
      fetchData(buildFilters());
    } catch (error: any) {
      message.error(error.response?.data?.error || t('common.error'));
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await llmConfigApi.delete(id);
      message.success(t('llmModels.deleteSuccess'));
      fetchData(buildFilters());
    } catch (error) {
      message.error(t('common.error'));
    }
  };

  const providerOptions = [
    { value: LLM_PROVIDERS.OPENAI, label: t('llmModels.openai') },
    { value: LLM_PROVIDERS.AZURE, label: t('llmModels.azure') },
    { value: LLM_PROVIDERS.ANTHROPIC, label: 'Anthropic' },
    { value: LLM_PROVIDERS.OLLAMA, label: t('llmModels.ollama') },
    { value: LLM_PROVIDERS.GEMINI, label: t('llmModels.gemini') },
  ];

  const handleProviderChange = (value: string) => {
    setSelectedProvider(value);
    // Native API providers handle endpoints internally via SDK
    // Only set Base URL for providers that need custom endpoints
    switch (value) {
      case LLM_PROVIDERS.OLLAMA:
        // Ollama needs server address (no /v1 suffix for native API)
        form.setFieldsValue({ base_url: 'http://localhost:11434' });
        break;
      case LLM_PROVIDERS.AZURE:
        // Azure needs resource URL
        form.setFieldsValue({ base_url: 'https://your-resource.openai.azure.com' });
        break;
      case LLM_PROVIDERS.OPENAI:
        // OpenAI default endpoint
        form.setFieldsValue({ base_url: 'https://api.openai.com/v1' });
        break;
      default:
        // Anthropic and Gemini use native SDKs, no Base URL needed
        form.setFieldsValue({ base_url: '' });
        break;
    }
  };

  // Get dynamic placeholder and hint for base_url based on provider
  const getBaseUrlConfig = () => {
    switch (selectedProvider) {
      case LLM_PROVIDERS.ANTHROPIC:
        return {
          placeholder: t('llmModels.anthropicBaseUrlPlaceholder', 'Optional - SDK uses https://api.anthropic.com'),
          hint: t('llmModels.anthropicBaseUrlHint', 'Leave empty to use official API (native SDK)'),
        };
      case LLM_PROVIDERS.GEMINI:
        return {
          placeholder: t('llmModels.geminiBaseUrlPlaceholder', 'Optional - SDK uses Google AI API'),
          hint: t('llmModels.geminiBaseUrlHint', 'Leave empty to use official API (native SDK)'),
        };
      case LLM_PROVIDERS.OLLAMA:
        return {
          placeholder: 'http://localhost:11434',
          hint: t('llmModels.ollamaBaseUrlHint', 'Ollama server address'),
        };
      case LLM_PROVIDERS.AZURE:
        return {
          placeholder: 'https://your-resource.openai.azure.com',
          hint: t('llmModels.azureBaseUrlHint', 'Azure OpenAI resource URL'),
        };
      default:
        return {
          placeholder: 'https://api.openai.com/v1',
          hint: t('llmModels.openaiBaseUrlHint', 'OpenAI API endpoint or compatible proxy'),
        };
    }
  };

  const columns: ColumnsType<LLMConfig> = [
    {
      title: t('llmModels.provider'),
      dataIndex: 'provider',
      key: 'provider',
      width: 100,
      render: (val: string) => <Tag color="blue">{val.toUpperCase()}</Tag>,
    },
    {
      title: t('llmModels.model'),
      dataIndex: 'model',
      key: 'model',
      width: 180,
    },
    {
      title: t('llmModels.baseUrl'),
      dataIndex: 'base_url',
      key: 'base_url',
      ellipsis: true,
    },
    {
      title: t('llmModels.maxTokens'),
      dataIndex: 'max_tokens',
      key: 'max_tokens',
      width: 100,
      render: (val: number) => val.toLocaleString(),
    },
    {
      title: t('llmModels.isDefault'),
      key: 'is_default',
      width: 100,
      render: (_, record) => record.is_default ? <Tag color="success">{t('llmModels.isDefault')}</Tag> : null,
    },
    {
      title: t('common.updatedAt'),
      dataIndex: 'updated_at',
      key: 'updated_at',
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
          <Popconfirm title={t('llmModels.deleteConfirm')} onConfirm={() => handleDelete(record.id)}>
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
            placeholder={t('llmModels.modelName')}
            style={{ width: 180 }}
            value={searchName}
            onChange={(e) => setSearchName(e.target.value)}
            onPressEnter={handleSearch}
          />
          <Select
            placeholder={t('llmModels.provider')}
            allowClear
            style={{ width: 120 }}
            value={provider || undefined}
            onChange={setProvider}
            options={providerOptions}
          />
          <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
            {t('common.search')}
          </Button>
          <Button icon={<ReloadOutlined />} onClick={handleReset}>
            {t('common.reset')}
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={showCreateModal}>
            {t('llmModels.createModel')}
          </Button>
        </Space>

        <Table
          columns={columns}
          dataSource={data}
          rowKey="id"
          loading={loading}
          scroll={{ x: 900 }}
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
        title={modal.isEdit ? t('llmModels.editModel') : t('llmModels.createModel')}
        open={modal.visible}
        onOk={handleSubmit}
        onCancel={modal.close}
        width={560}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label={t('llmModels.modelName')} rules={[{ required: true, message: t('llmModels.pleaseInputName') }]}>
            <Input placeholder="GPT-4 Turbo" />
          </Form.Item>
          <Form.Item name="provider" label={t('llmModels.provider')} rules={[{ required: true }]}>
            <Select options={providerOptions} onChange={handleProviderChange} />
          </Form.Item>
          <Form.Item name="base_url" label={t('llmModels.baseUrl')} extra={getBaseUrlConfig().hint}>
            <Input placeholder={getBaseUrlConfig().placeholder} />
          </Form.Item>
          <Form.Item
            name="api_key"
            label={t('llmModels.apiKey')}
            rules={[{ required: !modal.current }]}
            extra={modal.current ? t('llmModels.keepExistingKey', 'Leave empty to keep existing key') : undefined}
          >
            <Input.Password placeholder="sk-..." />
          </Form.Item>
          <Form.Item name="model" label={t('llmModels.model')} rules={[{ required: true, message: t('llmModels.pleaseInputModel') }]}>
            <Input placeholder="gpt-4-turbo-preview" />
          </Form.Item>
          <Form.Item name="max_tokens" label={t('llmModels.maxTokens')}>
            <InputNumber min={100} max={128000} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="temperature" label={t('llmModels.temperature')}>
            <Slider min={0} max={2} step={0.1} />
          </Form.Item>
          <Form.Item name="is_default" label={t('llmModels.setAsDefault')} valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="is_active" label={t('llmModels.isActive')} valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

export default LLMModels;
