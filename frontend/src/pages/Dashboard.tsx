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
        <Col xs={24} lg={12}>
          <Card title={t('dashboard.projectCommits', 'Project Commits')} bordered={false} hoverable>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={data?.project_stats || []}>
                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#e2e8f0" />
                <XAxis dataKey="project_name" tick={{ fontSize: 12, fill: '#64748b' }} axisLine={false} tickLine={false} />
                <YAxis axisLine={false} tickLine={false} tick={{ fill: '#64748b' }} />
                <Tooltip
                  cursor={{ fill: '#f1f5f9' }}
                  contentStyle={{ borderRadius: 8, border: 'none', boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1)' }}
                />
                <Bar dataKey="commit_count" fill="#3b82f6" radius={[4, 4, 0, 0]} name={t('dashboard.commits', 'Commits')} />
              </BarChart>
            </ResponsiveContainer>
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title={t('dashboard.authorCommits', 'Author Commits')} bordered={false} hoverable>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={data?.author_stats || []}>
                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#e2e8f0" />
                <XAxis dataKey="author" tick={{ fontSize: 12, fill: '#64748b' }} axisLine={false} tickLine={false} />
                <YAxis axisLine={false} tickLine={false} tick={{ fill: '#64748b' }} />
                <Tooltip
                  cursor={{ fill: '#f1f5f9' }}
                  contentStyle={{ borderRadius: 8, border: 'none', boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1)' }}
                />
                <Bar dataKey="commit_count" fill="#8b5cf6" radius={[4, 4, 0, 0]} name={t('dashboard.commits', 'Commits')} />
              </BarChart>
            </ResponsiveContainer>
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title={t('dashboard.projectAvgScore', 'Project Average Score')} bordered={false} hoverable>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={data?.project_stats || []}>
                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#e2e8f0" />
                <XAxis dataKey="project_name" tick={{ fontSize: 12, fill: '#64748b' }} axisLine={false} tickLine={false} />
                <YAxis domain={[0, 100]} axisLine={false} tickLine={false} tick={{ fill: '#64748b' }} />
                <Tooltip
                  cursor={{ fill: '#f1f5f9' }}
                  contentStyle={{ borderRadius: 8, border: 'none', boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1)' }}
                />
                <Bar dataKey="avg_score" fill="#10b981" radius={[4, 4, 0, 0]} name={t('dashboard.avgScore')} />
              </BarChart>
            </ResponsiveContainer>
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title={t('dashboard.authorAvgScore', 'Author Average Score')} bordered={false} hoverable>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={data?.author_stats || []}>
                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#e2e8f0" />
                <XAxis dataKey="author" tick={{ fontSize: 12, fill: '#64748b' }} axisLine={false} tickLine={false} />
                <YAxis domain={[0, 100]} axisLine={false} tickLine={false} tick={{ fill: '#64748b' }} />
                <Tooltip
                  cursor={{ fill: '#f1f5f9' }}
                  contentStyle={{ borderRadius: 8, border: 'none', boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1)' }}
                />
                <Bar dataKey="avg_score" fill="#f59e0b" radius={[4, 4, 0, 0]} name={t('dashboard.avgScore')} />
              </BarChart>
            </ResponsiveContainer>
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title={t('dashboard.projectCodeChanges', 'Project Code Changes')} bordered={false} hoverable>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={data?.project_stats || []}>
                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#e2e8f0" />
                <XAxis dataKey="project_name" tick={{ fontSize: 12, fill: '#64748b' }} axisLine={false} tickLine={false} />
                <YAxis axisLine={false} tickLine={false} tick={{ fill: '#64748b' }} />
                <Tooltip
                  cursor={{ fill: '#f1f5f9' }}
                  contentStyle={{ borderRadius: 8, border: 'none', boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1)' }}
                />
                <Legend />
                <Bar dataKey="additions" stackId="a" fill="#10b981" radius={[0, 0, 4, 4]} name={t('dashboard.additions', 'Additions')} />
                <Bar dataKey="deletions" stackId="a" fill="#ef4444" radius={[4, 4, 0, 0]} name={t('dashboard.deletions', 'Deletions')} />
              </BarChart>
            </ResponsiveContainer>
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title={t('dashboard.authorCodeChanges', 'Author Code Changes')} bordered={false} hoverable>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={data?.author_stats || []}>
                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#e2e8f0" />
                <XAxis dataKey="author" tick={{ fontSize: 12, fill: '#64748b' }} axisLine={false} tickLine={false} />
                <YAxis axisLine={false} tickLine={false} tick={{ fill: '#64748b' }} />
                <Tooltip
                  cursor={{ fill: '#f1f5f9' }}
                  contentStyle={{ borderRadius: 8, border: 'none', boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1)' }}
                />
                <Legend />
                <Bar dataKey="additions" stackId="a" fill="#10b981" radius={[0, 0, 4, 4]} name={t('dashboard.additions', 'Additions')} />
                <Bar dataKey="deletions" stackId="a" fill="#ef4444" radius={[4, 4, 0, 0]} name={t('dashboard.deletions', 'Deletions')} />
              </BarChart>
            </ResponsiveContainer>
          </Card>
        </Col>
      </Row>
    </Spin>
  );
};


export default Dashboard;
