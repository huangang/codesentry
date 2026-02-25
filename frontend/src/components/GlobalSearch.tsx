import React, { useState, useCallback, useRef, useEffect } from 'react';
import { Input, Popover, List, Tag, Typography, Space, Empty, Spin } from 'antd';
import { SearchOutlined, FileSearchOutlined, ProjectOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { searchApi } from '../services';
import type { SearchReviewItem, SearchProjectItem } from '../services';
import { useThemeStore } from '../stores/themeStore';

const { Text } = Typography;

const GlobalSearch: React.FC = () => {
    const { t } = useTranslation();
    const navigate = useNavigate();
    const isDark = useThemeStore((state) => state.isDark);
    const [query, setQuery] = useState('');
    const [loading, setLoading] = useState(false);
    const [reviews, setReviews] = useState<SearchReviewItem[]>([]);
    const [projects, setProjects] = useState<SearchProjectItem[]>([]);
    const [open, setOpen] = useState(false);
    const debounceRef = useRef<ReturnType<typeof setTimeout>>();

    const doSearch = useCallback(async (q: string) => {
        if (q.length < 2) {
            setReviews([]);
            setProjects([]);
            return;
        }
        setLoading(true);
        try {
            const res = await searchApi.search(q, 10);
            setReviews(res.data?.reviews || []);
            setProjects(res.data?.projects || []);
            setOpen(true);
        } catch {
            setReviews([]);
            setProjects([]);
        } finally {
            setLoading(false);
        }
    }, []);

    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const val = e.target.value;
        setQuery(val);
        if (debounceRef.current) clearTimeout(debounceRef.current);
        debounceRef.current = setTimeout(() => doSearch(val), 300);
    };

    useEffect(() => {
        return () => {
            if (debounceRef.current) clearTimeout(debounceRef.current);
        };
    }, []);

    const handleReviewClick = (id: number) => {
        setOpen(false);
        setQuery('');
        navigate(`/admin/review-logs?highlight=${id}`);
    };

    const handleProjectClick = (id: number) => {
        setOpen(false);
        setQuery('');
        navigate(`/admin/projects?highlight=${id}`);
    };

    const getStatusColor = (status: string) => {
        const map: Record<string, string> = {
            completed: 'green',
            failed: 'red',
            analyzing: 'blue',
            pending: 'orange',
        };
        return map[status] || 'default';
    };

    const content = (
        <div style={{ width: 420, maxHeight: 480, overflow: 'auto' }}>
            {loading ? (
                <div style={{ textAlign: 'center', padding: 24 }}><Spin /></div>
            ) : reviews.length === 0 && projects.length === 0 ? (
                <Empty description={t('common.noData')} image={Empty.PRESENTED_IMAGE_SIMPLE} />
            ) : (
                <>
                    {projects.length > 0 && (
                        <>
                            <Text type="secondary" style={{ fontSize: 12, padding: '8px 12px', display: 'block' }}>
                                <ProjectOutlined /> {t('menu.projects')} ({projects.length})
                            </Text>
                            <List
                                size="small"
                                dataSource={projects}
                                renderItem={(item) => (
                                    <List.Item
                                        style={{ cursor: 'pointer', padding: '8px 12px' }}
                                        onClick={() => handleProjectClick(item.id)}
                                    >
                                        <Space>
                                            <Tag>{item.platform}</Tag>
                                            <Text strong>{item.name}</Text>
                                        </Space>
                                    </List.Item>
                                )}
                            />
                        </>
                    )}
                    {reviews.length > 0 && (
                        <>
                            <Text type="secondary" style={{ fontSize: 12, padding: '8px 12px', display: 'block' }}>
                                <FileSearchOutlined /> {t('menu.reviewLogs')} ({reviews.length})
                            </Text>
                            <List
                                size="small"
                                dataSource={reviews}
                                renderItem={(item) => (
                                    <List.Item
                                        style={{ cursor: 'pointer', padding: '8px 12px' }}
                                        onClick={() => handleReviewClick(item.id)}
                                    >
                                        <div style={{ width: '100%' }}>
                                            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                                                <Text strong ellipsis style={{ maxWidth: 260 }}>
                                                    {item.commit_message || item.commit_hash?.slice(0, 8)}
                                                </Text>
                                                <Tag color={getStatusColor(item.review_status)} style={{ marginLeft: 8 }}>
                                                    {item.review_status}
                                                </Tag>
                                            </div>
                                            <div style={{ fontSize: 12, color: isDark ? '#94a3b8' : '#94a3b8', marginTop: 2 }}>
                                                <span>{item.author}</span>
                                                <span style={{ margin: '0 6px' }}>·</span>
                                                <span>{item.project_name}</span>
                                                {item.score !== null && (
                                                    <>
                                                        <span style={{ margin: '0 6px' }}>·</span>
                                                        <span>{item.score}分</span>
                                                    </>
                                                )}
                                            </div>
                                        </div>
                                    </List.Item>
                                )}
                            />
                        </>
                    )}
                </>
            )}
        </div>
    );

    return (
        <Popover
            content={content}
            trigger="click"
            open={open && query.length >= 2}
            onOpenChange={(v) => { if (!v) setOpen(false); }}
            placement="bottomLeft"
            arrow={false}
            overlayStyle={{ padding: 0 }}
        >
            <Input
                prefix={<SearchOutlined style={{ color: isDark ? '#64748b' : '#94a3b8' }} />}
                placeholder={t('common.search')}
                value={query}
                onChange={handleChange}
                onFocus={() => { if (reviews.length > 0 || projects.length > 0) setOpen(true); }}
                allowClear
                style={{
                    width: 220,
                    borderRadius: 8,
                    background: isDark ? 'rgba(30, 41, 59, 0.8)' : 'rgba(241, 245, 249, 0.8)',
                    border: isDark ? '1px solid #334155' : '1px solid #e2e8f0',
                }}
            />
        </Popover>
    );
};

export default GlobalSearch;
