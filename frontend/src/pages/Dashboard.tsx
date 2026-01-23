import React, { useState, useEffect, useCallback } from 'react';
import { Card, Row, Col, Statistic, Radio, DatePicker, Spin, Space, Button, Modal } from 'antd';
import {
  ProjectOutlined,
  TeamOutlined,
  CodeOutlined,
  TrophyOutlined,
  FullscreenOutlined,
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
import { dashboardApi } from '../services';
import type { DashboardResponse } from '../types';

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
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<DashboardResponse | null>(null);
  const [dateRange, setDateRange] = useState<string>('week');
  const [customRange, setCustomRange] = useState<[dayjs.Dayjs, dayjs.Dayjs] | null>(null);
  const [expandedChart, setExpandedChart] = useState<ChartType | null>(null);
  const [expandedData, setExpandedData] = useState<DashboardResponse | null>(null);
  const [expandedLoading, setExpandedLoading] = useState(false);
  const { t } = useTranslation();

  const getDateParams = useCallback(() => {
    let startDate: string | undefined;
    let endDate: string | undefined;

    const now = dayjs();
    switch (dateRange) {
      case 'week':
        startDate = now.subtract(7, 'day').format('YYYY-MM-DD');
        break;
      case 'twoWeeks':
        startDate = now.subtract(14, 'day').format('YYYY-MM-DD');
        break;
      case 'month':
        startDate = now.subtract(30, 'day').format('YYYY-MM-DD');
        break;
      case 'custom':
        if (customRange) {
          startDate = customRange[0].format('YYYY-MM-DD');
          endDate = customRange[1].format('YYYY-MM-DD');
        }
        break;
    }
    endDate = endDate || now.format('YYYY-MM-DD');
    return { startDate, endDate };
  }, [dateRange, customRange]);

  const fetchData = async () => {
    setLoading(true);
    try {
      const { startDate, endDate } = getDateParams();
      const res = await dashboardApi.getStats({
        start_date: startDate,
        end_date: endDate,
        project_limit: 10,
        author_limit: 10,
      });
      setData(res.data);
    } catch (error) {
      console.error('Failed to fetch dashboard data:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchExpandedData = async (chartType: ChartType) => {
    setExpandedLoading(true);
    try {
      const { startDate, endDate } = getDateParams();
      const isProjectChart = chartType.startsWith('project');
      const res = await dashboardApi.getStats({
        start_date: startDate,
        end_date: endDate,
        project_limit: isProjectChart ? 50 : 10,
        author_limit: isProjectChart ? 10 : 50,
      });
      setExpandedData(res.data);
    } catch (error) {
      console.error('Failed to fetch expanded data:', error);
    } finally {
      setExpandedLoading(false);
    }
  };

  const handleExpand = (chartType: ChartType) => {
    setExpandedChart(chartType);
    fetchExpandedData(chartType);
  };

  const handleCloseModal = () => {
    setExpandedChart(null);
    setExpandedData(null);
  };

  useEffect(() => {
    fetchData();
  }, [dateRange, customRange]);

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

  const renderChart = (config: ChartConfig, chartData: DashboardResponse | null, height: number) => (
    <ResponsiveContainer width="100%" height={height}>
      <BarChart data={chartData?.[config.dataKey] || []}>
        <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#e2e8f0" />
        <XAxis dataKey={config.xKey} tick={{ fontSize: 12, fill: '#64748b' }} axisLine={false} tickLine={false} />
        <YAxis domain={config.yDomain} axisLine={false} tickLine={false} tick={{ fill: '#64748b' }} />
        <Tooltip
          cursor={{ fill: '#f1f5f9' }}
          contentStyle={{ borderRadius: 8, border: 'none', boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1)' }}
        />
        {config.showLegend && <Legend />}
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
    <Spin spinning={loading}>
      <div style={{ marginBottom: 24 }}>
        <Space>
          <Radio.Group value={dateRange} onChange={(e) => setDateRange(e.target.value)} buttonStyle="solid">
            {dateRangeOptions.map(opt => (
              <Radio.Button key={opt.value} value={opt.value}>{opt.label}</Radio.Button>
            ))}
          </Radio.Group>
          {dateRange === 'custom' && (
            <RangePicker
              value={customRange}
              onChange={(dates) => setCustomRange(dates as [dayjs.Dayjs, dayjs.Dayjs])}
            />
          )}
        </Space>
      </div>

      <Row gutter={[24, 24]} style={{ marginBottom: 24 }}>
        {statsCards.map((card, index) => (
          <Col xs={24} sm={12} lg={6} key={index}>
            <Card hoverable bordered={false} style={{ height: '100%', boxShadow: '0 1px 2px 0 rgba(0, 0, 0, 0.05)' }}>
              <Statistic
                title={<span style={{ color: '#64748b', fontSize: 14 }}>{card.title}</span>}
                value={card.value}
                prefix={
                  <div style={{
                    backgroundColor: card.bg,
                    padding: 8,
                    borderRadius: 8,
                    display: 'flex',
                    marginRight: 12
                  }}>
                    {React.cloneElement(card.icon, { style: { color: card.color, fontSize: 20 } })}
                  </div>
                }
                valueStyle={{ color: '#0f172a', fontWeight: 600, fontSize: 24 }}
              />
            </Card>
          </Col>
        ))}
      </Row>

      <Row gutter={[24, 24]}>
        {chartConfigs.map((config) => (
          <Col xs={24} lg={12} key={config.key}>
            <Card
              title={config.title}
              bordered={false}
              hoverable
              extra={
                <Button
                  type="text"
                  size="small"
                  icon={<FullscreenOutlined />}
                  onClick={() => handleExpand(config.key)}
                >
                  {t('dashboard.expand', 'Expand')}
                </Button>
              }
            >
              {renderChart(config, data, 300)}
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
