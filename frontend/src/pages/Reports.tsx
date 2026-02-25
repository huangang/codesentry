import React, { useState } from 'react';
import { Card, Select, Statistic, Row, Col, Table, Space, Tag, Spin } from 'antd';
import { ArrowUpOutlined, ArrowDownOutlined, MinusOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Line } from 'recharts';
import { reportApi, type ReportResponse } from '../services';
import { useProjects } from '../hooks/queries';
import { ResponsiveContainer, LineChart, XAxis, YAxis, Tooltip, CartesianGrid } from 'recharts';

const Reports: React.FC = () => {
    const { t } = useTranslation();
    const [period, setPeriod] = useState<string>('weekly');
    const [projectId, setProjectId] = useState<number | undefined>();
    const { data: projectsData } = useProjects({ page_size: 100 });

    const { data: report, isLoading } = useQuery<ReportResponse>({
        queryKey: ['report', period, projectId],
        queryFn: async () => {
            const res = await reportApi.getReport({ period, project_id: projectId });
            return res.data;
        },
    });

    const getDelta = (current: number, previous: number) => {
        if (previous === 0) return current > 0 ? 100 : 0;
        return ((current - previous) / previous * 100);
    };

    const renderDelta = (current: number, previous: number, inverse = false) => {
        const delta = getDelta(current, previous);
        const isPositive = delta > 0;
        const color = inverse ? (isPositive ? '#ff4d4f' : '#52c41a') : (isPositive ? '#52c41a' : '#ff4d4f');
        if (delta === 0) return <span style={{ color: '#999' }}><MinusOutlined /> 0%</span>;
        return (
            <span style={{ color, fontSize: 13 }}>
                {isPositive ? <ArrowUpOutlined /> : <ArrowDownOutlined />} {Math.abs(delta).toFixed(1)}%
            </span>
        );
    };

    const rankColumns = [
        { title: '#', key: 'rank', width: 50, render: (_: any, __: any, i: number) => i + 1 },
        { title: t('reviewLogs.author', 'Author'), dataIndex: 'author', key: 'author' },
        { title: t('reports.reviews', 'Reviews'), dataIndex: 'review_count', key: 'review_count', sorter: (a: any, b: any) => a.review_count - b.review_count },
        {
            title: t('reports.avgScore', 'Avg Score'), dataIndex: 'avg_score', key: 'avg_score',
            render: (v: number) => <Tag color={v >= 80 ? 'green' : v >= 60 ? 'orange' : 'red'}>{v?.toFixed(1)}</Tag>,
            sorter: (a: any, b: any) => a.avg_score - b.avg_score,
        },
        {
            title: t('reports.changes', 'Changes'), key: 'changes',
            render: (_: any, r: any) => <><span style={{ color: '#52c41a' }}>+{r.total_additions}</span> / <span style={{ color: '#ff4d4f' }}>-{r.total_deletions}</span></>,
        },
    ];

    if (isLoading) return <div style={{ textAlign: 'center', padding: 80 }}><Spin size="large" /></div>;

    const cur = report?.current;
    const prev = report?.previous;

    return (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
            {/* Filters */}
            <Card styles={{ body: { padding: '12px 16px' } }}>
                <Space>
                    <Select value={period} onChange={setPeriod} style={{ width: 120 }}
                        options={[
                            { value: 'weekly', label: t('reports.weekly', 'Weekly') },
                            { value: 'monthly', label: t('reports.monthly', 'Monthly') },
                        ]}
                    />
                    <Select
                        placeholder={t('reports.allProjects', 'All Projects')}
                        allowClear showSearch optionFilterProp="label"
                        style={{ width: 180 }}
                        value={projectId}
                        onChange={setProjectId}
                        options={projectsData?.items?.map(p => ({ value: p.id, label: p.name })) ?? []}
                    />
                </Space>
            </Card>

            {/* Stats Cards */}
            {cur && prev && (
                <Row gutter={[16, 16]}>
                    <Col xs={12} sm={8} lg={4}>
                        <Card><Statistic title={t('reports.totalReviews', 'Total Reviews')} value={cur.total_reviews} suffix={renderDelta(cur.total_reviews, prev.total_reviews)} /></Card>
                    </Col>
                    <Col xs={12} sm={8} lg={4}>
                        <Card><Statistic title={t('reports.completed', 'Completed')} value={cur.completed} suffix={renderDelta(cur.completed, prev.completed)} /></Card>
                    </Col>
                    <Col xs={12} sm={8} lg={4}>
                        <Card><Statistic title={t('reports.failed', 'Failed')} value={cur.failed} suffix={renderDelta(cur.failed, prev.failed, true)} /></Card>
                    </Col>
                    <Col xs={12} sm={8} lg={4}>
                        <Card><Statistic title={t('reports.avgScore', 'Avg Score')} value={cur.avg_score?.toFixed(1)} suffix={renderDelta(cur.avg_score, prev.avg_score)} /></Card>
                    </Col>
                    <Col xs={12} sm={8} lg={4}>
                        <Card><Statistic title={t('reports.activeAuthors', 'Active Authors')} value={cur.active_authors} suffix={renderDelta(cur.active_authors, prev.active_authors)} /></Card>
                    </Col>
                    <Col xs={12} sm={8} lg={4}>
                        <Card>
                            <Statistic
                                title={t('reports.codeChanges', 'Code Changes')}
                                value={cur.total_additions + cur.total_deletions}
                                suffix={renderDelta(cur.total_additions + cur.total_deletions, prev.total_additions + prev.total_deletions)}
                            />
                        </Card>
                    </Col>
                </Row>
            )}

            {/* Trend Chart */}
            {report?.trend && report.trend.length > 0 && (
                <Card title={t('reports.trend', '14-Day Trend')}>
                    <ResponsiveContainer width="100%" height={300}>
                        <LineChart data={report.trend}>
                            <CartesianGrid strokeDasharray="3 3" />
                            <XAxis dataKey="date" tickFormatter={(v) => v.slice(5)} />
                            <YAxis yAxisId="left" />
                            <YAxis yAxisId="right" orientation="right" />
                            <Tooltip />
                            <Line yAxisId="left" type="monotone" dataKey="reviews" stroke="#1890ff" name={t('reports.reviews', 'Reviews')} strokeWidth={2} />
                            <Line yAxisId="right" type="monotone" dataKey="avg_score" stroke="#52c41a" name={t('reports.avgScore', 'Avg Score')} strokeWidth={2} />
                        </LineChart>
                    </ResponsiveContainer>
                </Card>
            )}

            {/* Author Rankings */}
            {report?.rankings && (
                <Card title={t('reports.authorRankings', 'Author Rankings')}>
                    <Table
                        dataSource={report.rankings}
                        columns={rankColumns}
                        rowKey="author"
                        pagination={false}
                        size="small"
                    />
                </Card>
            )}
        </Space>
    );
};

export default Reports;
