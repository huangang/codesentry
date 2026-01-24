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
  Drawer,
  Typography,
  Tooltip,
  Segmented,
} from 'antd';
import {
  PlusOutlined,
  SearchOutlined,
  ReloadOutlined,
  EditOutlined,
  DeleteOutlined,
  EyeOutlined,
  StarOutlined,
  StarFilled,
  CopyOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { promptApi } from '../services';
import type { PromptTemplate } from '../types';
import { usePaginatedList, useModal, usePermission } from '../hooks';

const { TextArea } = Input;
const { Paragraph } = Typography;

interface PromptFilters {
  name?: string;
  is_system?: boolean;
}

const Prompts: React.FC = () => {
  const { t, i18n } = useTranslation();
  const [form] = Form.useForm();
  const [searchName, setSearchName] = React.useState('');
  const [filterType, setFilterType] = React.useState<string>('');
  const [drawerVisible, setDrawerVisible] = React.useState(false);
  const [viewingPrompt, setViewingPrompt] = React.useState<PromptTemplate | null>(null);
  const [viewMode, setViewMode] = React.useState<'rendered' | 'source'>('rendered');
  const { canWrite } = usePermission();

  const modal = useModal<PromptTemplate>();

  const {
    loading,
    data,
    total,
    page,
    pageSize,
    setPage,
    fetchData,
    handlePageChange,
  } = usePaginatedList<PromptTemplate, PromptFilters>({
    fetchApi: promptApi.list,
    onError: () => message.error(t('common.error')),
  });

  const buildFilters = (): PromptFilters => {
    const filters: PromptFilters = {};
    if (searchName) filters.name = searchName;
    if (filterType === 'system') filters.is_system = true;
    if (filterType === 'custom') filters.is_system = false;
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
    setFilterType('');
    setPage(1);
    fetchData({});
  };

  const showCreateModal = () => {
    modal.open();
    form.resetFields();
    form.setFieldsValue({ is_default: false });
  };

  const showEditModal = (record: PromptTemplate) => {
    if (record.is_system) {
      message.warning(t('prompts.cannotEditSystem'));
      return;
    }
    modal.open(record);
    form.setFieldsValue(record);
  };

  const showViewDrawer = (record: PromptTemplate) => {
    setViewingPrompt(record);
    setDrawerVisible(true);
  };

  const handleDuplicate = (record: PromptTemplate) => {
    modal.open();
    form.setFieldsValue({
      name: `${record.name} (Copy)`,
      description: record.description,
      content: record.content,
      is_default: false,
    });
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (modal.current) {
        await promptApi.update(modal.current.id, values);
        message.success(t('prompts.updateSuccess'));
      } else {
        await promptApi.create(values);
        message.success(t('prompts.createSuccess'));
      }
      modal.close();
      fetchData(buildFilters());
    } catch (error: any) {
      message.error(error.response?.data?.error || t('common.error'));
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await promptApi.delete(id);
      message.success(t('prompts.deleteSuccess'));
      fetchData(buildFilters());
    } catch (error: any) {
      message.error(error.response?.data?.error || t('common.error'));
    }
  };

  const handleSetDefault = async (id: number) => {
    try {
      await promptApi.setDefault(id);
      message.success(t('prompts.setDefaultSuccess'));
      fetchData(buildFilters());
    } catch (error: any) {
      message.error(error.response?.data?.error || t('common.error'));
    }
  };

  const columns: ColumnsType<PromptTemplate> = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 60,
    },
    {
      title: t('prompts.name'),
      dataIndex: 'name',
      key: 'name',
      width: 200,
      ellipsis: true,
    },
    {
      title: t('prompts.description'),
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: t('prompts.type'),
      key: 'is_system',
      width: 100,
      render: (_, record) => (
        <Tag color={record.is_system ? 'blue' : 'green'}>
          {record.is_system ? t('prompts.system') : t('prompts.custom')}
        </Tag>
      ),
    },
    {
      title: t('prompts.isDefault'),
      key: 'is_default',
      width: 80,
      render: (_, record) =>
        record.is_default ? (
          <StarFilled style={{ color: '#faad14', fontSize: 16 }} />
        ) : null,
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
      width: 180,
      render: (_, record) => (
        <Space>
          <Tooltip title={t('common.view')}>
            <Button
              type="link"
              size="small"
              icon={<EyeOutlined />}
              onClick={() => showViewDrawer(record)}
            />
          </Tooltip>
          {canWrite && (
            <>
              <Tooltip title={t('prompts.duplicate')}>
                <Button
                  type="link"
                  size="small"
                  icon={<CopyOutlined />}
                  onClick={() => handleDuplicate(record)}
                />
              </Tooltip>
              {!record.is_system && (
                <>
                  <Tooltip title={t('common.edit')}>
                    <Button
                      type="link"
                      size="small"
                      icon={<EditOutlined />}
                      onClick={() => showEditModal(record)}
                    />
                  </Tooltip>
                  <Tooltip title={t('prompts.setAsDefault')}>
                    <Button
                      type="link"
                      size="small"
                      icon={<StarOutlined />}
                      onClick={() => handleSetDefault(record.id)}
                      disabled={record.is_default}
                    />
                  </Tooltip>
                  <Popconfirm
                    title={t('prompts.deleteConfirm')}
                    onConfirm={() => handleDelete(record.id)}
                  >
                    <Button type="link" size="small" danger icon={<DeleteOutlined />} />
                  </Popconfirm>
                </>
              )}
              {record.is_system && (
                <Tooltip title={t('prompts.setAsDefault')}>
                  <Button
                    type="link"
                    size="small"
                    icon={<StarOutlined />}
                    onClick={() => handleSetDefault(record.id)}
                    disabled={record.is_default}
                  />
                </Tooltip>
              )}
            </>
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
            placeholder={t('prompts.name')}
            style={{ width: 180 }}
            value={searchName}
            onChange={(e) => setSearchName(e.target.value)}
            onPressEnter={handleSearch}
          />
          <Select
            placeholder={t('prompts.type')}
            allowClear
            style={{ width: 120 }}
            value={filterType || undefined}
            onChange={setFilterType}
            options={[
              { value: 'system', label: t('prompts.system') },
              { value: 'custom', label: t('prompts.custom') },
            ]}
          />
          <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
            {t('common.search')}
          </Button>
          <Button icon={<ReloadOutlined />} onClick={handleReset}>
            {t('common.reset')}
          </Button>
          {canWrite && (
            <Button type="primary" icon={<PlusOutlined />} onClick={showCreateModal}>
              {t('prompts.createPrompt')}
            </Button>
          )}
        </Space>

        <Table
          columns={columns}
          dataSource={data}
          rowKey="id"
          loading={loading}
          scroll={{ x: 800 }}
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
        title={modal.isEdit ? t('prompts.editPrompt') : t('prompts.createPrompt')}
        open={modal.visible}
        onOk={handleSubmit}
        onCancel={modal.close}
        width={720}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label={t('prompts.name')}
            rules={[{ required: true, message: t('prompts.pleaseInputName') }]}
          >
            <Input placeholder={t('prompts.pleaseInputName')} />
          </Form.Item>
          <Form.Item name="description" label={t('prompts.description')}>
            <Input placeholder={t('prompts.descriptionPlaceholder')} />
          </Form.Item>
          <Form.Item
            name="content"
            label={t('prompts.content')}
            rules={[{ required: true, message: t('prompts.pleaseInputContent') }]}
            extra={t('prompts.contentHint')}
          >
            <TextArea rows={15} placeholder={t('prompts.contentPlaceholder')} />
          </Form.Item>
          <Form.Item
            name="is_default"
            label={t('prompts.setAsDefault')}
            valuePropName="checked"
          >
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer
        title={t('prompts.viewPrompt')}
        width="min(960px, 92vw)"
        open={drawerVisible}
        onClose={() => setDrawerVisible(false)}
        styles={{ body: { padding: 24, display: 'flex', flexDirection: 'column', height: '100%' } }}
      >
        {viewingPrompt && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 16, flex: 1, minHeight: 0 }}>
            <div>
              <strong>{t('prompts.name')}:</strong> {viewingPrompt.name}
              {viewingPrompt.is_system && (
                <Tag color="blue" style={{ marginLeft: 8 }}>
                  {t('prompts.system')}
                </Tag>
              )}
              {viewingPrompt.is_default && (
                <Tag color="gold" style={{ marginLeft: 8 }}>
                  {t('prompts.isDefault')}
                </Tag>
              )}
            </div>
            {viewingPrompt.description && (
              <div>
                <strong>{t('prompts.description')}:</strong> {viewingPrompt.description}
              </div>
            )}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <strong style={{ fontSize: 14 }}>{t('prompts.content')}:</strong>
              <Segmented
                size="small"
                value={viewMode}
                onChange={(val) => setViewMode(val as 'rendered' | 'source')}
                options={[
                  { label: i18n.language?.startsWith('zh') ? '渲染' : 'Rendered', value: 'rendered' },
                  { label: i18n.language?.startsWith('zh') ? '源码' : 'Source', value: 'source' },
                ]}
              />
            </div>
            <div
              style={{
                flex: 1,
                minHeight: 0,
                backgroundColor: '#fafafa',
                borderRadius: 8,
                border: '1px solid #f0f0f0',
              }}
            >
              {viewMode === 'rendered' ? (
                <div
                  style={{
                    padding: 18,
                    height: '100%',
                    overflow: 'auto',
                  }}
                  className="markdown-body"
                >
                  <ReactMarkdown remarkPlugins={[remarkGfm]}>
                    {viewingPrompt.content}
                  </ReactMarkdown>
                </div>
              ) : (
                <div
                  style={{
                    padding: 18,
                    height: '100%',
                    overflow: 'auto',
                  }}
                >
                  <Paragraph
                    copyable
                    style={{
                      whiteSpace: 'pre-wrap',
                      margin: 0,
                      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
                      fontSize: 13,
                    }}
                  >
                    {viewingPrompt.content}
                  </Paragraph>
                </div>
              )}
            </div>
          </div>
        )}
      </Drawer>
    </>
  );
};

export default Prompts;
