import React, { useState, useEffect } from 'react';
import { Card, Row, Col, Statistic, Radio, DatePicker, Spin, Space } from 'antd';
import {
  ProjectOutlined,
  TeamOutlined,
  CodeOutlined,
  TrophyOutlined,
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

const Dashboard: React.FC = () => {
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<DashboardResponse | null>(null);
  const [dateRange, setDateRange] = useState<string>('week');
  const [customRange, setCustomRange] = useState<[dayjs.Dayjs, dayjs.Dayjs] | null>(null);
  const { t } = useTranslation();

  const fetchData = async () => {
    setLoading(true);
    try {
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

      const res = await dashboardApi.getStats(startDate, endDate);
      setData(res.data);
    } catch (error) {
      console.error('Failed to fetch dashboard data:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, [dateRange, customRange]);

  const statsCards = [
    { title: t('dashboard.totalProjects'), value: data?.stats.active_projects || 0, icon: <ProjectOutlined />, color: '#1890ff' },
    { title: t('dashboard.totalReviews'), value: data?.stats.contributors || 0, icon: <TeamOutlined />, color: '#52c41a' },
    { title: t('dashboard.todayReviews'), value: data?.stats.total_commits || 0, icon: <CodeOutlined />, color: '#722ed1' },
    { title: t('dashboard.avgScore'), value: data?.stats.average_score?.toFixed(2) || '0', icon: <TrophyOutlined />, color: '#fa8c16' },
  ];

  const dateRangeOptions = [
    { value: 'week', label: t('dashboard.lastWeek', 'Last Week') },
    { value: 'twoWeeks', label: t('dashboard.lastTwoWeeks', 'Last 2 Weeks') },
    { value: 'month', label: t('dashboard.lastMonth', 'Last Month') },
    { value: 'custom', label: t('dashboard.custom', 'Custom') },
  ];

  return (
    <Spin spinning={loading}>
      <div style={{ marginBottom: 24 }}>
        <Space>
          <Radio.Group value={dateRange} onChange={(e) => setDateRange(e.target.value)}>
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

      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        {statsCards.map((card, index) => (
          <Col xs={24} sm={12} lg={6} key={index}>
            <Card>
              <Statistic
                title={card.title}
                value={card.value}
                prefix={React.cloneElement(card.icon, { style: { color: card.color } })}
                valueStyle={{ color: card.color }}
              />
            </Card>
          </Col>
        ))}
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card title={t('dashboard.projectCommits', 'Project Commits')} size="small">
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={data?.project_stats || []}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="project_name" tick={{ fontSize: 12 }} />
                <YAxis />
                <Tooltip />
                <Bar dataKey="commit_count" fill="#1890ff" name={t('dashboard.commits', 'Commits')} />
              </BarChart>
            </ResponsiveContainer>
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title={t('dashboard.authorCommits', 'Author Commits')} size="small">
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={data?.author_stats || []}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="author" tick={{ fontSize: 12 }} />
                <YAxis />
                <Tooltip />
                <Bar dataKey="commit_count" fill="#1890ff" name={t('dashboard.commits', 'Commits')} />
              </BarChart>
            </ResponsiveContainer>
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title={t('dashboard.projectAvgScore', 'Project Average Score')} size="small">
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={data?.project_stats || []}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="project_name" tick={{ fontSize: 12 }} />
                <YAxis domain={[0, 100]} />
                <Tooltip />
                <Bar dataKey="avg_score" fill="#52c41a" name={t('dashboard.avgScore')} />
              </BarChart>
            </ResponsiveContainer>
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title={t('dashboard.authorAvgScore', 'Author Average Score')} size="small">
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={data?.author_stats || []}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="author" tick={{ fontSize: 12 }} />
                <YAxis domain={[0, 100]} />
                <Tooltip />
                <Bar dataKey="avg_score" fill="#52c41a" name={t('dashboard.avgScore')} />
              </BarChart>
            </ResponsiveContainer>
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title={t('dashboard.projectCodeChanges', 'Project Code Changes')} size="small">
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={data?.project_stats || []}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="project_name" tick={{ fontSize: 12 }} />
                <YAxis />
                <Tooltip />
                <Legend />
                <Bar dataKey="additions" stackId="a" fill="#52c41a" name={t('dashboard.additions', 'Additions')} />
                <Bar dataKey="deletions" stackId="a" fill="#ff4d4f" name={t('dashboard.deletions', 'Deletions')} />
              </BarChart>
            </ResponsiveContainer>
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title={t('dashboard.authorCodeChanges', 'Author Code Changes')} size="small">
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={data?.author_stats || []}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="author" tick={{ fontSize: 12 }} />
                <YAxis />
                <Tooltip />
                <Legend />
                <Bar dataKey="additions" stackId="a" fill="#52c41a" name={t('dashboard.additions', 'Additions')} />
                <Bar dataKey="deletions" stackId="a" fill="#ff4d4f" name={t('dashboard.deletions', 'Deletions')} />
              </BarChart>
            </ResponsiveContainer>
          </Card>
        </Col>
      </Row>
    </Spin>
  );
};

export default Dashboard;
