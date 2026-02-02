import React, { useState } from 'react';
import {
  Card,
  Table,
  Space,
  Button,
  Drawer,
  Descriptions,
  message,
  Typography,
  Tag,
  Popconfirm,
} from 'antd';
import { ReloadOutlined, EyeOutlined, SendOutlined, PlusOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import ReactMarkdown from 'react-markdown';
import { type DailyReport } from '../services';
import { getResponsiveWidth } from '../hooks';
import { useThemeStore } from '../stores/themeStore';
import {
  useDailyReports,
  useGenerateDailyReport,
  useResendDailyReport,
  type DailyReportFilters,
} from '../hooks/queries';

const { Paragraph, Title } = Typography;

const DailyReports: React.FC = () => {
  const [filters, setFilters] = useState<DailyReportFilters>({ page: 1, page_size: 10 });
  const [selectedReport, setSelectedReport] = useState<DailyReport | null>(null);
  const [drawerVisible, setDrawerVisible] = useState(false);
  const { t } = useTranslation();
  const { isDark } = useThemeStore();

  const { data: reportsData, isLoading, refetch } = useDailyReports(filters);
  const generateReport = useGenerateDailyReport();
  const resendReport = useResendDailyReport();

  const handleGenerate = async () => {
    try {
      await generateReport.mutateAsync();
      message.success(t('dailyReports.generateSuccess'));
    } catch {
      message.error(t('common.error'));
    }
  };

  const handleResend = async (id: number) => {
    try {
      await resendReport.mutateAsync(id);
      message.success(t('dailyReports.resendSuccess'));
    } catch {
      message.error(t('common.error'));
    }
  };

  const handlePageChange = (page: number, pageSize: number) => {
    setFilters(prev => ({ ...prev, page, page_size: pageSize }));
  };

  const columns: ColumnsType<DailyReport> = [
    { title: t('dailyReports.reportDate'), dataIndex: 'report_date', key: 'report_date', width: 120, render: (date: string) => dayjs(date).format('YYYY-MM-DD') },
    { title: t('dailyReports.totalCommits'), dataIndex: 'total_commits', key: 'total_commits', width: 100 },
    { title: t('dailyReports.totalAuthors'), dataIndex: 'total_authors', key: 'total_authors', width: 100 },
    { title: t('dailyReports.averageScore'), dataIndex: 'average_score', key: 'average_score', width: 100, render: (score: number) => score?.toFixed(1) || '-' },
    {
      title: t('dailyReports.passRate'), key: 'pass_rate', width: 100,
      render: (_, record) => {
        const total = record.passed_count + record.failed_count;
        if (total === 0) return '-';
        return `${(record.passed_count / total * 100).toFixed(0)}%`;
      },
    },
    {
      title: t('dailyReports.notifyStatus'), key: 'notify_status', width: 100,
      render: (_, record) => {
        if (record.notified_at) return <Tag color="success">{t('dailyReports.sent')}</Tag>;
        if (record.notify_error) return <Tag color="error">{t('dailyReports.failed')}</Tag>;
        return <Tag color="default">{t('dailyReports.pending')}</Tag>;
      },
    },
    { title: t('common.createdAt'), dataIndex: 'created_at', key: 'created_at', width: 160, render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm') },
    {
      title: t('common.actions'), key: 'actions', width: 150,
      render: (_, record) => (
        <Space>
          <Button type="link" size="small" icon={<EyeOutlined />} onClick={() => { setSelectedReport(record); setDrawerVisible(true); }} />
          <Popconfirm title={t('dailyReports.resendConfirm')} onConfirm={() => handleResend(record.id)}>
            <Button type="link" size="small" icon={<SendOutlined />} loading={resendReport.isPending} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const parseJSON = (str: string) => { try { return JSON.parse(str); } catch { return []; } };

  return (
    <Card
      title={t('dailyReports.title')}
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => refetch()}>{t('common.refresh')}</Button>
          <Button type="primary" icon={<PlusOutlined />} loading={generateReport.isPending} onClick={handleGenerate}>{t('dailyReports.generate')}</Button>
        </Space>
      }
    >
      <Table columns={columns} dataSource={reportsData?.items ?? []} rowKey="id" loading={isLoading} scroll={{ x: 800 }}
        pagination={{ current: filters.page, pageSize: filters.page_size, total: reportsData?.total ?? 0, showSizeChanger: true, showTotal: (count) => `${t('common.total')} ${count}`, onChange: handlePageChange }} />

      <Drawer title={t('dailyReports.reportDetail')} placement="right" width={getResponsiveWidth(700)} open={drawerVisible} onClose={() => setDrawerVisible(false)}>
        {selectedReport && (
          <>
            <Descriptions column={2} bordered size="small">
              <Descriptions.Item label={t('dailyReports.reportDate')}>{dayjs(selectedReport.report_date).format('YYYY-MM-DD')}</Descriptions.Item>
              <Descriptions.Item label={t('dailyReports.totalCommits')}>{selectedReport.total_commits}</Descriptions.Item>
              <Descriptions.Item label={t('dailyReports.totalProjects')}>{selectedReport.total_projects}</Descriptions.Item>
              <Descriptions.Item label={t('dailyReports.totalAuthors')}>{selectedReport.total_authors}</Descriptions.Item>
              <Descriptions.Item label={t('dailyReports.averageScore')}>{selectedReport.average_score?.toFixed(1)}</Descriptions.Item>
              <Descriptions.Item label={t('dailyReports.passRate')}>{(() => { const total = selectedReport.passed_count + selectedReport.failed_count; if (total === 0) return '-'; return `${(selectedReport.passed_count / total * 100).toFixed(0)}%`; })()}</Descriptions.Item>
              <Descriptions.Item label={t('dailyReports.passed')}><Tag color="success">{selectedReport.passed_count}</Tag></Descriptions.Item>
              <Descriptions.Item label={t('dailyReports.failed')}><Tag color="error">{selectedReport.failed_count}</Tag></Descriptions.Item>
              <Descriptions.Item label={t('dailyReports.additions')}><span style={{ color: 'green' }}>+{selectedReport.total_additions}</span></Descriptions.Item>
              <Descriptions.Item label={t('dailyReports.deletions')}><span style={{ color: 'red' }}>-{selectedReport.total_deletions}</span></Descriptions.Item>
            </Descriptions>

            <Title level={5} style={{ marginTop: 24 }}>{t('dailyReports.topProjects')}</Title>
            <Table size="small" dataSource={parseJSON(selectedReport.top_projects)} rowKey="name" pagination={false}
              columns={[{ title: t('dailyReports.projectName'), dataIndex: 'name', key: 'name' }, { title: t('dailyReports.commitCount'), dataIndex: 'commit_count', key: 'commit_count' }, { title: t('dailyReports.avgScore'), dataIndex: 'avg_score', key: 'avg_score', render: (v: number) => v?.toFixed(1) }]} />

            <Title level={5} style={{ marginTop: 24 }}>{t('dailyReports.topAuthors')}</Title>
            <Table size="small" dataSource={parseJSON(selectedReport.top_authors)} rowKey="name" pagination={false}
              columns={[{ title: t('dailyReports.authorName'), dataIndex: 'name', key: 'name' }, { title: t('dailyReports.commitCount'), dataIndex: 'commit_count', key: 'commit_count' }, { title: t('dailyReports.avgScore'), dataIndex: 'avg_score', key: 'avg_score', render: (v: number) => v?.toFixed(1) }]} />

            {selectedReport.ai_analysis && (
              <>
                <Title level={5} style={{ marginTop: 24 }}>{t('dailyReports.aiAnalysis')}</Title>
                <Card size="small" style={{ background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)', borderRadius: 8 }}>
                  <div style={{ background: isDark ? '#1e293b' : '#fff', borderRadius: 6, padding: '16px 20px' }} className="ai-analysis-content">
                    <ReactMarkdown
                      components={{
                        h1: ({ children }) => <h2 style={{ fontSize: 18, marginTop: 0, marginBottom: 12, color: isDark ? '#f1f5f9' : '#1a1a1a' }}>{children}</h2>,
                        h2: ({ children }) => <h3 style={{ fontSize: 16, marginTop: 16, marginBottom: 10, color: isDark ? '#f1f5f9' : '#1a1a1a', borderBottom: isDark ? '1px solid #475569' : '1px solid #eee', paddingBottom: 6 }}>{children}</h3>,
                        p: ({ children }) => <p style={{ margin: '8px 0', lineHeight: 1.7, color: isDark ? '#cbd5e1' : '#444' }}>{children}</p>,
                        ul: ({ children }) => <ul style={{ margin: '8px 0', paddingLeft: 20 }}>{children}</ul>,
                        li: ({ children }) => <li style={{ margin: '4px 0', lineHeight: 1.6, color: isDark ? '#cbd5e1' : '#444' }}>{children}</li>,
                        strong: ({ children }) => <strong style={{ color: isDark ? '#f1f5f9' : '#1a1a1a', fontWeight: 600 }}>{children}</strong>,
                        code: ({ children }) => <code style={{ background: isDark ? '#334155' : '#f5f5f5', padding: '2px 6px', borderRadius: 4, fontSize: 13, color: isDark ? '#f472b6' : '#d63384', fontFamily: 'Monaco, Consolas, monospace' }}>{children}</code>,
                      }}
                    >{selectedReport.ai_analysis}</ReactMarkdown>
                  </div>
                </Card>
              </>
            )}

            {selectedReport.notify_error && (
              <>
                <Title level={5} style={{ marginTop: 24 }}>{t('dailyReports.notifyError')}</Title>
                <Paragraph type="danger">{selectedReport.notify_error}</Paragraph>
              </>
            )}
          </>
        )}
      </Drawer>
    </Card>
  );
};

export default DailyReports;
