import React, { useState, useEffect } from 'react';
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
} from 'recharts';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import { memberApi, projectApi } from '../services';
import type { Project } from '../types';

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
  project_stats: Array<{
    project_id: number;
    project_name: string;
    commit_count: number;
    avg_score: number;
    additions: number;
    deletions: number;
  }>;
  trend: Array<{
    date: string;
    commit_count: number;
    avg_score: number;
  }>;
}

const MemberAnalysis: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<MemberStats[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [drawerVisible, setDrawerVisible] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [memberDetail, setMemberDetail] = useState<MemberDetail | null>(null);
  const [projects, setProjects] = useState<Project[]>([]);
  const { t } = useTranslation();

  const [searchName, setSearchName] = useState('');
  const [projectId, setProjectId] = useState<number | undefined>();
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([
    dayjs().subtract(30, 'day'),
    dayjs(),
  ]);
  const [sortBy, setSortBy] = useState('commit_count');

  const fetchData = async () => {
    setLoading(true);
    try {
      const params: any = {
        page,
        page_size: pageSize,
        sort_by: sortBy,
        sort_order: 'desc',
      };
      if (searchName) params.name = searchName;
      if (projectId) params.project_id = projectId;
      if (dateRange) {
        params.start_date = dateRange[0].format('YYYY-MM-DD');
        params.end_date = dateRange[1].format('YYYY-MM-DD');
      }
      const res = await memberApi.list(params);
      setData(res.data.items || []);
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
      setProjects(res.data.items || []);
    } catch (error) {}
  };

  useEffect(() => {
    fetchData();
  }, [page, pageSize, sortBy]);

  useEffect(() => {
    fetchProjects();
  }, []);

  const handleSearch = () => {
    setPage(1);
    fetchData();
  };

  const handleReset = () => {
    setSearchName('');
    setProjectId(undefined);
    setDateRange([dayjs().subtract(30, 'day'), dayjs()]);
    setSortBy('commit_count');
    setPage(1);
    setTimeout(fetchData, 0);
  };

  const showMemberDetail = async (author: string) => {
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
    {
      title: t('memberAnalysis.author'),
      dataIndex: 'author',
      key: 'author',
      width: 150,
      render: (author: string) => (
        <a onClick={() => showMemberDetail(author)}>{author}</a>
      ),
    },
    {
      title: t('memberAnalysis.commitCount'),
      dataIndex: 'commit_count',
      key: 'commit_count',
      width: 100,
      sorter: true,
    },
    {
      title: t('memberAnalysis.avgScore'),
      dataIndex: 'avg_score',
      key: 'avg_score',
      width: 100,
      render: (score: number) => (
        <Tag color={getScoreColor(score)}>{score.toFixed(1)}</Tag>
      ),
      sorter: true,
    },
    {
      title: t('memberAnalysis.projectCount'),
      dataIndex: 'project_count',
      key: 'project_count',
      width: 100,
    },
    {
      title: t('memberAnalysis.additions'),
      dataIndex: 'additions',
      key: 'additions',
      width: 100,
      render: (val: number) => <span style={{ color: '#52c41a' }}>+{val}</span>,
      sorter: true,
    },
    {
      title: t('memberAnalysis.deletions'),
      dataIndex: 'deletions',
      key: 'deletions',
      width: 100,
      render: (val: number) => <span style={{ color: '#ff4d4f' }}>-{val}</span>,
      sorter: true,
    },
    {
      title: t('memberAnalysis.filesChanged'),
      dataIndex: 'files_changed',
      key: 'files_changed',
      width: 100,
    },
  ];

  const handleTableChange = (_pagination: any, _filters: any, sorter: any) => {
    if (sorter.field) {
      setSortBy(sorter.field);
    }
  };

  return (
    <>
      <Card>
        <Space style={{ marginBottom: 16 }} wrap>
          <Input
            placeholder={t('memberAnalysis.searchAuthor')}
            style={{ width: 150 }}
            value={searchName}
            onChange={(e) => setSearchName(e.target.value)}
            onPressEnter={handleSearch}
            prefix={<UserOutlined />}
          />
          <Select
            placeholder={t('memberAnalysis.selectProject')}
            allowClear
            style={{ width: 180 }}
            value={projectId}
            onChange={setProjectId}
            options={projects.map((p) => ({ value: p.id, label: p.name }))}
          />
          <RangePicker
            value={dateRange}
            onChange={(dates) =>
              setDateRange(dates as [dayjs.Dayjs, dayjs.Dayjs])
            }
          />
          <Button
            type="primary"
            icon={<SearchOutlined />}
            onClick={handleSearch}
          >
            {t('common.search')}
          </Button>
          <Button icon={<ReloadOutlined />} onClick={handleReset}>
            {t('common.reset')}
          </Button>
        </Space>

        <Table
          columns={columns}
          dataSource={data}
          rowKey="author"
          loading={loading}
          onChange={handleTableChange}
          scroll={{ x: 800 }}
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
        title={memberDetail?.author || t('memberAnalysis.memberDetail')}
        width={800}
        open={drawerVisible}
        onClose={() => setDrawerVisible(false)}
        loading={detailLoading}
      >
        {memberDetail && (
          <>
            <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
              <Col span={6}>
                <Card size="small">
                  <Statistic
                    title={t('memberAnalysis.commitCount')}
                    value={memberDetail.total_stats.commit_count}
                    prefix={<CodeOutlined />}
                  />
                </Card>
              </Col>
              <Col span={6}>
                <Card size="small">
                  <Statistic
                    title={t('memberAnalysis.avgScore')}
                    value={memberDetail.total_stats.avg_score}
                    precision={1}
                    prefix={<TrophyOutlined />}
                    valueStyle={{
                      color:
                        memberDetail.total_stats.avg_score >= 80
                          ? '#52c41a'
                          : memberDetail.total_stats.avg_score >= 60
                          ? '#faad14'
                          : '#ff4d4f',
                    }}
                  />
                </Card>
              </Col>
              <Col span={6}>
                <Card size="small">
                  <Statistic
                    title={t('memberAnalysis.additions')}
                    value={memberDetail.total_stats.additions}
                    prefix={<FileOutlined />}
                    valueStyle={{ color: '#52c41a' }}
                  />
                </Card>
              </Col>
              <Col span={6}>
                <Card size="small">
                  <Statistic
                    title={t('memberAnalysis.deletions')}
                    value={memberDetail.total_stats.deletions}
                    prefix={<FileOutlined />}
                    valueStyle={{ color: '#ff4d4f' }}
                  />
                </Card>
              </Col>
            </Row>

            <Card
              title={t('memberAnalysis.commitTrend')}
              size="small"
              style={{ marginBottom: 16 }}
            >
              <ResponsiveContainer width="100%" height={200}>
                <LineChart data={memberDetail.trend}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="date" tick={{ fontSize: 10 }} />
                  <YAxis yAxisId="left" />
                  <YAxis yAxisId="right" orientation="right" domain={[0, 100]} />
                  <Tooltip />
                  <Legend />
                  <Line
                    yAxisId="left"
                    type="monotone"
                    dataKey="commit_count"
                    stroke="#1890ff"
                    name={t('memberAnalysis.commitCount')}
                  />
                  <Line
                    yAxisId="right"
                    type="monotone"
                    dataKey="avg_score"
                    stroke="#52c41a"
                    name={t('memberAnalysis.avgScore')}
                  />
                </LineChart>
              </ResponsiveContainer>
            </Card>

            <Card
              title={t('memberAnalysis.projectDistribution')}
              size="small"
            >
              <ResponsiveContainer width="100%" height={250}>
                <BarChart data={memberDetail.project_stats}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="project_name" tick={{ fontSize: 10 }} />
                  <YAxis />
                  <Tooltip />
                  <Legend />
                  <Bar
                    dataKey="commit_count"
                    fill="#1890ff"
                    name={t('memberAnalysis.commitCount')}
                  />
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
