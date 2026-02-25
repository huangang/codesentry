import React, { useState } from 'react';
import { Card, Table, Button, Space, Modal, Form, Input, Select, InputNumber, Switch, message, Popconfirm, Tag } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { issueTrackerApi, type IssueTracker } from '../services';
import dayjs from 'dayjs';

const IssueTrackers: React.FC = () => {
    const { t } = useTranslation();
    const queryClient = useQueryClient();
    const [modalVisible, setModalVisible] = useState(false);
    const [editingItem, setEditingItem] = useState<IssueTracker | null>(null);
    const [form] = Form.useForm();

    const { data: trackers, isLoading } = useQuery<IssueTracker[]>({
        queryKey: ['issueTrackers'],
        queryFn: async () => {
            const res = await issueTrackerApi.list();
            return res.data;
        },
    });

    const createMutation = useMutation({
        mutationFn: async (data: any) => { const res = await issueTrackerApi.create(data); return res.data; },
        onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['issueTrackers'] }); message.success('Created'); setModalVisible(false); },
    });

    const updateMutation = useMutation({
        mutationFn: async ({ id, data }: { id: number; data: any }) => { const res = await issueTrackerApi.update(id, data); return res.data; },
        onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['issueTrackers'] }); message.success('Updated'); setModalVisible(false); },
    });

    const deleteMutation = useMutation({
        mutationFn: async (id: number) => { await issueTrackerApi.delete(id); },
        onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['issueTrackers'] }); message.success('Deleted'); },
    });

    const handleSubmit = async (values: any) => {
        if (editingItem) {
            await updateMutation.mutateAsync({ id: editingItem.id, data: values });
        } else {
            await createMutation.mutateAsync(values);
        }
    };

    const openEdit = (item: IssueTracker) => {
        setEditingItem(item);
        form.setFieldsValue(item);
        setModalVisible(true);
    };

    const openCreate = () => {
        setEditingItem(null);
        form.resetFields();
        form.setFieldsValue({ type: 'jira', issue_type: 'Bug', score_threshold: 60, is_active: true });
        setModalVisible(true);
    };

    const typeColors: Record<string, string> = { jira: 'blue', linear: 'purple', github_issues: 'green' };

    const columns = [
        { title: t('common.name', 'Name'), dataIndex: 'name', key: 'name' },
        {
            title: t('issueTrackers.type', 'Type'), dataIndex: 'type', key: 'type',
            render: (v: string) => <Tag color={typeColors[v] || 'default'}>{v}</Tag>,
        },
        { title: 'URL', dataIndex: 'base_url', key: 'base_url', ellipsis: true },
        { title: t('issueTrackers.projectKey', 'Project Key'), dataIndex: 'project_key', key: 'project_key' },
        { title: t('issueTrackers.threshold', 'Score Threshold'), dataIndex: 'score_threshold', key: 'score_threshold' },
        {
            title: t('common.status', 'Status'), dataIndex: 'is_active', key: 'is_active',
            render: (v: boolean) => <Tag color={v ? 'green' : 'default'}>{v ? 'Active' : 'Inactive'}</Tag>,
        },
        {
            title: t('common.createdAt', 'Created'), dataIndex: 'created_at', key: 'created_at',
            render: (v: string) => dayjs(v).format('YYYY-MM-DD'),
        },
        {
            title: t('common.actions', 'Actions'), key: 'actions',
            render: (_: any, record: IssueTracker) => (
                <Space>
                    <Button type="link" icon={<EditOutlined />} onClick={() => openEdit(record)} />
                    <Popconfirm title="Delete?" onConfirm={() => deleteMutation.mutate(record.id)}>
                        <Button type="link" danger icon={<DeleteOutlined />} />
                    </Popconfirm>
                </Space>
            ),
        },
    ];

    return (
        <Card
            title={t('issueTrackers.title', 'Issue Tracker Integrations')}
            extra={<Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>{t('common.create', 'Create')}</Button>}
        >
            <Table dataSource={trackers ?? []} columns={columns} rowKey="id" loading={isLoading} size="middle" />

            <Modal
                title={editingItem ? t('common.edit', 'Edit') : t('common.create', 'Create')}
                open={modalVisible}
                onCancel={() => setModalVisible(false)}
                onOk={() => form.submit()}
                confirmLoading={createMutation.isPending || updateMutation.isPending}
                width={600}
            >
                <Form form={form} layout="vertical" onFinish={handleSubmit}>
                    <Form.Item name="name" label={t('common.name', 'Name')} rules={[{ required: true }]}>
                        <Input />
                    </Form.Item>
                    <Form.Item name="type" label={t('issueTrackers.type', 'Type')} rules={[{ required: true }]}>
                        <Select options={[
                            { value: 'jira', label: 'Jira' },
                            { value: 'linear', label: 'Linear' },
                            { value: 'github_issues', label: 'GitHub Issues' },
                        ]} />
                    </Form.Item>
                    <Form.Item name="base_url" label="Base URL" extra="Jira: https://company.atlassian.net | GitHub: https://api.github.com">
                        <Input placeholder="https://company.atlassian.net" />
                    </Form.Item>
                    <Form.Item name="api_token" label="API Token">
                        <Input.Password placeholder={editingItem ? '(unchanged)' : 'Enter API token'} />
                    </Form.Item>
                    <Form.Item name="project_key" label={t('issueTrackers.projectKey', 'Project Key')} rules={[{ required: true }]} extra="Jira: project key | Linear: team ID | GitHub: owner/repo">
                        <Input />
                    </Form.Item>
                    <Form.Item name="issue_type" label={t('issueTrackers.issueType', 'Issue Type')}>
                        <Input placeholder="Bug" />
                    </Form.Item>
                    <Form.Item name="score_threshold" label={t('issueTrackers.threshold', 'Score Threshold')} extra="Auto-create issue when review score < threshold">
                        <InputNumber min={0} max={100} style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name="labels" label={t('issueTrackers.labels', 'Labels')}>
                        <Input placeholder="code-review, low-score" />
                    </Form.Item>
                    <Form.Item name="is_active" label={t('common.status', 'Active')} valuePropName="checked">
                        <Switch />
                    </Form.Item>
                </Form>
            </Modal>
        </Card>
    );
};

export default IssueTrackers;
