import React, { useState } from 'react';
import {
  Card,
  Table,
  Space,
  Input,
  Select,
  DatePicker,
  Button,
  Tag,
  Drawer,
  Statistic,
  Row,
  Col,
  message,
} from 'antd';
import {
  SearchOutlined,
  ReloadOutlined,
  UserOutlined,
  CodeOutlined,
  TrophyOutlined,
  FileOutlined,
  RiseOutlined,
  FallOutlined,
  TeamOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  LineChart,
  Line,
  Legend,
  PieChart,
  Pie,
  Cell,
  AreaChart,
  Area,
} from 'recharts';
import dayjs from 'dayjs';
import isoWeek from 'dayjs/plugin/isoWeek';
import { useTranslation } from 'react-i18next';
import { memberApi } from '../services';
import { useMemberStats, useProjects, useTeamOverview, useHeatmap, type MemberStatsFilters } from '../hooks/queries';
import { ContributionHeatmap } from '../components';
import { getResponsiveWidth } from '../hooks';

dayjs.extend(isoWeek);

const { RangePicker } = DatePicker;

interface MemberStats {
  author: string;
  author_email: string;
  commit_count: number;
  avg_score: number;
  max_score: number;
  min_score: number;
  additions: number;
  deletions: number;
  files_changed: number;
  project_count: number;
}

interface MemberDetail {
  author: string;
  author_email: string;
  total_stats: MemberStats;
  project_stats: Array<{ project_id: number; project_name: string; commit_count: number; avg_score: number; additions: number; deletions: number }>;
  trend: Array<{ date: string; commit_count: number; avg_score: number }>;
}



const MemberAnalysis: React.FC = () => {
  const [drawerVisible, setDrawerVisible] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [memberDetail, setMemberDetail] = useState<MemberDetail | null>(null);
  const [selectedAuthor, setSelectedAuthor] = useState<string>('');
  const { t } = useTranslation();

  const [searchName, setSearchName] = useState('');
  const [projectId, setProjectId] = useState<number | undefined>();
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([dayjs().subtract(30, 'day'), dayjs()]);
  const [sortBy, setSortBy] = useState('commit_count');
  const [filters, setFilters] = useState<MemberStatsFilters>({
    page: 1, page_size: 20, sort_by: 'commit_count', sort_order: 'desc',
    start_date: dayjs().subtract(30, 'day').format('YYYY-MM-DD'), end_date: dayjs().format('YYYY-MM-DD'),
  });

  const { data: memberData, isLoading } = useMemberStats(filters);
  const { data: projectsData } = useProjects({ page_size: 100 });
  const { data: overview, isLoading: overviewLoading } = useTeamOverview({
    start_date: dateRange[0].format('YYYY-MM-DD'),
    end_date: dateRange[1].format('YYYY-MM-DD'),
    project_id: projectId,
  });

  const { data: memberHeatmapData, isLoading: memberHeatmapLoading } = useHeatmap({
    start_date: dayjs().subtract(1, 'year').format('YYYY-MM-DD'),
    end_date: dayjs().format('YYYY-MM-DD'),
    author: selectedAuthor,
  });

  const handleSearch = () => {
    const newFilters: MemberStatsFilters = { page: 1, page_size: filters.page_size, sort_by: sortBy, sort_order: 'desc' };
    if (searchName) newFilters.name = searchName;
    if (projectId) newFilters.project_id = projectId;
    if (dateRange) {
      newFilters.start_date = dateRange[0].format('YYYY-MM-DD');
      newFilters.end_date = dateRange[1].format('YYYY-MM-DD');
    }
    setFilters(newFilters);
  };

  const handleReset = () => {
    setSearchName('');
    setProjectId(undefined);
    setDateRange([dayjs().subtract(30, 'day'), dayjs()]);
    setSortBy('commit_count');
    setFilters({
      page: 1, page_size: 20, sort_by: 'commit_count', sort_order: 'desc',
      start_date: dayjs().subtract(30, 'day').format('YYYY-MM-DD'), end_date: dayjs().format('YYYY-MM-DD'),
    });
  };

  const handlePageChange = (page: number, pageSize: number) => {
    setFilters((prev: MemberStatsFilters) => ({ ...prev, page, page_size: pageSize }));
  };

  const handleTableChange = (_pagination: any, _filters: any, sorter: any) => {
    if (sorter.field) {
      setSortBy(sorter.field);
      setFilters((prev: MemberStatsFilters) => ({ ...prev, sort_by: sorter.field }));
    }
  };

  const showMemberDetail = async (author: string) => {
    setSelectedAuthor(author);
    setDrawerVisible(true);
    setDetailLoading(true);
    try {
      const params: any = { author };
      if (dateRange) {
        params.start_date = dateRange[0].format('YYYY-MM-DD');
        params.end_date = dateRange[1].format('YYYY-MM-DD');
      }
      const res = await memberApi.getDetail(params);
      setMemberDetail(res.data);
    } catch (error) {
      message.error(t('common.error'));
    } finally {
      setDetailLoading(false);
    }
  };

  const getScoreColor = (score: number) => {
    if (score >= 80) return 'success';
    if (score >= 60) return 'warning';
    return 'error';
  };

  const columns: ColumnsType<MemberStats> = [
    { title: t('memberAnalysis.author'), dataIndex: 'author', key: 'author', width: 150, render: (author: string) => <a onClick={() => showMemberDetail(author)}>{author}</a> },
    { title: t('memberAnalysis.commitCount'), dataIndex: 'commit_count', key: 'commit_count', width: 100, sorter: true },
    { title: t('memberAnalysis.avgScore'), dataIndex: 'avg_score', key: 'avg_score', width: 100, render: (score: number) => <Tag color={getScoreColor(score)}>{score.toFixed(1)}</Tag>, sorter: true },
    { title: t('memberAnalysis.projectCount'), dataIndex: 'project_count', key: 'project_count', width: 100 },
    { title: t('memberAnalysis.additions'), dataIndex: 'additions', key: 'additions', width: 100, render: (val: number) => <span style={{ color: '#52c41a' }}>+{val}</span>, sorter: true },
    { title: t('memberAnalysis.deletions'), dataIndex: 'deletions', key: 'deletions', width: 100, render: (val: number) => <span style={{ color: '#ff4d4f' }}>-{val}</span>, sorter: true },
    { title: t('memberAnalysis.filesChanged'), dataIndex: 'files_changed', key: 'files_changed', width: 100 },
  ];

  // Score distribution for pie chart
  const scoreDistributionData = overview ? [
    { name: t('memberAnalysis.excellent'), value: overview.score_distribution.excellent, color: '#52c41a' },
    { name: t('memberAnalysis.good'), value: overview.score_distribution.good, color: '#faad14' },
    { name: t('memberAnalysis.needWork'), value: overview.score_distribution.need_work, color: '#ff4d4f' },
  ].filter(d => d.value > 0) : [];

  return (
    <>
      {/* Overview Stats Cards */}
      <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
        <Col xs={24} sm={12} md={6}>
          <Card size="small" style={{ background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)' }}>
            <Statistic title={<span style={{ color: 'rgba(255,255,255,0.85)' }}>{t('memberAnalysis.totalMembers')}</span>} value={overview?.total_members ?? 0} prefix={<TeamOutlined />} valueStyle={{ color: '#fff' }} loading={overviewLoading} />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card size="small" style={{ background: 'linear-gradient(135deg, #f093fb 0%, #f5576c 100%)' }}>
            <Statistic title={<span style={{ color: 'rgba(255,255,255,0.85)' }}>{t('memberAnalysis.totalCommits')}</span>} value={overview?.total_commits ?? 0} prefix={<CodeOutlined />} valueStyle={{ color: '#fff' }} loading={overviewLoading} />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card size="small" style={{ background: 'linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)' }}>
            <Statistic title={<span style={{ color: 'rgba(255,255,255,0.85)' }}>{t('memberAnalysis.teamAvgScore')}</span>} value={overview?.avg_score ?? 0} precision={1} prefix={<TrophyOutlined />} valueStyle={{ color: '#fff' }} loading={overviewLoading} />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card size="small" style={{ background: 'linear-gradient(135deg, #43e97b 0%, #38f9d7 100%)' }}>
            <Statistic title={<span style={{ color: 'rgba(255,255,255,0.85)' }}>{t('memberAnalysis.codeChanges')}</span>} value={`+${overview?.total_additions ?? 0} / -${overview?.total_deletions ?? 0}`} prefix={<FileOutlined />} valueStyle={{ color: '#fff', fontSize: 18 }} loading={overviewLoading} />
          </Card>
        </Col>
      </Row>

      {/* Charts Row */}
      <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
        {/* Team Trend Chart */}
        <Col xs={24} lg={16}>
          <Card title={t('memberAnalysis.teamTrend')} size="small" loading={overviewLoading}>
            <ResponsiveContainer width="100%" height={250}>
              <AreaChart data={overview?.trend ?? []}>
                <defs>
                  <linearGradient id="colorCommits" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#1890ff" stopOpacity={0.8} />
                    <stop offset="95%" stopColor="#1890ff" stopOpacity={0.1} />
                  </linearGradient>
                  <linearGradient id="colorScore" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#52c41a" stopOpacity={0.8} />
                    <stop offset="95%" stopColor="#52c41a" stopOpacity={0.1} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="date" tick={{ fontSize: 10 }} />
                <YAxis yAxisId="left" />
                <YAxis yAxisId="right" orientation="right" domain={[0, 100]} />
                <Tooltip />
                <Legend />
                <Area yAxisId="left" type="monotone" dataKey="commit_count" stroke="#1890ff" fillOpacity={1} fill="url(#colorCommits)" name={t('memberAnalysis.commitCount')} />
                <Line yAxisId="right" type="monotone" dataKey="avg_score" stroke="#52c41a" strokeWidth={2} dot={false} name={t('memberAnalysis.avgScore')} />
              </AreaChart>
            </ResponsiveContainer>
          </Card>
        </Col>

        {/* Score Distribution Pie */}
        <Col xs={24} lg={8}>
          <Card title={t('memberAnalysis.scoreDistribution')} size="small" loading={overviewLoading}>
            <ResponsiveContainer width="100%" height={250}>
              <PieChart>
                <Pie data={scoreDistributionData} cx="50%" cy="50%" innerRadius={50} outerRadius={80} paddingAngle={5} dataKey="value" label={({ name, percent }) => `${name} ${((percent ?? 0) * 100).toFixed(0)}%`}>
                  {scoreDistributionData.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={entry.color} />
                  ))}
                </Pie>
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          </Card>
        </Col>
      </Row>

      {/* Top Members Comparison */}
      <Card title={t('memberAnalysis.topMembersComparison')} size="small" style={{ marginBottom: 16 }} loading={overviewLoading}>
        <ResponsiveContainer width="100%" height={300}>
          <BarChart data={overview?.top_members ?? []} layout="vertical" margin={{ left: 80 }}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis type="number" />
            <YAxis dataKey="author" type="category" tick={{ fontSize: 11 }} width={80} />
            <Tooltip />
            <Legend />
            <Bar dataKey="commit_count" fill="#1890ff" name={t('memberAnalysis.commitCount')} />
            <Bar dataKey="avg_score" fill="#52c41a" name={t('memberAnalysis.avgScore')} />
          </BarChart>
        </ResponsiveContainer>
      </Card>

      {/* Search and Table */}
      <Card>
        <Space style={{ marginBottom: 16 }} wrap>
          <Input placeholder={t('memberAnalysis.searchAuthor')} style={{ width: 150 }} value={searchName} onChange={(e) => setSearchName(e.target.value)} onPressEnter={handleSearch} prefix={<UserOutlined />} />
          <Select placeholder={t('memberAnalysis.selectProject')} allowClear style={{ width: 180 }} value={projectId} onChange={setProjectId} options={projectsData?.items?.map((p) => ({ value: p.id, label: p.name })) ?? []} />
          <RangePicker
            value={dateRange}
            onChange={(dates) => setDateRange(dates as [dayjs.Dayjs, dayjs.Dayjs])}
            presets={[
              { label: t('memberAnalysis.last7Days'), value: [dayjs().subtract(7, 'day'), dayjs()] },
              { label: t('memberAnalysis.last30Days'), value: [dayjs().subtract(30, 'day'), dayjs()] },
              { label: t('memberAnalysis.last90Days'), value: [dayjs().subtract(90, 'day'), dayjs()] },
              { label: t('memberAnalysis.thisMonth'), value: [dayjs().startOf('month'), dayjs()] },
              { label: t('memberAnalysis.lastMonth'), value: [dayjs().subtract(1, 'month').startOf('month'), dayjs().subtract(1, 'month').endOf('month')] },
              { label: t('memberAnalysis.thisYear'), value: [dayjs().startOf('year'), dayjs()] },
            ]}
          />
          <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>{t('common.search')}</Button>
          <Button icon={<ReloadOutlined />} onClick={handleReset}>{t('common.reset')}</Button>
        </Space>

        <Table columns={columns} dataSource={memberData?.items ?? []} rowKey="author" loading={isLoading} onChange={handleTableChange} scroll={{ x: 800 }}
          pagination={{ current: filters.page, pageSize: filters.page_size, total: memberData?.total ?? 0, showSizeChanger: true, showTotal: (total) => `${t('common.total')} ${total}`, onChange: handlePageChange }} />
      </Card>

      <Drawer title={memberDetail?.author || t('memberAnalysis.memberDetail')} width={getResponsiveWidth(800)} open={drawerVisible} onClose={() => { setDrawerVisible(false); setSelectedAuthor(''); }} loading={detailLoading}>
        {memberDetail && (
          <>
            {/* Contribution Heatmap */}
            <Card title={t('memberAnalysis.contributionHeatmap', 'Contribution Heatmap')} size="small" style={{ marginBottom: 16 }}>
              <ContributionHeatmap
                data={memberHeatmapData?.data ?? []}
                totalCount={memberHeatmapData?.total_count ?? 0}
                maxCount={memberHeatmapData?.max_count ?? 0}
                loading={memberHeatmapLoading}
              />
            </Card>
            <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
              <Col span={6}><Card size="small"><Statistic title={t('memberAnalysis.commitCount')} value={memberDetail.total_stats.commit_count} prefix={<CodeOutlined />} /></Card></Col>
              <Col span={6}><Card size="small"><Statistic title={t('memberAnalysis.avgScore')} value={memberDetail.total_stats.avg_score} precision={1} prefix={<TrophyOutlined />} valueStyle={{ color: memberDetail.total_stats.avg_score >= 80 ? '#52c41a' : memberDetail.total_stats.avg_score >= 60 ? '#faad14' : '#ff4d4f' }} /></Card></Col>
              <Col span={6}><Card size="small"><Statistic title={t('memberAnalysis.additions')} value={memberDetail.total_stats.additions} prefix={<RiseOutlined />} valueStyle={{ color: '#52c41a' }} /></Card></Col>
              <Col span={6}><Card size="small"><Statistic title={t('memberAnalysis.deletions')} value={memberDetail.total_stats.deletions} prefix={<FallOutlined />} valueStyle={{ color: '#ff4d4f' }} /></Card></Col>
            </Row>
            <Card title={t('memberAnalysis.commitTrend')} size="small" style={{ marginBottom: 16 }}>
              <ResponsiveContainer width="100%" height={200}>
                <LineChart data={memberDetail.trend}>
                  <CartesianGrid strokeDasharray="3 3" /><XAxis dataKey="date" tick={{ fontSize: 10 }} /><YAxis yAxisId="left" /><YAxis yAxisId="right" orientation="right" domain={[0, 100]} /><Tooltip /><Legend />
                  <Line yAxisId="left" type="monotone" dataKey="commit_count" stroke="#1890ff" name={t('memberAnalysis.commitCount')} />
                  <Line yAxisId="right" type="monotone" dataKey="avg_score" stroke="#52c41a" name={t('memberAnalysis.avgScore')} />
                </LineChart>
              </ResponsiveContainer>
            </Card>
            <Card title={t('memberAnalysis.projectDistribution')} size="small">
              <ResponsiveContainer width="100%" height={250}>
                <BarChart data={memberDetail.project_stats}>
                  <CartesianGrid strokeDasharray="3 3" /><XAxis dataKey="project_name" tick={{ fontSize: 10 }} /><YAxis /><Tooltip /><Legend />
                  <Bar dataKey="commit_count" fill="#1890ff" name={t('memberAnalysis.commitCount')} />
                </BarChart>
              </ResponsiveContainer>
            </Card>
          </>
        )}
      </Drawer>
    </>
  );
};

export default MemberAnalysis;
