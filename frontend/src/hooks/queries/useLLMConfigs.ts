import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { llmConfigApi } from '../../services';

export interface LLMConfigFilters {
    page?: number;
    page_size?: number;
    name?: string;
    provider?: string;
    is_active?: boolean;
}

export const llmConfigKeys = {
    all: ['llmConfigs'] as const,
    lists: () => [...llmConfigKeys.all, 'list'] as const,
    list: (filters: LLMConfigFilters) => [...llmConfigKeys.lists(), filters] as const,
    active: () => [...llmConfigKeys.all, 'active'] as const,
};

export function useLLMConfigs(filters: LLMConfigFilters) {
    return useQuery({
        queryKey: llmConfigKeys.list(filters),
        queryFn: async () => {
            const res = await llmConfigApi.list(filters);
            return res.data;
        },
    });
}

export function useCreateLLMConfig() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (data: Parameters<typeof llmConfigApi.create>[0]) => {
            const res = await llmConfigApi.create(data);
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: llmConfigKeys.all });
        },
    });
}

export function useUpdateLLMConfig() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async ({ id, data }: { id: number; data: Parameters<typeof llmConfigApi.update>[1] }) => {
            const res = await llmConfigApi.update(id, data);
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: llmConfigKeys.all });
        },
    });
}

export function useDeleteLLMConfig() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (id: number) => {
            await llmConfigApi.delete(id);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: llmConfigKeys.all });
        },
    });
}
