import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { promptApi } from '../../services';

export interface PromptFilters {
    page?: number;
    page_size?: number;
    name?: string;
    is_system?: boolean;
}

export const promptKeys = {
    all: ['prompts'] as const,
    lists: () => [...promptKeys.all, 'list'] as const,
    list: (filters: PromptFilters) => [...promptKeys.lists(), filters] as const,
    active: () => [...promptKeys.all, 'active'] as const,
    default: () => [...promptKeys.all, 'default'] as const,
};

export function usePrompts(filters: PromptFilters) {
    return useQuery({
        queryKey: promptKeys.list(filters),
        queryFn: async () => {
            const res = await promptApi.list(filters);
            return res.data;
        },
    });
}

export function useCreatePrompt() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (data: Parameters<typeof promptApi.create>[0]) => {
            const res = await promptApi.create(data);
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: promptKeys.all });
        },
    });
}

export function useUpdatePrompt() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async ({ id, data }: { id: number; data: Parameters<typeof promptApi.update>[1] }) => {
            const res = await promptApi.update(id, data);
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: promptKeys.all });
        },
    });
}

export function useDeletePrompt() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (id: number) => {
            await promptApi.delete(id);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: promptKeys.all });
        },
    });
}

export function useSetDefaultPrompt() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (id: number) => {
            await promptApi.setDefault(id);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: promptKeys.all });
        },
    });
}
