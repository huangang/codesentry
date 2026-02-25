import React, { useState, useMemo, useCallback } from 'react';
import { Card, Row, Col, Statistic, Radio, DatePicker, Spin, Space, Button, Modal } from 'antd';
import {
  ProjectOutlined,
  TeamOutlined,
  CodeOutlined,
  TrophyOutlined,
  FullscreenOutlined,
  RobotOutlined,
  ThunderboltOutlined,
  DashboardOutlined,
  CheckCircleOutlined,
  CopyOutlined,
} from '@ant-design/icons';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import { useDashboardStats, useAIUsageStats, type DashboardFilters } from '../hooks/queries';
import type { DashboardResponse } from '../types';
import { useAuthStore } from '../stores/authStore';

const { RangePicker } = DatePicker;

type ChartType = 'projectCommits' | 'authorCommits' | 'projectAvgScore' | 'authorAvgScore' | 'projectCodeChanges' | 'authorCodeChanges';

interface ChartConfig {
  key: ChartType;
  title: string;
  dataKey: 'project_stats' | 'author_stats';
  xKey: string;
  bars: Array<{ dataKey: string; fill: string; name: string; stackId?: string; radius?: [number, number, number, number] }>;
  yDomain?: [number, number];
  showLegend?: boolean;
}

const Dashboard: React.FC = () => {
  const [dateRange, setDateRange] = useState<string>('week');
  const [customRange, setCustomRange] = useState<[dayjs.Dayjs, dayjs.Dayjs] | null>(null);
  const [expandedChart, setExpandedChart] = useState<ChartType | null>(null);
  const { t } = useTranslation();

  const getDateParams = useCallback((): DashboardFilters => {
    let start_date: string | undefined;
    let end_date: string | undefined;

    const now = dayjs();
    switch (dateRange) {
      case 'week':
        start_date = now.subtract(7, 'day').format('YYYY-MM-DD');
        break;
      case 'twoWeeks':
        start_date = now.subtract(14, 'day').format('YYYY-MM-DD');
        break;
      case 'month':
        start_date = now.subtract(30, 'day').format('YYYY-MM-DD');
        break;
      case 'custom':
        if (customRange) {
          start_date = customRange[0].format('YYYY-MM-DD');
          end_date = customRange[1].format('YYYY-MM-DD');
        }
        break;
    }
    end_date = end_date || now.format('YYYY-MM-DD');
    return { start_date, end_date, project_limit: 10, author_limit: 10 };
  }, [dateRange, customRange]);

  const filters = useMemo(() => getDateParams(), [getDateParams]);
  const { data, isLoading } = useDashboardStats(filters);
  const { isAdmin } = useAuthStore();
  const { data: aiUsageData } = useAIUsageStats({
    start_date: filters.start_date,
    end_date: filters.end_date,
  });

  const expandedFilters = useMemo((): DashboardFilters => {
    if (!expandedChart) return {};
    const isProjectChart = expandedChart.startsWith('project');
    return {
      ...filters,
      project_limit: isProjectChart ? 50 : 10,
      author_limit: isProjectChart ? 10 : 50,
    };
  }, [expandedChart, filters]);

  const { data: expandedData, isLoading: expandedLoading } = useDashboardStats(
    expandedFilters,
  );

  const handleExpand = (chartType: ChartType) => {
    setExpandedChart(chartType);
  };

  const handleCloseModal = () => {
    setExpandedChart(null);
  };

  const statsCards = [
    { title: t('dashboard.totalProjects'), value: data?.stats.active_projects || 0, icon: <ProjectOutlined />, color: '#06b6d4', bg: 'rgba(6,182,212,0.1)' },
    { title: t('dashboard.totalReviews'), value: data?.stats.contributors || 0, icon: <TeamOutlined />, color: '#8b5cf6', bg: 'rgba(139,92,246,0.1)' },
    { title: t('dashboard.todayReviews'), value: data?.stats.total_commits || 0, icon: <CodeOutlined />, color: '#10b981', bg: 'rgba(16,185,129,0.1)' },
    { title: t('dashboard.avgScore'), value: data?.stats.average_score?.toFixed(2) || '0', icon: <TrophyOutlined />, color: '#f59e0b', bg: 'rgba(245,158,11,0.1)' },
  ];

  const dateRangeOptions = [
    { value: 'week', label: t('dashboard.lastWeek', 'Last Week') },
    { value: 'twoWeeks', label: t('dashboard.lastTwoWeeks', 'Last 2 Weeks') },
    { value: 'month', label: t('dashboard.lastMonth', 'Last Month') },
    { value: 'custom', label: t('dashboard.custom', 'Custom') },
  ];

  const chartConfigs: ChartConfig[] = [
    {
      key: 'projectCommits',
      title: t('dashboard.projectCommits', 'Project Commits'),
      dataKey: 'project_stats',
      xKey: 'project_name',
      bars: [{ dataKey: 'commit_count', fill: '#3b82f6', name: t('dashboard.commits', 'Commits'), radius: [4, 4, 0, 0] }],
    },
    {
      key: 'authorCommits',
      title: t('dashboard.authorCommits', 'Author Commits'),
      dataKey: 'author_stats',
      xKey: 'author',
      bars: [{ dataKey: 'commit_count', fill: '#8b5cf6', name: t('dashboard.commits', 'Commits'), radius: [4, 4, 0, 0] }],
    },
    {
      key: 'projectAvgScore',
      title: t('dashboard.projectAvgScore', 'Project Average Score'),
      dataKey: 'project_stats',
      xKey: 'project_name',
      bars: [{ dataKey: 'avg_score', fill: '#10b981', name: t('dashboard.avgScore'), radius: [4, 4, 0, 0] }],
      yDomain: [0, 100],
    },
    {
      key: 'authorAvgScore',
      title: t('dashboard.authorAvgScore', 'Author Average Score'),
      dataKey: 'author_stats',
      xKey: 'author',
      bars: [{ dataKey: 'avg_score', fill: '#f59e0b', name: t('dashboard.avgScore'), radius: [4, 4, 0, 0] }],
      yDomain: [0, 100],
    },
    {
      key: 'projectCodeChanges',
      title: t('dashboard.projectCodeChanges', 'Project Code Changes'),
      dataKey: 'project_stats',
      xKey: 'project_name',
      bars: [
        { dataKey: 'additions', fill: '#10b981', name: t('dashboard.additions', 'Additions'), stackId: 'a', radius: [0, 0, 4, 4] },
        { dataKey: 'deletions', fill: '#ef4444', name: t('dashboard.deletions', 'Deletions'), stackId: 'a', radius: [4, 4, 0, 0] },
      ],
      showLegend: true,
    },
    {
      key: 'authorCodeChanges',
      title: t('dashboard.authorCodeChanges', 'Author Code Changes'),
      dataKey: 'author_stats',
      xKey: 'author',
      bars: [
        { dataKey: 'additions', fill: '#10b981', name: t('dashboard.additions', 'Additions'), stackId: 'a', radius: [0, 0, 4, 4] },
        { dataKey: 'deletions', fill: '#ef4444', name: t('dashboard.deletions', 'Deletions'), stackId: 'a', radius: [4, 4, 0, 0] },
      ],
      showLegend: true,
    },
  ];

  const renderChart = (config: ChartConfig, chartData: DashboardResponse | null | undefined, height: number) => (
    <ResponsiveContainer width="100%" height={height}>
      <BarChart data={chartData?.[config.dataKey] || []} margin={{ top: 5, right: 5, left: -15, bottom: 5 }}>
        <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#e2e8f0" />
        <XAxis
          dataKey={config.xKey}
          tick={{ fontSize: 10, fill: '#64748b' }}
          axisLine={false}
          tickLine={false}
          interval={0}
          angle={-45}
          textAnchor="end"
          height={60}
        />
        <YAxis domain={config.yDomain} axisLine={false} tickLine={false} tick={{ fill: '#64748b', fontSize: 10 }} width={35} />
        <Tooltip
          cursor={{ fill: '#f1f5f9' }}
          contentStyle={{ borderRadius: 8, border: 'none', boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1)', fontSize: 12 }}
        />
        {config.showLegend && <Legend wrapperStyle={{ fontSize: 12 }} />}
        {config.bars.map((bar) => (
          <Bar
            key={bar.dataKey}
            dataKey={bar.dataKey}
            fill={bar.fill}
            name={bar.name}
            stackId={bar.stackId}
            radius={bar.radius}
          />
        ))}
      </BarChart>
    </ResponsiveContainer>
  );

  const currentExpandedConfig = chartConfigs.find((c) => c.key === expandedChart);

  return (
    <Spin spinning={isLoading}>
      <div style={{ marginBottom: 24 }}>
        <Space wrap>
          <Radio.Group value={dateRange} onChange={(e) => setDateRange(e.target.value)} buttonStyle="solid" size="middle">
            {dateRangeOptions.map(opt => (
              <Radio.Button key={opt.value} value={opt.value}>{opt.label}</Radio.Button>
            ))}
          </Radio.Group>
          {dateRange === 'custom' && (
            <RangePicker
              value={customRange}
              onChange={(dates) => setCustomRange(dates as [dayjs.Dayjs, dayjs.Dayjs])}
              style={{ width: '100%', maxWidth: 280 }}
            />
          )}
        </Space>
      </div>

      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        {statsCards.map((card, index) => (
          <Col xs={12} sm={12} lg={6} key={index}>
            <Card
              hoverable
              bordered={false}
              style={{ height: '100%', boxShadow: '0 1px 2px 0 rgba(0, 0, 0, 0.05)' }}
              className="dashboard-stats-card"
              styles={{ body: { padding: '16px 12px' } }}
            >
              <Statistic
                title={<span style={{ color: '#64748b', fontSize: 12 }}>{card.title}</span>}
                value={card.value}
                prefix={
                  <div style={{
                    backgroundColor: card.bg,
                    padding: 6,
                    borderRadius: 8,
                    display: 'flex',
                    marginRight: 8
                  }}>
                    {React.cloneElement(card.icon, { style: { color: card.color, fontSize: 16 } })}
                  </div>
                }
                valueStyle={{ color: '#0f172a', fontWeight: 600, fontSize: 20 }}
              />
            </Card>
          </Col>
        ))}
      </Row>

      {isAdmin && aiUsageData && (
        <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
          <Col span={24}>
            <div style={{ fontSize: 14, fontWeight: 600, color: '#0f172a', marginBottom: 8 }}>{t('aiUsage.title')}</div>
          </Col>
          <Col xs={24} sm={24} lg={24}>
            <Row gutter={[16, 16]}>
              {[
                { title: t('aiUsage.totalCalls'), value: aiUsageData.total_calls || 0, icon: <RobotOutlined />, color: '#6366f1', bg: 'rgba(99,102,241,0.1)' },
                { title: t('aiUsage.totalTokens'), value: (aiUsageData.total_tokens || 0).toLocaleString(), icon: <ThunderboltOutlined />, color: '#ec4899', bg: 'rgba(236,72,153,0.1)' },
                { title: t('aiUsage.avgLatency'), value: `${Math.round(aiUsageData.avg_latency_ms || 0)}${t('aiUsage.ms')}`, icon: <DashboardOutlined />, color: '#14b8a6', bg: 'rgba(20,184,166,0.1)' },
                { title: t('aiUsage.successRate'), value: `${(aiUsageData.success_rate || 0).toFixed(1)}%`, icon: <CheckCircleOutlined />, color: '#22c55e', bg: 'rgba(34,197,94,0.1)' },
                { title: t('aiUsage.cacheHits'), value: aiUsageData.cache_hits || 0, icon: <CopyOutlined />, color: '#f97316', bg: 'rgba(249,115,22,0.1)' },
              ].map((card, index) => (
                <Col xs={12} sm={8} lg={4} key={`ai-${index}`}>
                  <Card
                    hoverable
                    bordered={false}
                    style={{ height: '100%', boxShadow: '0 1px 2px 0 rgba(0, 0, 0, 0.05)' }}
                    className="dashboard-stats-card"
                    styles={{ body: { padding: '16px 12px' } }}
                  >
                    <Statistic
                      title={<span style={{ color: '#64748b', fontSize: 12 }}>{card.title}</span>}
                      value={card.value}
                      prefix={
                        <div style={{
                          backgroundColor: card.bg,
                          padding: 6,
                          borderRadius: 8,
                          display: 'flex',
                          marginRight: 8
                        }}>
                          {React.cloneElement(card.icon, { style: { color: card.color, fontSize: 16 } })}
                        </div>
                      }
                      valueStyle={{ color: '#0f172a', fontWeight: 600, fontSize: 20 }}
                    />
                  </Card>
                </Col>
              ))}
            </Row>
          </Col>
        </Row>
      )}

      <Row gutter={[16, 16]}>
        {chartConfigs.map((config) => (
          <Col xs={24} lg={12} key={config.key}>
            <Card
              title={<span style={{ fontSize: 14 }}>{config.title}</span>}
              bordered={false}
              hoverable
              styles={{ body: { padding: '12px 8px' } }}
              extra={
                <Button
                  type="text"
                  size="small"
                  icon={<FullscreenOutlined />}
                  onClick={() => handleExpand(config.key)}
                  className="chart-expand-btn"
                >
                  <span className="hide-on-mobile">{t('dashboard.expand', 'Expand')}</span>
                </Button>
              }
            >
              {renderChart(config, data, 250)}
            </Card>
          </Col>
        ))}
      </Row>

      <Modal
        title={currentExpandedConfig?.title}
        open={!!expandedChart}
        onCancel={handleCloseModal}
        footer={null}
        width="90vw"
        style={{ top: 20 }}
        styles={{ body: { height: 'calc(80vh - 55px)', padding: '24px' } }}
      >
        <Spin spinning={expandedLoading}>
          {currentExpandedConfig && renderChart(currentExpandedConfig, expandedData, window.innerHeight * 0.7)}
        </Spin>
      </Modal>
    </Spin>
  );
};


export default Dashboard;
