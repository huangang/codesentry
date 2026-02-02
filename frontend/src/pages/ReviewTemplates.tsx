import React, { useState } from 'react';
import {
    Card,
    Table,
    Button,
    Space,
    Tag,
    Modal,
    Form,
    Input,
    Select,
    Switch,
    message,
    Popconfirm,
    Drawer,
    Typography,
} from 'antd';
import {
    PlusOutlined,
    EditOutlined,
    DeleteOutlined,
    EyeOutlined,
    CopyOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import { usePermission, getResponsiveWidth } from '../hooks';
import {
    useReviewTemplates,
    useCreateReviewTemplate,
    useUpdateReviewTemplate,
    useDeleteReviewTemplate,
} from '../hooks/queries';
import type { ReviewTemplate } from '../services';

const { TextArea } = Input;
const { Paragraph } = Typography;

const TEMPLATE_TYPES = ['general', 'frontend', 'backend', 'security', 'custom'];

const ReviewTemplates: React.FC = () => {
    const { t } = useTranslation();
    const { isAdmin } = usePermission();
    const [modalVisible, setModalVisible] = useState(false);
    const [drawerVisible, setDrawerVisible] = useState(false);
    const [editingTemplate, setEditingTemplate] = useState<ReviewTemplate | null>(null);
    const [viewingTemplate, setViewingTemplate] = useState<ReviewTemplate | null>(null);
    const [typeFilter, setTypeFilter] = useState<string | undefined>(undefined);
    const [form] = Form.useForm();

    const { data: templates, isLoading } = useReviewTemplates(typeFilter);
    const createTemplate = useCreateReviewTemplate();
    const updateTemplate = useUpdateReviewTemplate();
    const deleteTemplate = useDeleteReviewTemplate();

    const showCreateModal = () => {
        setEditingTemplate(null);
        form.resetFields();
        form.setFieldsValue({ type: 'custom', is_active: true });
        setModalVisible(true);
    };

    const showEditModal = (record: ReviewTemplate) => {
        setEditingTemplate(record);
        form.setFieldsValue(record);
        setModalVisible(true);
    };

    const showViewDrawer = (record: ReviewTemplate) => {
        setViewingTemplate(record);
        setDrawerVisible(true);
    };

    const handleDuplicate = (record: ReviewTemplate) => {
        setEditingTemplate(null);
        form.resetFields();
        form.setFieldsValue({
            ...record,
            name: `${record.name} (Copy)`,
            is_built_in: false,
            type: 'custom',
        });
        setModalVisible(true);
    };

    const handleSubmit = async () => {
        try {
            const values = await form.validateFields();
            if (editingTemplate) {
                await updateTemplate.mutateAsync({ id: editingTemplate.id, data: values });
                message.success(t('common.updateSuccess'));
            } else {
                await createTemplate.mutateAsync(values);
                message.success(t('common.createSuccess'));
            }
            setModalVisible(false);
        } catch (error) {
            message.error(t('common.error'));
        }
    };

    const handleDelete = async (id: number) => {
        try {
            await deleteTemplate.mutateAsync(id);
            message.success(t('common.deleteSuccess'));
        } catch (error) {
            message.error(t('common.error'));
        }
    };

    const getTypeColor = (type: string) => {
        const colors: Record<string, string> = {
            general: 'default',
            frontend: 'blue',
            backend: 'green',
            security: 'red',
            custom: 'purple',
        };
        return colors[type] || 'default';
    };

    const columns: ColumnsType<ReviewTemplate> = [
        {
            title: t('reviewTemplates.name', 'Name'),
            dataIndex: 'name',
            key: 'name',
            render: (name, record) => (
                <Space>
                    {name}
                    {record.is_built_in && <Tag color="gold">{t('reviewTemplates.builtIn', 'Built-in')}</Tag>}
                </Space>
            ),
        },
        {
            title: t('reviewTemplates.type', 'Type'),
            dataIndex: 'type',
            key: 'type',
            render: (type: string) => <Tag color={getTypeColor(type)}>{type}</Tag>,
        },
        {
            title: t('reviewTemplates.description', 'Description'),
            dataIndex: 'description',
            key: 'description',
            ellipsis: true,
        },
        {
            title: t('common.status'),
            dataIndex: 'is_active',
            key: 'is_active',
            render: (isActive: boolean) => (
                <Tag color={isActive ? 'success' : 'default'}>
                    {isActive ? t('common.active') : t('common.inactive')}
                </Tag>
            ),
        },
        {
            title: t('common.actions'),
            key: 'actions',
            width: 200,
            render: (_, record) => (
                <Space size="small">
                    <Button
                        type="link"
                        size="small"
                        icon={<EyeOutlined />}
                        onClick={() => showViewDrawer(record)}
                    />
                    {isAdmin && (
                        <>
                            <Button
                                type="link"
                                size="small"
                                icon={<CopyOutlined />}
                                onClick={() => handleDuplicate(record)}
                            />
                            {!record.is_built_in && (
                                <>
                                    <Button
                                        type="link"
                                        size="small"
                                        icon={<EditOutlined />}
                                        onClick={() => showEditModal(record)}
                                    />
                                    <Popconfirm
                                        title={t('common.confirmDelete')}
                                        onConfirm={() => handleDelete(record.id)}
                                    >
                                        <Button
                                            type="link"
                                            size="small"
                                            danger
                                            icon={<DeleteOutlined />}
                                        />
                                    </Popconfirm>
                                </>
                            )}
                        </>
                    )}
                </Space>
            ),
        },
    ];

    return (
        <Card
            title={t('reviewTemplates.title', 'Review Templates')}
            extra={
                <Space>
                    <Select
                        placeholder={t('reviewTemplates.filterByType', 'Filter by type')}
                        allowClear
                        style={{ width: 150 }}
                        value={typeFilter}
                        onChange={setTypeFilter}
                        options={TEMPLATE_TYPES.map(type => ({ label: type, value: type }))}
                    />
                    {isAdmin && (
                        <Button type="primary" icon={<PlusOutlined />} onClick={showCreateModal}>
                            {t('reviewTemplates.create', 'Create Template')}
                        </Button>
                    )}
                </Space>
            }
        >
            <Table
                columns={columns}
                dataSource={templates}
                rowKey="id"
                loading={isLoading}
                pagination={{ pageSize: 10 }}
            />

            <Modal
                title={editingTemplate ? t('reviewTemplates.edit', 'Edit Template') : t('reviewTemplates.create', 'Create Template')}
                open={modalVisible}
                onOk={handleSubmit}
                onCancel={() => setModalVisible(false)}
                width={800}
                confirmLoading={createTemplate.isPending || updateTemplate.isPending}
            >
                <Form form={form} layout="vertical">
                    <Form.Item
                        name="name"
                        label={t('reviewTemplates.name', 'Name')}
                        rules={[{ required: true }]}
                    >
                        <Input />
                    </Form.Item>
                    <Form.Item
                        name="type"
                        label={t('reviewTemplates.type', 'Type')}
                        rules={[{ required: true }]}
                    >
                        <Select options={TEMPLATE_TYPES.map(type => ({ label: type, value: type }))} />
                    </Form.Item>
                    <Form.Item
                        name="description"
                        label={t('reviewTemplates.description', 'Description')}
                    >
                        <Input />
                    </Form.Item>
                    <Form.Item
                        name="content"
                        label={t('reviewTemplates.content', 'Template Content')}
                        rules={[{ required: true }]}
                    >
                        <TextArea rows={12} />
                    </Form.Item>
                    <Form.Item
                        name="is_active"
                        label={t('common.active')}
                        valuePropName="checked"
                    >
                        <Switch />
                    </Form.Item>
                </Form>
            </Modal>

            <Drawer
                title={viewingTemplate?.name}
                open={drawerVisible}
                onClose={() => setDrawerVisible(false)}
                width={getResponsiveWidth()}
            >
                {viewingTemplate && (
                    <div>
                        <Space style={{ marginBottom: 16 }}>
                            <Tag color={getTypeColor(viewingTemplate.type)}>{viewingTemplate.type}</Tag>
                            {viewingTemplate.is_built_in && <Tag color="gold">{t('reviewTemplates.builtIn', 'Built-in')}</Tag>}
                            <Tag color={viewingTemplate.is_active ? 'success' : 'default'}>
                                {viewingTemplate.is_active ? t('common.active') : t('common.inactive')}
                            </Tag>
                        </Space>
                        <Paragraph type="secondary">{viewingTemplate.description}</Paragraph>
                        <Card size="small" title={t('reviewTemplates.content', 'Template Content')}>
                            <pre style={{ whiteSpace: 'pre-wrap', fontSize: 12 }}>
                                {viewingTemplate.content}
                            </pre>
                        </Card>
                    </div>
                )}
            </Drawer>
        </Card>
    );
};

export default ReviewTemplates;
