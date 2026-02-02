import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { userApi } from '../../services';

export interface UserFilters {
    page?: number;
    page_size?: number;
    username?: string;
    role?: string;
    auth_type?: string;
}

export const userKeys = {
    all: ['users'] as const,
    lists: () => [...userKeys.all, 'list'] as const,
    list: (filters: UserFilters) => [...userKeys.lists(), filters] as const,
};

export function useUsers(filters: UserFilters) {
    return useQuery({
        queryKey: userKeys.list(filters),
        queryFn: async () => {
            const res = await userApi.list(filters);
            return res.data;
        },
    });
}

export function useUpdateUser() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async ({ id, data }: { id: number; data: Parameters<typeof userApi.update>[1] }) => {
            const res = await userApi.update(id, data);
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: userKeys.all });
        },
    });
}

export function useDeleteUser() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (id: number) => {
            await userApi.delete(id);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: userKeys.all });
        },
    });
}
