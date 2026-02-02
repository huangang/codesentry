import React, { useState, useEffect } from 'react';
import {
  Card,
  Table,
  Space,
  Input,
  Select,
  DatePicker,
  Tag,
  Button,
  Drawer,
  Descriptions,
  message,
  Typography,
  InputNumber,
  Popconfirm,
  Divider,
} from 'antd';
import { SearchOutlined, ReloadOutlined, EyeOutlined, DeleteOutlined, SettingOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import { systemLogApi, type SystemLog } from '../services';
import { useSystemLogs, useCleanupSystemLogs, type SystemLogFilters } from '../hooks/queries';

const { RangePicker } = DatePicker;
const { Paragraph } = Typography;

const SystemLogs: React.FC = () => {
  const [modules, setModules] = useState<string[]>([]);
  const [selectedLog, setSelectedLog] = useState<SystemLog | null>(null);
  const [drawerVisible, setDrawerVisible] = useState(false);
  const [retentionDays, setRetentionDays] = useState(30);
  const [retentionLoading, setRetentionLoading] = useState(false);
  const { t } = useTranslation();

  const [level, setLevel] = useState<string>('');
  const [module, setModule] = useState<string>('');
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs] | null>(null);
  const [search, setSearch] = useState('');
  const [filters, setFilters] = useState<SystemLogFilters>({ page: 1, page_size: 20 });

  const { data: logsData, isLoading } = useSystemLogs(filters);
  const cleanupLogs = useCleanupSystemLogs();

  useEffect(() => {
    const fetchModules = async () => {
      try {
        const res = await systemLogApi.getModules();
        setModules(res.data.modules || []);
      } catch { }
    };
    const fetchRetentionDays = async () => {
      try {
        const res = await systemLogApi.getRetentionDays();
        setRetentionDays(res.data.retention_days);
      } catch { }
    };
    fetchModules();
    fetchRetentionDays();
  }, []);

  const handleSearch = () => {
    const newFilters: SystemLogFilters = { page: 1, page_size: filters.page_size };
    if (level) newFilters.level = level;
    if (module) newFilters.module = module;
    if (search) newFilters.search = search;
    if (dateRange) {
      newFilters.start_date = dateRange[0].format('YYYY-MM-DD');
      newFilters.end_date = dateRange[1].format('YYYY-MM-DD');
    }
    setFilters(newFilters);
  };

  const handleReset = () => {
    setLevel('');
    setModule('');
    setDateRange(null);
    setSearch('');
    setFilters({ page: 1, page_size: 20 });
  };

  const handlePageChange = (page: number, pageSize: number) => {
    setFilters(prev => ({ ...prev, page, page_size: pageSize }));
  };

  const handleSaveRetention = async () => {
    setRetentionLoading(true);
    try {
      await systemLogApi.setRetentionDays(retentionDays);
      message.success(t('common.success'));
    } catch {
      message.error(t('common.error'));
    } finally {
      setRetentionLoading(false);
    }
  };

  const handleCleanup = async () => {
    try {
      const res = await cleanupLogs.mutateAsync();
      message.success(t('systemLogs.cleanupSuccess', { count: res.deleted }));
    } catch {
      message.error(t('common.error'));
    }
  };

  const showDetail = (record: SystemLog) => {
    setSelectedLog(record);
    setDrawerVisible(true);
  };

  const getLevelColor = (lvl: string) => {
    switch (lvl) {
      case 'error': return 'error';
      case 'warning': return 'warning';
      case 'info': return 'processing';
      default: return 'default';
    }
  };

  const columns: ColumnsType<SystemLog> = [
    { title: t('systemLogs.time'), dataIndex: 'created_at', key: 'created_at', width: 160, render: (val: string) => dayjs(val).format('YYYY-MM-DD HH:mm:ss') },
    { title: t('systemLogs.level'), dataIndex: 'level', key: 'level', width: 80, render: (lvl: string) => <Tag color={getLevelColor(lvl)}>{lvl.toUpperCase()}</Tag> },
    { title: t('systemLogs.module'), dataIndex: 'module', key: 'module', width: 120, ellipsis: true },
    { title: t('systemLogs.action'), dataIndex: 'action', key: 'action', width: 150, ellipsis: true },
    { title: t('systemLogs.message'), dataIndex: 'message', key: 'message', ellipsis: true },
    { title: t('systemLogs.ip'), dataIndex: 'ip', key: 'ip', width: 120 },
    {
      title: t('common.actions'), key: 'action', width: 80,
      render: (_, record) => <Button type="link" icon={<EyeOutlined />} onClick={() => showDetail(record)}>{t('common.details')}</Button>,
    },
  ];

  return (
    <>
      <Card>
        <Space wrap style={{ marginBottom: 16 }}>
          <RangePicker value={dateRange} onChange={(dates) => setDateRange(dates as [dayjs.Dayjs, dayjs.Dayjs])} />
          <Select placeholder={t('systemLogs.level')} allowClear style={{ width: 100 }} value={level || undefined} onChange={setLevel} options={[{ value: 'info', label: 'INFO' }, { value: 'warning', label: 'WARNING' }, { value: 'error', label: 'ERROR' }]} />
          <Select placeholder={t('systemLogs.module')} allowClear showSearch style={{ width: 150 }} value={module || undefined} onChange={setModule} options={modules.map(m => ({ value: m, label: m }))} />
          <Input placeholder={t('systemLogs.searchMessage')} style={{ width: 200 }} value={search} onChange={(e) => setSearch(e.target.value)} onPressEnter={handleSearch} />
          <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>{t('common.search')}</Button>
          <Button icon={<ReloadOutlined />} onClick={handleReset}>{t('common.reset')}</Button>
          <Divider type="vertical" />
          <SettingOutlined style={{ color: '#666' }} />
          <span style={{ color: '#666', fontSize: 13 }}>{t('systemLogs.retentionDays')}:</span>
          <InputNumber min={1} max={365} value={retentionDays} onChange={(val) => setRetentionDays(val || 30)} style={{ width: 80 }} />
          <Button size="small" onClick={handleSaveRetention} loading={retentionLoading}>{t('common.save')}</Button>
          <Popconfirm title={t('systemLogs.cleanupConfirm', { days: retentionDays })} onConfirm={handleCleanup} okText={t('common.confirm')} cancelText={t('common.cancel')}>
            <Button size="small" danger icon={<DeleteOutlined />} loading={cleanupLogs.isPending}>{t('systemLogs.cleanup')}</Button>
          </Popconfirm>
        </Space>

        <Table columns={columns} dataSource={logsData?.items ?? []} rowKey="id" loading={isLoading} scroll={{ x: 1000 }}
          pagination={{ current: filters.page, pageSize: filters.page_size, total: logsData?.total ?? 0, showSizeChanger: true, pageSizeOptions: ['20', '50', '100'], showTotal: (total) => `${t('common.total')} ${total}`, onChange: handlePageChange }} />
      </Card>

      <Drawer title={t('systemLogs.logDetail')} width={640} open={drawerVisible} onClose={() => setDrawerVisible(false)}>
        {selectedLog && (
          <>
            <Descriptions column={1} bordered size="small">
              <Descriptions.Item label={t('systemLogs.time')}>{dayjs(selectedLog.created_at).format('YYYY-MM-DD HH:mm:ss')}</Descriptions.Item>
              <Descriptions.Item label={t('systemLogs.level')}><Tag color={getLevelColor(selectedLog.level)}>{selectedLog.level.toUpperCase()}</Tag></Descriptions.Item>
              <Descriptions.Item label={t('systemLogs.module')}>{selectedLog.module}</Descriptions.Item>
              <Descriptions.Item label={t('systemLogs.action')}>{selectedLog.action}</Descriptions.Item>
              <Descriptions.Item label={t('systemLogs.ip')}>{selectedLog.ip || '-'}</Descriptions.Item>
              <Descriptions.Item label={t('systemLogs.userAgent')}>{selectedLog.user_agent || '-'}</Descriptions.Item>
            </Descriptions>
            <Card title={t('systemLogs.message')} size="small" style={{ marginTop: 16 }}>
              <Paragraph><pre style={{ whiteSpace: 'pre-wrap', background: '#f5f5f5', padding: 16, borderRadius: 4 }}>{selectedLog.message || '-'}</pre></Paragraph>
            </Card>
            {selectedLog.extra && (
              <Card title={t('systemLogs.extra')} size="small" style={{ marginTop: 16 }}>
                <Paragraph><pre style={{ whiteSpace: 'pre-wrap', background: '#f5f5f5', padding: 16, borderRadius: 4 }}>{selectedLog.extra}</pre></Paragraph>
              </Card>
            )}
          </>
        )}
      </Drawer>
    </>
  );
};

export default SystemLogs;
