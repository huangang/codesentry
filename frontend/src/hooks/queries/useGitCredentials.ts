import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { gitCredentialApi } from '../../services';

export interface GitCredentialFilters {
    page?: number;
    page_size?: number;
    name?: string;
    platform?: string;
    is_active?: boolean;
}

export const gitCredentialKeys = {
    all: ['gitCredentials'] as const,
    lists: () => [...gitCredentialKeys.all, 'list'] as const,
    list: (filters: GitCredentialFilters) => [...gitCredentialKeys.lists(), filters] as const,
    active: () => [...gitCredentialKeys.all, 'active'] as const,
};

export function useGitCredentials(filters: GitCredentialFilters) {
    return useQuery({
        queryKey: gitCredentialKeys.list(filters),
        queryFn: async () => {
            const res = await gitCredentialApi.list(filters);
            return res.data;
        },
    });
}

export function useCreateGitCredential() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (data: Parameters<typeof gitCredentialApi.create>[0]) => {
            const res = await gitCredentialApi.create(data);
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: gitCredentialKeys.all });
        },
    });
}

export function useUpdateGitCredential() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async ({ id, data }: { id: number; data: Parameters<typeof gitCredentialApi.update>[1] }) => {
            const res = await gitCredentialApi.update(id, data);
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: gitCredentialKeys.all });
        },
    });
}

export function useDeleteGitCredential() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (id: number) => {
            await gitCredentialApi.delete(id);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: gitCredentialKeys.all });
        },
    });
}
