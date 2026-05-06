import React, { useState, useEffect, useCallback } from 'react';
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
  Modal,
  Spin,
  Empty,
  Tooltip as AntTooltip,
  List,
  Collapse,
  Typography,
} from 'antd';
import {
  SearchOutlined,
  ReloadOutlined,
  PlusOutlined,
  DeleteOutlined,
  UserOutlined,
  CodeOutlined,
  TrophyOutlined,
  FileOutlined,
  RiseOutlined,
  FallOutlined,
  TeamOutlined,
  FileTextOutlined,
  CopyOutlined,
  HistoryOutlined,
  SyncOutlined,
  CheckCircleOutlined,
  CalendarOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import type { RangePickerProps } from 'antd/es/date-picker';
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
import type { Dayjs } from 'dayjs';
import isoWeek from 'dayjs/plugin/isoWeek';
import { useTranslation } from 'react-i18next';
import { memberApi, memberAliasApi, type MemberAlias, type WorkSummaryResponse, type WorkSummaryListItem, type WorkPeriodSummaryResponse } from '../services';
import { useMemberStats, useProjects, useTeamOverview, useHeatmap, type MemberStatsFilters } from '../hooks/queries';
import { ContributionHeatmap } from '../components';
import MarkdownContent from '../components/MarkdownContent';
import { getResponsiveWidth } from '../hooks';
import { useAuthStore } from '../stores/authStore';

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
  const { isAdmin } = useAuthStore();
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

  // Work Summary state
  const [summaryModalVisible, setSummaryModalVisible] = useState(false);
  const [summaryLoading, setSummaryLoading] = useState(false);
  const [summaryData, setSummaryData] = useState<WorkSummaryResponse | null>(null);
  const [summaryDate, setSummaryDate] = useState<dayjs.Dayjs>(dayjs());

  // Work Period Summary state
  const [periodSummaryModalVisible, setPeriodSummaryModalVisible] = useState(false);
  const [periodSummaryLoading, setPeriodSummaryLoading] = useState(false);
  const [periodSummaryData, setPeriodSummaryData] = useState<WorkPeriodSummaryResponse | null>(null);
  const [periodSummaryRange, setPeriodSummaryRange] = useState<[Dayjs, Dayjs]>([dayjs().subtract(6, 'day'), dayjs()]);

  // History state
  const [historyModalVisible, setHistoryModalVisible] = useState(false);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [historyItems, setHistoryItems] = useState<WorkSummaryListItem[]>([]);
  const [historyTotal, setHistoryTotal] = useState(0);
  const [historyPage, setHistoryPage] = useState(1);

  // Alias management state
  const [aliasModalVisible, setAliasModalVisible] = useState(false);
  const [aliasLoading, setAliasLoading] = useState(false);
  const [aliases, setAliases] = useState<MemberAlias[]>([]);
  const [aliasInput, setAliasInput] = useState('');
  const [canonicalInput, setCanonicalInput] = useState('');

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

  // Auto-load cached summary when date changes or modal opens
  const loadCachedSummary = useCallback(async (author: string, date: dayjs.Dayjs) => {
    setSummaryLoading(true);
    try {
      const res = await memberApi.getWorkSummary({
        author,
        date: date.format('YYYY-MM-DD'),
      });
      setSummaryData(res.data);
    } catch {
      setSummaryData(null);
    } finally {
      setSummaryLoading(false);
    }
  }, []);

  // When summaryDate changes and modal is visible, reload cached
  useEffect(() => {
    if (summaryModalVisible && selectedAuthor) {
      loadCachedSummary(selectedAuthor, summaryDate);
    }
  }, [summaryDate, summaryModalVisible, selectedAuthor, loadCachedSummary]);

  const handleOpenSummaryModal = () => {
    if (!selectedAuthor) return;
    setSummaryModalVisible(true);
    // loadCachedSummary will be triggered by useEffect
  };

  const handleGenerateWorkSummary = async () => {
    if (!selectedAuthor) return;
    setSummaryLoading(true);
    try {
      const res = await memberApi.generateWorkSummary({
        author: selectedAuthor,
        date: summaryDate.format('YYYY-MM-DD'),
      });
      setSummaryData(res.data);
      message.success(t('common.success'));
    } catch {
      message.error(t('common.error'));
    } finally {
      setSummaryLoading(false);
    }
  };

  const handleCopySummary = () => {
    if (summaryData?.summary) {
      navigator.clipboard.writeText(summaryData.summary);
      message.success(t('memberAnalysis.workSummaryCopied'));
    }
  };

  const loadCachedPeriodSummary = useCallback(async (author: string, range: [Dayjs, Dayjs]) => {
    setPeriodSummaryLoading(true);
    try {
      const res = await memberApi.getWorkPeriodSummary({
        author,
        start_date: range[0].format('YYYY-MM-DD'),
        end_date: range[1].format('YYYY-MM-DD'),
      });
      setPeriodSummaryData(res.data);
    } catch {
      setPeriodSummaryData(null);
    } finally {
      setPeriodSummaryLoading(false);
    }
  }, []);

  useEffect(() => {
    if (periodSummaryModalVisible && selectedAuthor) {
      loadCachedPeriodSummary(selectedAuthor, periodSummaryRange);
    }
  }, [periodSummaryRange, periodSummaryModalVisible, selectedAuthor, loadCachedPeriodSummary]);

  const handleOpenPeriodSummaryModal = () => {
    if (!selectedAuthor) return;
    setPeriodSummaryRange(dateRange);
    setPeriodSummaryModalVisible(true);
  };

  const handlePeriodRangeChange: RangePickerProps['onChange'] = (dates) => {
    if (dates?.[0] && dates?.[1]) {
      setPeriodSummaryRange([dates[0], dates[1]]);
    }
  };

  const handleGenerateWorkPeriodSummary = async () => {
    if (!selectedAuthor) return;
    setPeriodSummaryLoading(true);
    try {
      const res = await memberApi.generateWorkPeriodSummary({
        author: selectedAuthor,
        start_date: periodSummaryRange[0].format('YYYY-MM-DD'),
        end_date: periodSummaryRange[1].format('YYYY-MM-DD'),
      });
      setPeriodSummaryData(res.data);
      message.success(t('common.success'));
    } catch {
      message.error(t('common.error'));
    } finally {
      setPeriodSummaryLoading(false);
    }
  };

  const handleCopyPeriodSummary = () => {
    if (periodSummaryData?.summary) {
      navigator.clipboard.writeText(periodSummaryData.summary);
      message.success(t('memberAnalysis.workSummaryCopied'));
    }
  };

  // History handlers
  const loadHistory = useCallback(async (author: string, page: number) => {
    setHistoryLoading(true);
    try {
      const res = await memberApi.listWorkSummaries({ author, page, page_size: 10 });
      setHistoryItems(res.data.items);
      setHistoryTotal(res.data.total);
      setHistoryPage(res.data.page);
    } catch {
      message.error(t('common.error'));
    } finally {
      setHistoryLoading(false);
    }
  }, [t]);

  const handleOpenHistory = () => {
    if (!selectedAuthor) return;
    setHistoryModalVisible(true);
    loadHistory(selectedAuthor, 1);
  };

  const handleViewHistoryItem = async (item: WorkSummaryListItem) => {
    setHistoryModalVisible(false);
    setSummaryDate(dayjs(item.date));
    setSummaryModalVisible(true);
    // useEffect will load the cached summary for that date
  };

  const loadAliases = useCallback(async () => {
    setAliasLoading(true);
    try {
      const res = await memberAliasApi.list();
      setAliases(res.data);
    } catch {
      message.error(t('common.error'));
    } finally {
      setAliasLoading(false);
    }
  }, [t]);

  const handleOpenAliasModal = () => {
    setAliasModalVisible(true);
    loadAliases();
  };

  const handleCreateAlias = async () => {
    const alias = aliasInput.trim();
    const canonicalName = canonicalInput.trim();
    if (!alias || !canonicalName) {
      message.warning(t('memberAnalysis.aliasFormRequired'));
      return;
    }
    if (alias === canonicalName) {
      message.warning(t('memberAnalysis.aliasCannotSame'));
      return;
    }
    setAliasLoading(true);
    try {
      await memberAliasApi.create({ alias, canonical_name: canonicalName });
      message.success(t('memberAnalysis.aliasCreateSuccess'));
      setAliasInput('');
      setCanonicalInput('');
      await loadAliases();
      handleSearch();
    } catch {
      message.error(t('common.error'));
    } finally {
      setAliasLoading(false);
    }
  };

  const handleDeleteAlias = async (id: number) => {
    setAliasLoading(true);
    try {
      await memberAliasApi.delete(id);
      message.success(t('memberAnalysis.aliasDeleteSuccess'));
      await loadAliases();
      handleSearch();
    } catch {
      message.error(t('common.error'));
    } finally {
      setAliasLoading(false);
    }
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
          {isAdmin && (
            <Button icon={<PlusOutlined />} onClick={handleOpenAliasModal}>
              {t('memberAnalysis.manageAliases')}
            </Button>
          )}
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
            {/* Work Summary Button */}
            <div style={{ marginBottom: 16 }}>
              <Space wrap>
                <DatePicker
                  value={summaryDate}
                  onChange={(d) => d && setSummaryDate(d)}
                  allowClear={false}
                  style={{ width: 140 }}
                  presets={[
                    { label: t('memberAnalysis.workSummaryToday'), value: dayjs() },
                  ]}
                />
                <Button
                  type="primary"
                  icon={<FileTextOutlined />}
                  onClick={handleOpenSummaryModal}
                >
                  {t('memberAnalysis.workSummary')}
                </Button>
                <Button
                  icon={<HistoryOutlined />}
                  onClick={handleOpenHistory}
                >
                  {t('memberAnalysis.workSummaryViewHistory')}
                </Button>
                <Button
                  icon={<CalendarOutlined />}
                  onClick={handleOpenPeriodSummaryModal}
                >
                  {t('memberAnalysis.workPeriodSummary')}
                </Button>
              </Space>
            </div>
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

      {/* Work Summary Modal */}
      <Modal
        title={
          <Space>
            <FileTextOutlined />
            {t('memberAnalysis.workSummary')} - {summaryData?.date || summaryDate.format('YYYY-MM-DD')}
            {summaryData?.cached && (
              <Tag icon={<CheckCircleOutlined />} color="success">{t('memberAnalysis.workSummaryCached')}</Tag>
            )}
          </Space>
        }
        open={summaryModalVisible}
        onCancel={() => { setSummaryModalVisible(false); setSummaryData(null); }}
        width={getResponsiveWidth(720)}
        footer={[
          <Button
            key="regenerate"
            icon={<SyncOutlined />}
            onClick={handleGenerateWorkSummary}
            loading={summaryLoading}
          >
            {t('memberAnalysis.workSummaryRegenerate')}
          </Button>,
          <Button key="copy" icon={<CopyOutlined />} onClick={handleCopySummary} disabled={!summaryData?.summary}>
            {t('memberAnalysis.workSummaryCopy')}
          </Button>,
          <Button key="close" type="primary" onClick={() => { setSummaryModalVisible(false); setSummaryData(null); }}>
            {t('common.close')}
          </Button>,
        ]}
      >
        {summaryLoading ? (
          <div style={{ textAlign: 'center', padding: 40 }}>
            <Spin size="large" />
            <p style={{ marginTop: 16, color: '#888' }}>{t('memberAnalysis.workSummaryGenerating')}</p>
          </div>
        ) : summaryData?.summary ? (
          <div>
            <MarkdownContent content={summaryData.summary} />
            <div style={{ marginTop: 12, display: 'flex', justifyContent: 'space-between', color: '#999', fontSize: 12 }}>
              {summaryData.updated_at && (
                <span>{t('memberAnalysis.workSummaryLastUpdated')}: {dayjs(summaryData.updated_at).format('YYYY-MM-DD HH:mm')}</span>
              )}
              {summaryData.ai_model_used && (
                <AntTooltip title="AI Model">
                  Powered by {summaryData.ai_model_used}
                </AntTooltip>
              )}
            </div>
          </div>
        ) : (
          <div style={{ textAlign: 'center', padding: 40 }}>
            <Empty description={t('memberAnalysis.workSummaryEmpty')} />
            <Button
              type="primary"
              icon={<FileTextOutlined />}
              onClick={handleGenerateWorkSummary}
              loading={summaryLoading}
              style={{ marginTop: 16 }}
            >
              {t('memberAnalysis.generateWorkSummary')}
            </Button>
          </div>
        )}
      </Modal>

      {/* Work Period Summary Modal */}
      <Modal
        title={
          <Space>
            <CalendarOutlined />
            {t('memberAnalysis.workPeriodSummary')} - {periodSummaryData?.start_date || periodSummaryRange[0].format('YYYY-MM-DD')} ~ {periodSummaryData?.end_date || periodSummaryRange[1].format('YYYY-MM-DD')}
            {periodSummaryData?.cached && (
              <Tag icon={<CheckCircleOutlined />} color="success">{t('memberAnalysis.workSummaryCached')}</Tag>
            )}
          </Space>
        }
        open={periodSummaryModalVisible}
        onCancel={() => { setPeriodSummaryModalVisible(false); setPeriodSummaryData(null); }}
        width={getResponsiveWidth(840)}
        footer={[
          <Button
            key="regenerate"
            icon={<SyncOutlined />}
            onClick={handleGenerateWorkPeriodSummary}
            loading={periodSummaryLoading}
          >
            {t('memberAnalysis.workSummaryRegenerate')}
          </Button>,
          <Button key="copy" icon={<CopyOutlined />} onClick={handleCopyPeriodSummary} disabled={!periodSummaryData?.summary}>
            {t('memberAnalysis.workSummaryCopy')}
          </Button>,
          <Button key="close" type="primary" onClick={() => { setPeriodSummaryModalVisible(false); setPeriodSummaryData(null); }}>
            {t('common.close')}
          </Button>,
        ]}
      >
        <Space wrap style={{ marginBottom: 16 }}>
          <RangePicker
            value={periodSummaryRange}
            onChange={handlePeriodRangeChange}
            allowClear={false}
            presets={[
              { label: t('memberAnalysis.last7Days'), value: [dayjs().subtract(6, 'day'), dayjs()] },
              { label: t('memberAnalysis.last30Days'), value: [dayjs().subtract(29, 'day'), dayjs()] },
              { label: t('memberAnalysis.thisMonth'), value: [dayjs().startOf('month'), dayjs()] },
              { label: t('memberAnalysis.lastMonth'), value: [dayjs().subtract(1, 'month').startOf('month'), dayjs().subtract(1, 'month').endOf('month')] },
            ]}
          />
        </Space>
        {periodSummaryLoading ? (
          <div style={{ textAlign: 'center', padding: 40 }}>
            <Spin size="large" />
            <Typography.Text type="secondary" style={{ display: 'block', marginTop: 16 }}>
              {t('memberAnalysis.workPeriodSummaryGenerating')}
            </Typography.Text>
          </div>
        ) : periodSummaryData?.summary ? (
          <div>
            <MarkdownContent content={periodSummaryData.summary} />
            {(periodSummaryData.raw_data.daily_summaries?.length ?? 0) > 0 && (
              <Collapse
                size="small"
                style={{ marginTop: 16 }}
                items={[
                  {
                    key: 'daily-summaries',
                    label: t('memberAnalysis.workPeriodDailySummaries'),
                    children: (
                      <List
                        size="small"
                        dataSource={periodSummaryData.raw_data.daily_summaries ?? []}
                        renderItem={(item) => (
                          <List.Item>
                            <List.Item.Meta
                              title={
                                <Space wrap size="small">
                                  <span>{item.date}</span>
                                  <Tag>{t('memberAnalysis.commitCount')}: {item.commit_count}</Tag>
                                  <Tag>+{item.additions} / -{item.deletions}</Tag>
                                </Space>
                              }
                              description={<MarkdownContent content={item.summary} />}
                            />
                          </List.Item>
                        )}
                      />
                    ),
                  },
                ]}
              />
            )}
            <div style={{ marginTop: 12, display: 'flex', justifyContent: 'space-between', fontSize: 12 }}>
              {periodSummaryData.updated_at && (
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                  {t('memberAnalysis.workSummaryLastUpdated')}: {dayjs(periodSummaryData.updated_at).format('YYYY-MM-DD HH:mm')}
                </Typography.Text>
              )}
              {periodSummaryData.ai_model_used && (
                <AntTooltip title="AI Model">
                  <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                    Powered by {periodSummaryData.ai_model_used}
                  </Typography.Text>
                </AntTooltip>
              )}
            </div>
          </div>
        ) : (
          <div style={{ textAlign: 'center', padding: 40 }}>
            <Empty description={t('memberAnalysis.workPeriodSummaryEmpty')} />
            <Button
              type="primary"
              icon={<CalendarOutlined />}
              onClick={handleGenerateWorkPeriodSummary}
              loading={periodSummaryLoading}
              style={{ marginTop: 16 }}
            >
              {t('memberAnalysis.generateWorkPeriodSummary')}
            </Button>
          </div>
        )}
      </Modal>

      {/* Work Summary History Modal */}
      <Modal
        title={<Space><HistoryOutlined />{t('memberAnalysis.workSummaryHistory')}</Space>}
        open={historyModalVisible}
        onCancel={() => setHistoryModalVisible(false)}
        width={getResponsiveWidth(520)}
        footer={null}
      >
        <List
          loading={historyLoading}
          dataSource={historyItems}
          locale={{ emptyText: t('memberAnalysis.workSummaryNoHistory') }}
          pagination={{
            current: historyPage,
            total: historyTotal,
            pageSize: 10,
            size: 'small',
            onChange: (page) => selectedAuthor && loadHistory(selectedAuthor, page),
          }}
          renderItem={(item: WorkSummaryListItem) => (
            <List.Item
              style={{ cursor: 'pointer' }}
              onClick={() => handleViewHistoryItem(item)}
              actions={[
                <Button key="view" type="link" size="small">{t('common.view')}</Button>,
              ]}
            >
              <List.Item.Meta
                title={item.date}
                description={
                  <Space size="small">
                    {item.ai_model_used && <Tag>{item.ai_model_used}</Tag>}
                    <span style={{ color: '#999', fontSize: 12 }}>
                      {dayjs(item.updated_at).format('YYYY-MM-DD HH:mm')}
                    </span>
                  </Space>
                }
              />
            </List.Item>
          )}
        />
      </Modal>

      {/* Member Alias Management Modal */}
      <Modal
        title={t('memberAnalysis.manageAliases')}
        open={aliasModalVisible}
        onCancel={() => setAliasModalVisible(false)}
        width={getResponsiveWidth(720)}
        footer={null}
      >
        <Space wrap style={{ marginBottom: 16, width: '100%' }}>
          <Input
            placeholder={t('memberAnalysis.aliasInput')}
            value={aliasInput}
            onChange={(e) => setAliasInput(e.target.value)}
            style={{ width: 220 }}
          />
          <Input
            placeholder={t('memberAnalysis.canonicalInput')}
            value={canonicalInput}
            onChange={(e) => setCanonicalInput(e.target.value)}
            style={{ width: 220 }}
          />
          <Button type="primary" onClick={handleCreateAlias} loading={aliasLoading}>
            {t('memberAnalysis.addAlias')}
          </Button>
        </Space>

        <Table<MemberAlias>
          rowKey="id"
          loading={aliasLoading}
          dataSource={aliases}
          pagination={false}
          scroll={{ x: 600 }}
          columns={[
            { title: t('memberAnalysis.alias'), dataIndex: 'alias', key: 'alias' },
            { title: t('memberAnalysis.canonicalName'), dataIndex: 'canonical_name', key: 'canonical_name' },
            {
              title: t('common.actions'),
              key: 'actions',
              width: 120,
              render: (_, record) => (
                <Button
                  danger
                  size="small"
                  icon={<DeleteOutlined />}
                  onClick={() => {
                    Modal.confirm({
                      title: t('memberAnalysis.aliasDeleteConfirm'),
                      onOk: () => handleDeleteAlias(record.id),
                    });
                  }}
                >
                  {t('common.delete')}
                </Button>
              ),
            },
          ]}
        />
      </Modal>
    </>
  );
};

export default MemberAnalysis;
