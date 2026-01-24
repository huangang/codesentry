import React, { useState, useEffect, useCallback } from 'react';
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
  Popconfirm,
} from 'antd';
import { SearchOutlined, ReloadOutlined, EyeOutlined, LinkOutlined, DeleteOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import { reviewLogApi, projectApi, reviewLogApiExtra } from '../services';
import type { ReviewLog, Project } from '../types';
import { usePaginatedList, usePermission, getResponsiveWidth } from '../hooks';
import { MarkdownContent } from '../components';
import { REVIEW_STATUS, EVENT_TYPES, getScoreColor, getStatusColor } from '../constants';

const { RangePicker } = DatePicker;
const { Paragraph } = Typography;

interface ReviewLogFilters {
  event_type?: string;
  project_id?: number;
  author?: string;
  start_date?: string;
  end_date?: string;
  search_text?: string;
}

const ReviewLogs: React.FC = () => {
  const { t } = useTranslation();
  const { isAdmin } = usePermission();
  const [projects, setProjects] = useState<Project[]>([]);
  const [selectedLog, setSelectedLog] = useState<ReviewLog | null>(null);
  const [drawerVisible, setDrawerVisible] = useState(false);
  
  const [eventType, setEventType] = useState<string>('');
  const [projectId, setProjectId] = useState<number | undefined>();
  const [author, setAuthor] = useState('');
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs] | null>(null);
  const [searchText, setSearchText] = useState('');

  const {
    loading,
    data,
    total,
    page,
    pageSize,
    setPage,
    fetchData,
    handlePageChange,
  } = usePaginatedList<ReviewLog, ReviewLogFilters>({
    fetchApi: reviewLogApi.list,
    onError: () => message.error(t('common.error')),
  });

  const buildFilters = useCallback((): ReviewLogFilters => {
    const filters: ReviewLogFilters = {};
    if (eventType) filters.event_type = eventType;
    if (projectId) filters.project_id = projectId;
    if (author) filters.author = author;
    if (searchText) filters.search_text = searchText;
    if (dateRange) {
      filters.start_date = dateRange[0].format('YYYY-MM-DD');
      filters.end_date = dateRange[1].format('YYYY-MM-DD');
    }
    return filters;
  }, [eventType, projectId, author, searchText, dateRange]);

  const fetchProjects = useCallback(async () => {
    try {
      const res = await projectApi.list({ page_size: 100 });
      setProjects(res.data.items);
    } catch (error) {
      console.error('Failed to fetch projects:', error);
    }
  }, []);

  useEffect(() => {
    fetchData(buildFilters());
  }, [page, pageSize]);

  useEffect(() => {
    fetchProjects();
  }, [fetchProjects]);

  const handleSearch = () => {
    setPage(1);
    fetchData(buildFilters());
  };

  const handleReset = () => {
    setEventType('');
    setProjectId(undefined);
    setAuthor('');
    setDateRange(null);
    setSearchText('');
    setPage(1);
    fetchData({});
  };

  const showDetail = (record: ReviewLog) => {
    setSelectedLog(record);
    setDrawerVisible(true);
  };

  const handleRetry = async (id: number) => {
    try {
      await reviewLogApi.retry(id);
      message.success(t('reviewLogs.retryInitiated', 'Retry initiated'));
      fetchData(buildFilters());
      setDrawerVisible(false);
    } catch (error) {
      message.error(t('common.error'));
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await reviewLogApiExtra.delete(id);
      message.success(t('reviewLogs.deleteSuccess', 'Review log deleted successfully'));
      fetchData(buildFilters());
      setDrawerVisible(false);
    } catch (error) {
      message.error(t('common.error'));
    }
  };

  const getStatusText = (status: string) => {
    switch (status) {
      case REVIEW_STATUS.PENDING: return t('reviewLogs.pending');
      case REVIEW_STATUS.PROCESSING: return t('reviewLogs.processing');
      case REVIEW_STATUS.COMPLETED: return t('reviewLogs.completed');
      case REVIEW_STATUS.FAILED: return t('reviewLogs.failed');
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
      width: 150,
      render: (_, record) => (
        <Space>
          <Button
            type="link"
            icon={<EyeOutlined />}
            onClick={() => showDetail(record)}
          >
            {t('common.details')}
          </Button>
          {isAdmin && (
            <Popconfirm
              title={t('reviewLogs.deleteConfirm', 'Are you sure you want to delete this review log?')}
              onConfirm={() => handleDelete(record.id)}
              okText={t('common.yes')}
              cancelText={t('common.no')}
            >
              <Button
                type="link"
                danger
                icon={<DeleteOutlined />}
              />
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  return (
    <>
      <Card styles={{ body: { padding: '16px 12px' } }}>
        <Space wrap style={{ marginBottom: 16, width: '100%' }} className="filter-area">
          <RangePicker
            value={dateRange}
            onChange={(dates) => setDateRange(dates as [dayjs.Dayjs, dayjs.Dayjs])}
            style={{ minWidth: 220 }}
          />
          <Select
            placeholder={t('reviewLogs.eventType')}
            allowClear
            style={{ minWidth: 100 }}
            value={eventType || undefined}
            onChange={setEventType}
            options={[
              { value: EVENT_TYPES.PUSH, label: t('reviewLogs.push') },
              { value: EVENT_TYPES.MERGE_REQUEST, label: t('reviewLogs.mergeRequest') },
            ]}
          />
          <Select
            placeholder={t('reviewLogs.project')}
            allowClear
            showSearch
            optionFilterProp="label"
            style={{ minWidth: 140 }}
            value={projectId}
            onChange={setProjectId}
            options={projects.map(p => ({ value: p.id, label: p.name }))}
          />
          <Input
            placeholder={t('reviewLogs.author')}
            style={{ minWidth: 100, maxWidth: 120 }}
            value={author}
            onChange={(e) => setAuthor(e.target.value)}
          />
          <Input
            placeholder={t('reviewLogs.commitMessage')}
            style={{ minWidth: 140 }}
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
          scroll={{ x: 900 }}
          size="middle"
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            showTotal: (total) => `${t('common.total')} ${total}`,
            onChange: handlePageChange,
            size: 'small',
          }}
        />
      </Card>

      <Drawer
        title={t('reviewLogs.reviewDetail')}
        width={getResponsiveWidth(720)}
        open={drawerVisible}
        onClose={() => setDrawerVisible(false)}
        styles={{ body: { padding: '16px 12px' } }}
        extra={
          isAdmin && selectedLog && (
            <Popconfirm
              title={t('reviewLogs.deleteConfirm', 'Are you sure you want to delete this review log?')}
              onConfirm={() => handleDelete(selectedLog.id)}
              okText={t('common.yes')}
              cancelText={t('common.no')}
            >
              <Button danger icon={<DeleteOutlined />}>
                {t('common.delete')}
              </Button>
            </Popconfirm>
          )
        }
      >
        {selectedLog && (
          <>
            <Descriptions column={{ xs: 1, sm: 2 }} bordered size="small">
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
                <Tag color={getStatusColor(selectedLog.review_status)}>
                  {getStatusText(selectedLog.review_status)}
                </Tag>
                {selectedLog.review_status === REVIEW_STATUS.FAILED && selectedLog.retry_count > 0 && (
                  <Tag color="orange" style={{ marginLeft: 8 }}>
                    {t('reviewLogs.retryCount', 'Retries')}: {selectedLog.retry_count}/3
                  </Tag>
                )}
                {selectedLog.review_status === REVIEW_STATUS.FAILED && (
                  <Button 
                    type="link" 
                    size="small" 
                    icon={<ReloadOutlined />}
                    onClick={() => handleRetry(selectedLog.id)}
                    style={{ marginLeft: 8 }}
                  >
                    {t('reviewLogs.retry', 'Retry')}
                  </Button>
                )}
              </Descriptions.Item>
              <Descriptions.Item label={t('reviewLogs.changes', 'Changes')} span={2}>
                <span style={{ color: '#52c41a' }}>+{selectedLog.additions}</span>
                {' / '}
                <span style={{ color: '#ff4d4f' }}>-{selectedLog.deletions}</span>
              </Descriptions.Item>
              <Descriptions.Item label="Commit Hash" span={2}>
                <Space>
                  <code>{selectedLog.commit_hash}</code>
                  {(selectedLog.commit_url || selectedLog.mr_url) && (
                    <Button
                      type="link"
                      size="small"
                      icon={<LinkOutlined />}
                      href={selectedLog.mr_url || selectedLog.commit_url}
                      target="_blank"
                    >
                      {selectedLog.mr_url ? t('reviewLogs.viewMR', '查看 MR') : t('reviewLogs.viewCommit', '查看 Commit')}
                    </Button>
                  )}
                </Space>
              </Descriptions.Item>
              <Descriptions.Item label={t('reviewLogs.commitMessage')} span={2}>
                {selectedLog.commit_message}
              </Descriptions.Item>
              <Descriptions.Item label={t('common.createdAt')} span={2}>
                {dayjs(selectedLog.created_at).format('YYYY-MM-DD HH:mm:ss')}
              </Descriptions.Item>
            </Descriptions>

            <Card title={t('reviewLogs.reviewResult')} size="small" style={{ marginTop: 16 }}>
              {selectedLog.review_result ? (
                <MarkdownContent content={selectedLog.review_result} />
              ) : (
                <Paragraph type="secondary">{t('common.noData')}</Paragraph>
              )}
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
