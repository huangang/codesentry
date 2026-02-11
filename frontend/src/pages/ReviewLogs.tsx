import React, { useState, useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';
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
  Divider,
  List,
  Avatar,
  Spin,
} from 'antd';
import { SearchOutlined, ReloadOutlined, EyeOutlined, LinkOutlined, DeleteOutlined, SendOutlined, CommentOutlined, CheckCircleOutlined, CloseCircleOutlined, QuestionCircleOutlined, InfoCircleOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import type { ReviewLog } from '../types';
import { usePermission, getResponsiveWidth } from '../hooks';
import { useReviewSSE, type ReviewEvent } from '../hooks/useSSE';
import {
  useReviewLogs,
  useRetryReview,
  useDeleteReviewLog,
  useProjects,
  useReviewFeedbacks,
  useCreateReviewFeedback,
  type ReviewLogFilters,
} from '../hooks/queries';
import { MarkdownContent } from '../components';
import { REVIEW_STATUS, EVENT_TYPES, getScoreColor, getStatusColor } from '../constants';

const { RangePicker } = DatePicker;
const { Paragraph, Text } = Typography;
const { TextArea } = Input;

// Feedback Section Component
const FeedbackSection: React.FC<{ reviewLogId: number }> = ({ reviewLogId }) => {
  const { t } = useTranslation();
  const [feedbackType, setFeedbackType] = useState<string>('question');
  const [feedbackMessage, setFeedbackMessage] = useState('');

  const { data: feedbacks, isLoading } = useReviewFeedbacks(reviewLogId);
  const createFeedback = useCreateReviewFeedback();

  const feedbackTypeOptions = [
    { value: 'agree', label: t('feedback.types.agree', '同意'), icon: <CheckCircleOutlined style={{ color: '#52c41a' }} /> },
    { value: 'disagree', label: t('feedback.types.disagree', '不同意'), icon: <CloseCircleOutlined style={{ color: '#ff4d4f' }} /> },
    { value: 'question', label: t('feedback.types.question', '提问'), icon: <QuestionCircleOutlined style={{ color: '#1890ff' }} /> },
    { value: 'clarification', label: t('feedback.types.clarification', '需要说明'), icon: <InfoCircleOutlined style={{ color: '#faad14' }} /> },
  ];

  const handleSubmit = async () => {
    if (!feedbackMessage.trim()) return;
    try {
      await createFeedback.mutateAsync({
        review_log_id: reviewLogId,
        feedback_type: feedbackType,
        user_message: feedbackMessage,
      });
      setFeedbackMessage('');
      message.success(t('feedback.submitSuccess', '反馈已提交'));
    } catch {
      message.error(t('feedback.submitError', '提交失败'));
    }
  };

  const getStatusTag = (status: string) => {
    switch (status) {
      case 'completed': return <Tag color="success">{t('feedback.status.completed', '已回复')}</Tag>;
      case 'processing': return <Tag color="processing">{t('feedback.status.processing', '处理中')}</Tag>;
      case 'failed': return <Tag color="error">{t('feedback.status.failed', '失败')}</Tag>;
      default: return <Tag>{t('feedback.status.pending', '等待中')}</Tag>;
    }
  };

  return (
    <Card
      title={<><CommentOutlined /> {t('feedback.title', 'AI 反馈对话')}</>}
      size="small"
      style={{ marginTop: 16 }}
    >
      {/* Feedback Form */}
      <Space direction="vertical" style={{ width: '100%' }} size="small">
        <Space>
          <Text strong>{t('feedback.type', '反馈类型')}:</Text>
          <Select
            value={feedbackType}
            onChange={setFeedbackType}
            style={{ width: 140 }}
            options={feedbackTypeOptions}
          />
        </Space>
        <TextArea
          value={feedbackMessage}
          onChange={(e) => setFeedbackMessage(e.target.value)}
          placeholder={t('feedback.placeholder', '输入您的反馈或问题，AI将重新评估...')}
          rows={3}
        />
        <Button
          type="primary"
          icon={<SendOutlined />}
          onClick={handleSubmit}
          loading={createFeedback.isPending}
          disabled={!feedbackMessage.trim()}
        >
          {t('feedback.submit', '提交反馈')}
        </Button>
      </Space>

      {/* Feedback History */}
      {isLoading ? (
        <Spin style={{ display: 'block', marginTop: 16 }} />
      ) : feedbacks && feedbacks.length > 0 ? (
        <>
          <Divider>{t('feedback.history', '历史反馈')}</Divider>
          <List
            itemLayout="vertical"
            dataSource={feedbacks}
            renderItem={(item) => (
              <List.Item>
                <List.Item.Meta
                  avatar={<Avatar icon={<CommentOutlined />} />}
                  title={
                    <Space>
                      <Tag>{feedbackTypeOptions.find(o => o.value === item.feedback_type)?.label || item.feedback_type}</Tag>
                      {getStatusTag(item.process_status)}
                      {item.score_changed && (
                        <Tag color="orange">
                          {t('feedback.scoreChanged', '评分已更新')}: {item.previous_score?.toFixed(0)} → {item.updated_score?.toFixed(0)}
                        </Tag>
                      )}
                    </Space>
                  }
                  description={dayjs(item.created_at).format('YYYY-MM-DD HH:mm')}
                />
                <Paragraph style={{ marginBottom: 8 }}><strong>{t('feedback.yourMessage', '您的反馈')}:</strong> {item.user_message}</Paragraph>
                {item.ai_response && (
                  <Card size="small" style={{ background: 'var(--ant-color-bg-container-alt, #fafafa)' }}>
                    <MarkdownContent content={item.ai_response} />
                  </Card>
                )}
              </List.Item>
            )}
          />
        </>
      ) : null}
    </Card>
  );
};

const ReviewLogs: React.FC = () => {
  const { t } = useTranslation();
  const { isAdmin } = usePermission();
  const [selectedLog, setSelectedLog] = useState<ReviewLog | null>(null);
  const [drawerVisible, setDrawerVisible] = useState(false);

  const [eventType, setEventType] = useState<string>('');
  const [projectId, setProjectId] = useState<number | undefined>();
  const [author, setAuthor] = useState('');
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs] | null>(null);
  const [searchText, setSearchText] = useState('');
  const [filters, setFilters] = useState<ReviewLogFilters>({ page: 1, page_size: 10 });

  const { data: logsData, isLoading } = useReviewLogs(filters);
  const { data: projectsData } = useProjects({ page_size: 100 });
  const retryReview = useRetryReview();
  const deleteReviewLog = useDeleteReviewLog();
  const queryClient = useQueryClient();

  // SSE real-time updates
  const handleSSEEvent = useCallback((event: ReviewEvent) => {
    // Update the specific review log in cache
    queryClient.setQueryData(['reviewLogs', filters], (old: { items: ReviewLog[]; total: number } | undefined) => {
      if (!old) return old;
      return {
        ...old,
        items: old.items.map((item: ReviewLog) =>
          item.id === event.id
            ? {
              ...item,
              review_status: event.status,
              score: event.score ?? item.score,
              error_message: event.error ?? item.error_message,
            }
            : item
        ),
      };
    });
  }, [queryClient, filters]);

  useReviewSSE({ onEvent: handleSSEEvent });

  const buildFilters = useCallback((): ReviewLogFilters => {
    const newFilters: ReviewLogFilters = { page: 1, page_size: filters.page_size };
    if (eventType) newFilters.event_type = eventType;
    if (projectId) newFilters.project_id = projectId;
    if (author) newFilters.author = author;
    if (searchText) newFilters.search_text = searchText;
    if (dateRange) {
      newFilters.start_date = dateRange[0].format('YYYY-MM-DD');
      newFilters.end_date = dateRange[1].format('YYYY-MM-DD');
    }
    return newFilters;
  }, [eventType, projectId, author, searchText, dateRange, filters.page_size]);

  const handleSearch = () => {
    setFilters(buildFilters());
  };

  const handleReset = () => {
    setEventType('');
    setProjectId(undefined);
    setAuthor('');
    setDateRange(null);
    setSearchText('');
    setFilters({ page: 1, page_size: 10 });
  };

  const handlePageChange = (page: number, pageSize: number) => {
    setFilters(prev => ({ ...prev, page, page_size: pageSize }));
  };

  const showDetail = (record: ReviewLog) => {
    setSelectedLog(record);
    setDrawerVisible(true);
  };

  const handleRetry = async (id: number) => {
    try {
      await retryReview.mutateAsync(id);
      message.success(t('reviewLogs.retryInitiated', 'Retry initiated'));
      setDrawerVisible(false);
    } catch (error) {
      message.error(t('common.error'));
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await deleteReviewLog.mutateAsync(id);
      message.success(t('reviewLogs.deleteSuccess', 'Review log deleted successfully'));
      setDrawerVisible(false);
    } catch (error) {
      message.error(t('common.error'));
    }
  };

  const getStatusText = (status: string) => {
    switch (status) {
      case REVIEW_STATUS.PENDING: return t('reviewLogs.pending');
      case REVIEW_STATUS.PROCESSING: return t('reviewLogs.processing');
      case REVIEW_STATUS.ANALYZING: return t('reviewLogs.analyzing', 'Analyzing');
      case REVIEW_STATUS.COMPLETED: return t('reviewLogs.completed');
      case REVIEW_STATUS.FAILED: return t('reviewLogs.failed');
      case REVIEW_STATUS.SKIPPED: return t('reviewLogs.skipped', 'Skipped');
      default: return status;
    }
  };

  const canRetry = (status: string) => {
    return status === REVIEW_STATUS.FAILED;
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
      width: 100,
      render: (score: number | null, record: ReviewLog) => {
        // Show status tag for non-completed reviews
        if (record.review_status === REVIEW_STATUS.SKIPPED) {
          return <Tag color="warning">{t('reviewLogs.skipped', 'Skipped')}</Tag>;
        }
        if (record.review_status === REVIEW_STATUS.PENDING ||
          record.review_status === REVIEW_STATUS.ANALYZING ||
          record.review_status === REVIEW_STATUS.PROCESSING) {
          return <Tag color="processing">{getStatusText(record.review_status)}</Tag>;
        }
        if (record.review_status === REVIEW_STATUS.FAILED) {
          return <Tag color="error">{t('reviewLogs.failed')}</Tag>;
        }
        return (
          <Tag color={getScoreColor(score)}>
            {score !== null ? score.toFixed(0) : '-'}
          </Tag>
        );
      },
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
      width: 180,
      render: (_, record) => (
        <Space>
          <Button
            type="link"
            icon={<EyeOutlined />}
            onClick={() => showDetail(record)}
          >
            {t('common.details')}
          </Button>
          {isAdmin && canRetry(record.review_status) && (
            <Button
              type="link"
              icon={<ReloadOutlined />}
              onClick={() => handleRetry(record.id)}
              loading={retryReview.isPending}
            >
              {t('reviewLogs.retry', 'Retry')}
            </Button>
          )}
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
            options={projectsData?.items?.map(p => ({ value: p.id, label: p.name })) ?? []}
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
          dataSource={logsData?.items ?? []}
          rowKey="id"
          loading={isLoading}
          scroll={{ x: 900 }}
          size="middle"
          pagination={{
            current: filters.page,
            pageSize: filters.page_size,
            total: logsData?.total ?? 0,
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
                    loading={retryReview.isPending}
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

            {/* AI Feedback Section */}
            <FeedbackSection reviewLogId={selectedLog.id} />
          </>
        )}
      </Drawer>
    </>
  );
};

export default ReviewLogs;
