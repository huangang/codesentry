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
  Drawer,
  Typography,
  Tooltip,
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
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import { promptApi } from '../services';
import type { PromptTemplate } from '../types';

const { TextArea } = Input;
const { Paragraph } = Typography;

const Prompts: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<PromptTemplate[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [modalVisible, setModalVisible] = useState(false);
  const [drawerVisible, setDrawerVisible] = useState(false);
  const [currentPrompt, setCurrentPrompt] = useState<PromptTemplate | null>(null);
  const [form] = Form.useForm();
  const { t } = useTranslation();

  const [searchName, setSearchName] = useState('');
  const [filterType, setFilterType] = useState<string>('');

  const fetchData = async () => {
    setLoading(true);
    try {
      const params: any = { page, page_size: pageSize };
      if (searchName) params.name = searchName;
      if (filterType === 'system') params.is_system = true;
      if (filterType === 'custom') params.is_system = false;
      const res = await promptApi.list(params);
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
    setFilterType('');
    setPage(1);
    setTimeout(fetchData, 0);
  };

  const showCreateModal = () => {
    setCurrentPrompt(null);
    form.resetFields();
    form.setFieldsValue({
      is_default: false,
    });
    setModalVisible(true);
  };

  const showEditModal = (record: PromptTemplate) => {
    if (record.is_system) {
      message.warning(t('prompts.cannotEditSystem'));
      return;
    }
    setCurrentPrompt(record);
    form.setFieldsValue(record);
    setModalVisible(true);
  };

  const showViewDrawer = (record: PromptTemplate) => {
    setCurrentPrompt(record);
    setDrawerVisible(true);
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (currentPrompt) {
        await promptApi.update(currentPrompt.id, values);
        message.success(t('prompts.updateSuccess'));
      } else {
        await promptApi.create(values);
        message.success(t('prompts.createSuccess'));
      }
      setModalVisible(false);
      fetchData();
    } catch (error: any) {
      message.error(error.response?.data?.error || t('common.error'));
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await promptApi.delete(id);
      message.success(t('prompts.deleteSuccess'));
      fetchData();
    } catch (error: any) {
      message.error(error.response?.data?.error || t('common.error'));
    }
  };

  const handleSetDefault = async (id: number) => {
    try {
      await promptApi.setDefault(id);
      message.success(t('prompts.setDefaultSuccess'));
      fetchData();
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
          <Button type="primary" icon={<PlusOutlined />} onClick={showCreateModal}>
            {t('prompts.createPrompt')}
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
        title={currentPrompt ? t('prompts.editPrompt') : t('prompts.createPrompt')}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => setModalVisible(false)}
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
        width={720}
        open={drawerVisible}
        onClose={() => setDrawerVisible(false)}
      >
        {currentPrompt && (
          <div>
            <div style={{ marginBottom: 16 }}>
              <strong>{t('prompts.name')}:</strong> {currentPrompt.name}
              {currentPrompt.is_system && (
                <Tag color="blue" style={{ marginLeft: 8 }}>
                  {t('prompts.system')}
                </Tag>
              )}
              {currentPrompt.is_default && (
                <Tag color="gold" style={{ marginLeft: 8 }}>
                  {t('prompts.isDefault')}
                </Tag>
              )}
            </div>
            {currentPrompt.description && (
              <div style={{ marginBottom: 16 }}>
                <strong>{t('prompts.description')}:</strong> {currentPrompt.description}
              </div>
            )}
            <div style={{ marginBottom: 8 }}>
              <strong>{t('prompts.content')}:</strong>
            </div>
            <Paragraph
              copyable
              style={{
                whiteSpace: 'pre-wrap',
                backgroundColor: '#f5f5f5',
                padding: 16,
                borderRadius: 8,
                maxHeight: 500,
                overflow: 'auto',
              }}
            >
              {currentPrompt.content}
            </Paragraph>
          </div>
        )}
      </Drawer>
    </>
  );
};

export default Prompts;
