import React, { useState, useCallback } from 'react';
import { Badge, Popover, List, Tag, Typography, Empty, Button } from 'antd';
import { BellOutlined, CheckCircleOutlined, CloseCircleOutlined, SyncOutlined, ClockCircleOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useReviewSSE } from '../hooks/useSSE';
import type { ReviewEvent } from '../hooks/useSSE';
import { useThemeStore } from '../stores/themeStore';

const { Text } = Typography;

const MAX_NOTIFICATIONS = 30;

const NotificationBell: React.FC = () => {
    const { t } = useTranslation();
    const navigate = useNavigate();
    const isDark = useThemeStore((state) => state.isDark);
    const [notifications, setNotifications] = useState<ReviewEvent[]>([]);
    const [unreadCount, setUnreadCount] = useState(0);
    const [open, setOpen] = useState(false);

    const handleEvent = useCallback((event: ReviewEvent) => {
        setNotifications((prev) => {
            const updated = [event, ...prev.filter((n) => n.id !== event.id)];
            return updated.slice(0, MAX_NOTIFICATIONS);
        });
        if (!open) {
            setUnreadCount((c) => c + 1);
        }
    }, [open]);

    useReviewSSE({
        enabled: true,
        onEvent: handleEvent,
    });

    const handleOpenChange = (v: boolean) => {
        setOpen(v);
        if (v) setUnreadCount(0);
    };

    const handleClick = (id: number) => {
        setOpen(false);
        navigate(`/admin/review-logs?highlight=${id}`);
    };

    const handleClear = () => {
        setNotifications([]);
        setUnreadCount(0);
    };

    const getStatusIcon = (status: string) => {
        switch (status) {
            case 'completed': return <CheckCircleOutlined style={{ color: '#52c41a' }} />;
            case 'failed': return <CloseCircleOutlined style={{ color: '#ff4d4f' }} />;
            case 'analyzing': return <SyncOutlined spin style={{ color: '#1890ff' }} />;
            default: return <ClockCircleOutlined style={{ color: '#faad14' }} />;
        }
    };

    const getStatusColor = (status: string) => {
        const map: Record<string, string> = { completed: 'green', failed: 'red', analyzing: 'blue', pending: 'orange' };
        return map[status] || 'default';
    };

    const content = (
        <div style={{ width: 360, maxHeight: 420, overflow: 'auto' }}>
            {notifications.length === 0 ? (
                <Empty description={t('common.noData')} image={Empty.PRESENTED_IMAGE_SIMPLE} style={{ padding: 24 }} />
            ) : (
                <>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '8px 12px', borderBottom: isDark ? '1px solid #334155' : '1px solid #f0f0f0' }}>
                        <Text type="secondary" style={{ fontSize: 12 }}>
                            {t('common.total')} {notifications.length}
                        </Text>
                        <Button type="link" size="small" onClick={handleClear}>
                            {t('common.reset')}
                        </Button>
                    </div>
                    <List
                        size="small"
                        dataSource={notifications}
                        renderItem={(item) => (
                            <List.Item
                                style={{ cursor: 'pointer', padding: '8px 12px' }}
                                onClick={() => handleClick(item.id)}
                            >
                                <div style={{ display: 'flex', gap: 8, width: '100%', alignItems: 'flex-start' }}>
                                    <div style={{ paddingTop: 2 }}>
                                        {getStatusIcon(item.status)}
                                    </div>
                                    <div style={{ flex: 1, minWidth: 0 }}>
                                        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                                            <Text ellipsis style={{ maxWidth: 220, fontSize: 13 }}>
                                                {item.commit_sha?.slice(0, 8)}
                                            </Text>
                                            <Tag color={getStatusColor(item.status)} style={{ fontSize: 11, lineHeight: '18px', margin: 0 }}>
                                                {item.status}
                                            </Tag>
                                        </div>
                                        {item.score !== undefined && item.score !== null && (
                                            <Text type="secondary" style={{ fontSize: 12 }}>Score: {item.score}</Text>
                                        )}
                                        {item.error && (
                                            <Text type="danger" ellipsis style={{ fontSize: 12, display: 'block' }}>{item.error}</Text>
                                        )}
                                    </div>
                                </div>
                            </List.Item>
                        )}
                    />
                </>
            )}
        </div>
    );

    return (
        <Popover
            content={content}
            trigger="click"
            open={open}
            onOpenChange={handleOpenChange}
            placement="bottomRight"
            arrow={false}
        >
            <Badge count={unreadCount} size="small" offset={[-2, 4]}>
                <Button
                    type="text"
                    icon={<BellOutlined style={{ fontSize: 18, color: isDark ? '#e2e8f0' : '#64748b' }} />}
                    style={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}
                />
            </Badge>
        </Popover>
    );
};

export default NotificationBell;
