import React, { useState } from 'react';
import { Card, Table, Button, Space, Modal, Form, Input, Select, InputNumber, Switch, message, Popconfirm, Tag } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { reviewRuleApi, type ReviewRule } from '../services';
import { useProjects } from '../hooks/queries';

const conditionOptions = [
    { value: 'score_below', label: 'Score Below Threshold' },
    { value: 'files_changed_above', label: 'Files Changed Above' },
    { value: 'has_keyword', label: 'Contains Keyword' },
    { value: 'additions_above', label: 'Additions Above' },
];

const actionOptions = [
    { value: 'block', label: 'Block (CI/CD Gating)' },
    { value: 'warn', label: 'Warn' },
    { value: 'notify', label: 'Notify' },
    { value: 'label', label: 'Label' },
];

const actionColors: Record<string, string> = { block: 'red', warn: 'orange', notify: 'blue', label: 'purple' };
const conditionLabels: Record<string, string> = {
    score_below: 'Score <', files_changed_above: 'Files >', has_keyword: 'Keyword', additions_above: 'Additions >',
};

const ReviewRules: React.FC = () => {
    const { t } = useTranslation();
    const queryClient = useQueryClient();
    const [modalVisible, setModalVisible] = useState(false);
    const [editingItem, setEditingItem] = useState<ReviewRule | null>(null);
    const [form] = Form.useForm();
    const conditionValue = Form.useWatch('condition', form);
    const { data: projectsData } = useProjects({ page_size: 100 });

    const { data: rules, isLoading } = useQuery<ReviewRule[]>({
        queryKey: ['reviewRules'],
        queryFn: async () => {
            const res = await reviewRuleApi.list();
            return res.data;
        },
    });

    const createMutation = useMutation({
        mutationFn: async (data: any) => { const res = await reviewRuleApi.create(data); return res.data; },
        onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['reviewRules'] }); message.success('Created'); setModalVisible(false); },
    });

    const updateMutation = useMutation({
        mutationFn: async ({ id, data }: { id: number; data: any }) => { const res = await reviewRuleApi.update(id, data); return res.data; },
        onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['reviewRules'] }); message.success('Updated'); setModalVisible(false); },
    });

    const deleteMutation = useMutation({
        mutationFn: async (id: number) => { await reviewRuleApi.delete(id); },
        onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['reviewRules'] }); message.success('Deleted'); },
    });

    const handleSubmit = async (values: any) => {
        if (editingItem) {
            await updateMutation.mutateAsync({ id: editingItem.id, data: values });
        } else {
            await createMutation.mutateAsync(values);
        }
    };

    const openEdit = (item: ReviewRule) => {
        setEditingItem(item);
        form.setFieldsValue(item);
        setModalVisible(true);
    };

    const openCreate = () => {
        setEditingItem(null);
        form.resetFields();
        form.setFieldsValue({ condition: 'score_below', action: 'warn', threshold: 60, priority: 0, is_active: true });
        setModalVisible(true);
    };

    const columns = [
        { title: t('common.name', 'Name'), dataIndex: 'name', key: 'name', width: 200 },
        {
            title: t('reviewRules.condition', 'Condition'), key: 'condition', width: 200,
            render: (_: any, r: ReviewRule) => {
                const label = conditionLabels[r.condition] || r.condition;
                return r.condition === 'has_keyword'
                    ? <Tag>{label}: {r.keyword}</Tag>
                    : <Tag>{label} {r.threshold}</Tag>;
            },
        },
        {
            title: t('reviewRules.action', 'Action'), dataIndex: 'action', key: 'action', width: 120,
            render: (v: string) => <Tag color={actionColors[v]}>{v}</Tag>,
        },
        {
            title: t('reviewRules.scope', 'Scope'), key: 'scope', width: 150,
            render: (_: any, r: ReviewRule) => r.project_id
                ? <Tag color="blue">{projectsData?.items?.find(p => p.id === r.project_id)?.name || `Project #${r.project_id}`}</Tag>
                : <Tag>Global</Tag>,
        },
        { title: t('reviewRules.priority', 'Priority'), dataIndex: 'priority', key: 'priority', width: 80 },
        {
            title: t('common.status', 'Status'), dataIndex: 'is_active', key: 'is_active', width: 80,
            render: (v: boolean) => <Tag color={v ? 'green' : 'default'}>{v ? 'Active' : 'Off'}</Tag>,
        },
        {
            title: t('common.actions', 'Actions'), key: 'actions', width: 100,
            render: (_: any, record: ReviewRule) => (
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
            title={t('reviewRules.title', 'Review Rules (CI/CD Policies)')}
            extra={<Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>{t('common.create', 'Create')}</Button>}
        >
            <Table dataSource={rules ?? []} columns={columns} rowKey="id" loading={isLoading} size="middle" scroll={{ x: 800 }} />

            <Modal
                title={editingItem ? t('common.edit', 'Edit Rule') : t('common.create', 'Create Rule')}
                open={modalVisible}
                onCancel={() => setModalVisible(false)}
                onOk={() => form.submit()}
                confirmLoading={createMutation.isPending || updateMutation.isPending}
                width={600}
            >
                <Form form={form} layout="vertical" onFinish={handleSubmit}>
                    <Form.Item name="name" label={t('common.name', 'Name')} rules={[{ required: true }]}>
                        <Input placeholder="Low Score Block" />
                    </Form.Item>
                    <Form.Item name="description" label={t('common.description', 'Description')}>
                        <Input.TextArea rows={2} />
                    </Form.Item>
                    <Form.Item name="condition" label={t('reviewRules.condition', 'Condition')} rules={[{ required: true }]}>
                        <Select options={conditionOptions} />
                    </Form.Item>
                    {conditionValue === 'has_keyword' ? (
                        <Form.Item name="keyword" label={t('reviewRules.keyword', 'Keywords')} extra="Comma-separated: SQL injection, XSS">
                            <Input placeholder="SQL injection, XSS, vulnerability" />
                        </Form.Item>
                    ) : (
                        <Form.Item name="threshold" label={t('reviewRules.threshold', 'Threshold')}>
                            <InputNumber style={{ width: '100%' }} />
                        </Form.Item>
                    )}
                    <Form.Item name="action" label={t('reviewRules.action', 'Action')} rules={[{ required: true }]}>
                        <Select options={actionOptions} />
                    </Form.Item>
                    <Form.Item name="action_value" label={t('reviewRules.actionValue', 'Action Value')} extra="Optional: label name or notification message">
                        <Input />
                    </Form.Item>
                    <Form.Item name="project_id" label={t('reviewRules.scope', 'Project Scope')} extra="Leave empty for global rule">
                        <Select
                            allowClear showSearch optionFilterProp="label"
                            placeholder="Global (all projects)"
                            options={projectsData?.items?.map(p => ({ value: p.id, label: p.name })) ?? []}
                        />
                    </Form.Item>
                    <Form.Item name="priority" label={t('reviewRules.priority', 'Priority')} extra="Higher = evaluated first">
                        <InputNumber style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name="is_active" label={t('common.status', 'Active')} valuePropName="checked">
                        <Switch />
                    </Form.Item>
                </Form>
            </Modal>
        </Card>
    );
};

export default ReviewRules;
