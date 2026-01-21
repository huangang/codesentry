import React, { useState, useEffect } from 'react';
import {
  Card,
  Table,
  Button,
  Space,
  Input,
  Select,
  DatePicker,
  Tag,
  Drawer,
  Descriptions,
  message,
  Typography,
} from 'antd';
import { SearchOutlined, ReloadOutlined, EyeOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import { reviewLogApi, projectApi } from '../services';
import type { ReviewLog, Project } from '../types';

const { RangePicker } = DatePicker;
const { Paragraph } = Typography;

const ReviewLogs: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<ReviewLog[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [projects, setProjects] = useState<Project[]>([]);
  const [selectedLog, setSelectedLog] = useState<ReviewLog | null>(null);
  const [drawerVisible, setDrawerVisible] = useState(false);
  const { t } = useTranslation();
  
  // Filters
  const [eventType, setEventType] = useState<string>('');
  const [projectId, setProjectId] = useState<number | undefined>();
  const [author, setAuthor] = useState('');
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs] | null>(null);
  const [searchText, setSearchText] = useState('');

  const fetchData = async () => {
    setLoading(true);
    try {
      const params: any = {
        page,
        page_size: pageSize,
      };
      if (eventType) params.event_type = eventType;
      if (projectId) params.project_id = projectId;
      if (author) params.author = author;
      if (searchText) params.search_text = searchText;
      if (dateRange) {
        params.start_date = dateRange[0].format('YYYY-MM-DD');
        params.end_date = dateRange[1].format('YYYY-MM-DD');
      }

      const res = await reviewLogApi.list(params);
      setData(res.data.items);
      setTotal(res.data.total);
    } catch (error) {
      message.error(t('common.error'));
    } finally {
      setLoading(false);
    }
  };

  const fetchProjects = async () => {
    try {
      const res = await projectApi.list({ page_size: 100 });
      setProjects(res.data.items);
    } catch (error) {}
  };

  useEffect(() => {
    fetchData();
    fetchProjects();
  }, [page, pageSize]);

  const handleSearch = () => {
    setPage(1);
    fetchData();
  };

  const handleReset = () => {
    setEventType('');
    setProjectId(undefined);
    setAuthor('');
    setDateRange(null);
    setSearchText('');
    setPage(1);
    setTimeout(fetchData, 0);
  };

  const showDetail = (record: ReviewLog) => {
    setSelectedLog(record);
    setDrawerVisible(true);
  };

  const getScoreColor = (score: number | null) => {
    if (score === null) return 'default';
    if (score >= 80) return 'success';
    if (score >= 60) return 'warning';
    return 'error';
  };

  const getStatusText = (status: string) => {
    switch (status) {
      case 'pending': return t('reviewLogs.pending');
      case 'processing': return t('reviewLogs.processing');
      case 'completed': return t('reviewLogs.completed');
      case 'failed': return t('reviewLogs.failed');
      default: return status;
    }
  };

  const columns: ColumnsType<ReviewLog> = [
    {
      title: t('reviewLogs.project'),
      dataIndex: ['project', 'name'],
      key: 'project',
      width: 150,
      ellipsis: true,
    },
    {
      title: t('reviewLogs.author'),
      dataIndex: 'author',
      key: 'author',
      width: 120,
      ellipsis: true,
    },
    {
      title: t('reviewLogs.branch'),
      dataIndex: 'branch',
      key: 'branch',
      width: 120,
      ellipsis: true,
    },
    {
      title: t('reviewLogs.score'),
      dataIndex: 'score',
      key: 'score',
      width: 80,
      render: (score: number | null) => (
        <Tag color={getScoreColor(score)}>
          {score !== null ? score.toFixed(0) : '-'}
        </Tag>
      ),
    },
    {
      title: t('reviewLogs.changes', 'Changes'),
      key: 'changes',
      width: 120,
      render: (_, record) => (
        <Space>
          <span style={{ color: '#52c41a' }}>+{record.additions}</span>
          <span style={{ color: '#ff4d4f' }}>-{record.deletions}</span>
        </Space>
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
      title: t('reviewLogs.commitMessage'),
      dataIndex: 'commit_message',
      key: 'commit_message',
      ellipsis: true,
    },
    {
      title: t('common.actions'),
      key: 'action',
      width: 100,
      render: (_, record) => (
        <Button
          type="link"
          icon={<EyeOutlined />}
          onClick={() => showDetail(record)}
        >
          {t('common.details')}
        </Button>
      ),
    },
  ];

  return (
    <>
      <Card>
        <Space wrap style={{ marginBottom: 16 }}>
          <RangePicker
            value={dateRange}
            onChange={(dates) => setDateRange(dates as [dayjs.Dayjs, dayjs.Dayjs])}
          />
          <Select
            placeholder={t('reviewLogs.eventType')}
            allowClear
            style={{ width: 120 }}
            value={eventType || undefined}
            onChange={setEventType}
            options={[
              { value: 'push', label: t('reviewLogs.push') },
              { value: 'merge_request', label: t('reviewLogs.mergeRequest') },
            ]}
          />
          <Select
            placeholder={t('reviewLogs.project')}
            allowClear
            showSearch
            optionFilterProp="label"
            style={{ width: 180 }}
            value={projectId}
            onChange={setProjectId}
            options={projects.map(p => ({ value: p.id, label: p.name }))}
          />
          <Input
            placeholder={t('reviewLogs.author')}
            style={{ width: 120 }}
            value={author}
            onChange={(e) => setAuthor(e.target.value)}
          />
          <Input
            placeholder={t('reviewLogs.commitMessage')}
            style={{ width: 180 }}
            value={searchText}
            onChange={(e) => setSearchText(e.target.value)}
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

      <Drawer
        title={t('reviewLogs.reviewDetail')}
        width={720}
        open={drawerVisible}
        onClose={() => setDrawerVisible(false)}
      >
        {selectedLog && (
          <>
            <Descriptions column={2} bordered size="small">
              <Descriptions.Item label={t('reviewLogs.project')}>{selectedLog.project?.name}</Descriptions.Item>
              <Descriptions.Item label={t('reviewLogs.eventType')}>
                <Tag>{selectedLog.event_type}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label={t('reviewLogs.author')}>{selectedLog.author}</Descriptions.Item>
              <Descriptions.Item label={t('reviewLogs.branch')}>{selectedLog.branch}</Descriptions.Item>
              <Descriptions.Item label={t('reviewLogs.score')}>
                <Tag color={getScoreColor(selectedLog.score)}>
                  {selectedLog.score !== null ? selectedLog.score.toFixed(0) : '-'}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label={t('reviewLogs.reviewStatus')}>
                <Tag color={selectedLog.review_status === 'completed' ? 'success' : selectedLog.review_status === 'failed' ? 'error' : 'processing'}>
                  {getStatusText(selectedLog.review_status)}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label={t('reviewLogs.changes', 'Changes')} span={2}>
                <span style={{ color: '#52c41a' }}>+{selectedLog.additions}</span>
                {' / '}
                <span style={{ color: '#ff4d4f' }}>-{selectedLog.deletions}</span>
              </Descriptions.Item>
              <Descriptions.Item label="Commit Hash" span={2}>
                <code>{selectedLog.commit_hash}</code>
              </Descriptions.Item>
              <Descriptions.Item label={t('reviewLogs.commitMessage')} span={2}>
                {selectedLog.commit_message}
              </Descriptions.Item>
              <Descriptions.Item label={t('common.createdAt')} span={2}>
                {dayjs(selectedLog.created_at).format('YYYY-MM-DD HH:mm:ss')}
              </Descriptions.Item>
            </Descriptions>

            <Card title={t('reviewLogs.reviewResult')} size="small" style={{ marginTop: 16 }}>
              <Paragraph>
                <pre style={{ whiteSpace: 'pre-wrap', background: '#f5f5f5', padding: 16, borderRadius: 4 }}>
                  {selectedLog.review_result || t('common.noData')}
                </pre>
              </Paragraph>
            </Card>

            {selectedLog.error_message && (
              <Card title={t('reviewLogs.errorMessage')} size="small" style={{ marginTop: 16 }}>
                <Paragraph type="danger">{selectedLog.error_message}</Paragraph>
              </Card>
            )}
          </>
        )}
      </Drawer>
    </>
  );
};

export default ReviewLogs;
