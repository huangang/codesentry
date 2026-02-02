import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { reviewTemplateApi } from '../../services';

export const reviewTemplateKeys = {
    all: ['reviewTemplates'] as const,
    lists: () => [...reviewTemplateKeys.all, 'list'] as const,
    list: (type?: string) => [...reviewTemplateKeys.lists(), { type }] as const,
    detail: (id: number) => [...reviewTemplateKeys.all, 'detail', id] as const,
};

export function useReviewTemplates(type?: string) {
    return useQuery({
        queryKey: reviewTemplateKeys.list(type),
        queryFn: async () => {
            const res = await reviewTemplateApi.list({ type });
            return res.data;
        },
    });
}

export function useReviewTemplate(id: number) {
    return useQuery({
        queryKey: reviewTemplateKeys.detail(id),
        queryFn: async () => {
            const res = await reviewTemplateApi.getById(id);
            return res.data;
        },
        enabled: id > 0,
    });
}

export function useCreateReviewTemplate() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (data: Parameters<typeof reviewTemplateApi.create>[0]) => {
            const res = await reviewTemplateApi.create(data);
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: reviewTemplateKeys.all });
        },
    });
}

export function useUpdateReviewTemplate() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async ({ id, data }: { id: number; data: Parameters<typeof reviewTemplateApi.update>[1] }) => {
            const res = await reviewTemplateApi.update(id, data);
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: reviewTemplateKeys.all });
        },
    });
}

export function useDeleteReviewTemplate() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (id: number) => {
            await reviewTemplateApi.delete(id);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: reviewTemplateKeys.all });
        },
    });
}
