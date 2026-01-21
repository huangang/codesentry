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
  InputNumber,
  Slider,
} from 'antd';
import { PlusOutlined, SearchOutlined, ReloadOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import { llmConfigApi } from '../services';
import type { LLMConfig } from '../types';

const LLMModels: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<LLMConfig[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [modalVisible, setModalVisible] = useState(false);
  const [currentConfig, setCurrentConfig] = useState<LLMConfig | null>(null);
  const [form] = Form.useForm();
  const { t } = useTranslation();

  // Filters
  const [searchName, setSearchName] = useState('');
  const [provider, setProvider] = useState<string>('');

  const fetchData = async () => {
    setLoading(true);
    try {
      const params: any = { page, page_size: pageSize };
      if (searchName) params.name = searchName;
      if (provider) params.provider = provider;
      const res = await llmConfigApi.list(params);
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
    setProvider('');
    setPage(1);
    setTimeout(fetchData, 0);
  };

  const showCreateModal = () => {
    setCurrentConfig(null);
    form.resetFields();
    form.setFieldsValue({
      provider: 'openai',
      max_tokens: 4096,
      temperature: 0.3,
      is_active: true,
      is_default: false,
    });
    setModalVisible(true);
  };

  const showEditModal = (record: LLMConfig) => {
    setCurrentConfig(record);
    form.setFieldsValue({
      ...record,
      api_key: '', // Don't show actual key
    });
    setModalVisible(true);
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      // Remove empty api_key for update
      if (currentConfig && !values.api_key) {
        delete values.api_key;
      }
      
      if (currentConfig) {
        await llmConfigApi.update(currentConfig.id, values);
        message.success(t('llmModels.updateSuccess'));
      } else {
        await llmConfigApi.create(values);
        message.success(t('llmModels.createSuccess'));
      }
      setModalVisible(false);
      fetchData();
    } catch (error: any) {
      message.error(error.response?.data?.error || t('common.error'));
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await llmConfigApi.delete(id);
      message.success(t('llmModels.deleteSuccess'));
      fetchData();
    } catch (error) {
      message.error(t('common.error'));
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
            options={[
              { value: 'openai', label: t('llmModels.openai') },
              { value: 'azure', label: t('llmModels.azure') },
              { value: 'anthropic', label: 'Anthropic' },
              { value: 'other', label: t('llmModels.custom') },
            ]}
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
        title={currentConfig ? t('llmModels.editModel') : t('llmModels.createModel')}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => setModalVisible(false)}
        width={560}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label={t('llmModels.modelName')} rules={[{ required: true, message: t('llmModels.pleaseInputName') }]}>
            <Input placeholder="GPT-4 Turbo" />
          </Form.Item>
          <Form.Item name="provider" label={t('llmModels.provider')} rules={[{ required: true }]}>
            <Select options={[
              { value: 'openai', label: t('llmModels.openai') },
              { value: 'azure', label: t('llmModels.azure') },
              { value: 'anthropic', label: 'Anthropic' },
              { value: 'other', label: t('llmModels.custom') },
            ]} />
          </Form.Item>
          <Form.Item name="base_url" label={t('llmModels.baseUrl')} rules={[{ required: true, message: t('llmModels.pleaseInputBaseUrl') }]}>
            <Input placeholder="https://api.openai.com/v1" />
          </Form.Item>
          <Form.Item 
            name="api_key" 
            label={t('llmModels.apiKey')} 
            rules={[{ required: !currentConfig }]}
            extra={currentConfig ? t('llmModels.keepExistingKey', 'Leave empty to keep existing key') : undefined}
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
