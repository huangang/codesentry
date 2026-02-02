import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { imBotApi } from '../../services';

export interface IMBotFilters {
    page?: number;
    page_size?: number;
    name?: string;
    type?: string;
    is_active?: boolean;
}

export const imBotKeys = {
    all: ['imBots'] as const,
    lists: () => [...imBotKeys.all, 'list'] as const,
    list: (filters: IMBotFilters) => [...imBotKeys.lists(), filters] as const,
    active: () => [...imBotKeys.all, 'active'] as const,
};

export function useIMBots(filters: IMBotFilters) {
    return useQuery({
        queryKey: imBotKeys.list(filters),
        queryFn: async () => {
            const res = await imBotApi.list(filters);
            return res.data;
        },
    });
}

export function useCreateIMBot() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (data: Parameters<typeof imBotApi.create>[0]) => {
            const res = await imBotApi.create(data);
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: imBotKeys.all });
        },
    });
}

export function useUpdateIMBot() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async ({ id, data }: { id: number; data: Parameters<typeof imBotApi.update>[1] }) => {
            const res = await imBotApi.update(id, data);
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: imBotKeys.all });
        },
    });
}

export function useDeleteIMBot() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (id: number) => {
            await imBotApi.delete(id);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: imBotKeys.all });
        },
    });
}
