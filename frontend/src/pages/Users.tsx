import React, { useEffect, useState } from 'react';
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
import { SearchOutlined, ReloadOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import { userApi } from '../services';
import type { User } from '../types';
import { useModal } from '../hooks';

interface UserFilters {
  username?: string;
  role?: string;
  auth_type?: string;
}

const Users: React.FC = () => {
  const { t } = useTranslation();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<User[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [searchUsername, setSearchUsername] = useState('');
  const [filterRole, setFilterRole] = useState<string>('');
  const [filterAuthType, setFilterAuthType] = useState<string>('');

  const modal = useModal<User>();

  const fetchData = async (filters: UserFilters = {}) => {
    try {
      setLoading(true);
      const res = await userApi.list({
        page,
        page_size: pageSize,
        ...filters,
      });
      setData(res.data.items || []);
      setTotal(res.data.total);
    } catch (error) {
      message.error(t('common.error'));
    } finally {
      setLoading(false);
    }
  };

  const buildFilters = (): UserFilters => {
    const filters: UserFilters = {};
    if (searchUsername) filters.username = searchUsername;
    if (filterRole) filters.role = filterRole;
    if (filterAuthType) filters.auth_type = filterAuthType;
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
    setSearchUsername('');
    setFilterRole('');
    setFilterAuthType('');
    setPage(1);
    fetchData({});
  };

  const showEditModal = (record: User) => {
    modal.open(record);
    form.setFieldsValue({
      role: record.role,
      is_active: record.is_active,
      nickname: record.nickname,
    });
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (modal.current) {
        await userApi.update(modal.current.id, values);
        message.success(t('users.updateSuccess'));
        modal.close();
        fetchData(buildFilters());
      }
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: string } } };
      message.error(err.response?.data?.error || t('common.error'));
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await userApi.delete(id);
      message.success(t('users.deleteSuccess'));
      fetchData(buildFilters());
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: string } } };
      message.error(err.response?.data?.error || t('common.error'));
    }
  };

  const handlePageChange = (newPage: number, newPageSize: number) => {
    setPage(newPage);
    setPageSize(newPageSize);
  };

  const columns: ColumnsType<User> = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 60,
    },
    {
      title: t('users.username'),
      dataIndex: 'username',
      key: 'username',
      width: 120,
    },
    {
      title: t('users.nickname'),
      dataIndex: 'nickname',
      key: 'nickname',
      width: 120,
    },
    {
      title: t('users.email'),
      dataIndex: 'email',
      key: 'email',
      width: 180,
      ellipsis: true,
    },
    {
      title: t('users.role'),
      dataIndex: 'role',
      key: 'role',
      width: 100,
      render: (role: string) => (
        <Tag color={role === 'admin' ? 'red' : 'blue'}>
          {role === 'admin' ? t('users.admin') : t('users.user')}
        </Tag>
      ),
    },
    {
      title: t('users.authType'),
      dataIndex: 'auth_type',
      key: 'auth_type',
      width: 100,
      render: (authType: string) => (
        <Tag color={authType === 'ldap' ? 'purple' : 'green'}>
          {authType.toUpperCase()}
        </Tag>
      ),
    },
    {
      title: t('users.isActive'),
      dataIndex: 'is_active',
      key: 'is_active',
      width: 80,
      render: (isActive: boolean) => (
        <Tag color={isActive ? 'success' : 'default'}>
          {isActive ? t('common.yes') : t('common.no')}
        </Tag>
      ),
    },
    {
      title: t('users.lastLogin'),
      dataIndex: 'last_login',
      key: 'last_login',
      width: 160,
      render: (val: string | null) => val ? dayjs(val).format('YYYY-MM-DD HH:mm') : '-',
    },
    {
      title: t('common.actions'),
      key: 'action',
      width: 100,
      render: (_, record) => (
        <Space>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => showEditModal(record)} />
          <Popconfirm title={t('users.deleteConfirm')} onConfirm={() => handleDelete(record.id)}>
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
            placeholder={t('users.username')}
            style={{ width: 150 }}
            value={searchUsername}
            onChange={(e) => setSearchUsername(e.target.value)}
            onPressEnter={handleSearch}
          />
          <Select
            placeholder={t('users.role')}
            allowClear
            style={{ width: 120 }}
            value={filterRole || undefined}
            onChange={setFilterRole}
            options={[
              { value: 'admin', label: t('users.admin') },
              { value: 'user', label: t('users.user') },
            ]}
          />
          <Select
            placeholder={t('users.authType')}
            allowClear
            style={{ width: 120 }}
            value={filterAuthType || undefined}
            onChange={setFilterAuthType}
            options={[
              { value: 'local', label: 'LOCAL' },
              { value: 'ldap', label: 'LDAP' },
            ]}
          />
          <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
            {t('common.search')}
          </Button>
          <Button icon={<ReloadOutlined />} onClick={handleReset}>
            {t('common.reset')}
          </Button>
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
        title={t('users.editUser')}
        open={modal.visible}
        onOk={handleSubmit}
        onCancel={modal.close}
        width={480}
      >
        <Form form={form} layout="vertical">
          <Form.Item label={t('users.username')}>
            <Input value={modal.current?.username} disabled />
          </Form.Item>
          <Form.Item name="nickname" label={t('users.nickname')}>
            <Input />
          </Form.Item>
          <Form.Item name="role" label={t('users.role')} rules={[{ required: true }]}>
            <Select
              options={[
                { value: 'admin', label: t('users.admin') },
                { value: 'user', label: t('users.user') },
              ]}
            />
          </Form.Item>
          <Form.Item name="is_active" label={t('users.isActive')} valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

export default Users;
