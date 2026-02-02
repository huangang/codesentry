import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { reviewLogApi, reviewLogApiExtra } from '../../services';

export interface ReviewLogFilters {
    page?: number;
    page_size?: number;
    event_type?: string;
    project_id?: number;
    author?: string;
    start_date?: string;
    end_date?: string;
    search_text?: string;
}

// Query keys
export const reviewLogKeys = {
    all: ['reviewLogs'] as const,
    lists: () => [...reviewLogKeys.all, 'list'] as const,
    list: (filters: ReviewLogFilters) => [...reviewLogKeys.lists(), filters] as const,
    details: () => [...reviewLogKeys.all, 'detail'] as const,
    detail: (id: number) => [...reviewLogKeys.details(), id] as const,
};

// Queries
export function useReviewLogs(filters: ReviewLogFilters) {
    return useQuery({
        queryKey: reviewLogKeys.list(filters),
        queryFn: async () => {
            const res = await reviewLogApi.list(filters);
            return res.data;
        },
    });
}

export function useReviewLog(id: number) {
    return useQuery({
        queryKey: reviewLogKeys.detail(id),
        queryFn: async () => {
            const res = await reviewLogApi.getById(id);
            return res.data;
        },
        enabled: id > 0,
    });
}

// Mutations
export function useRetryReview() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (id: number) => {
            const res = await reviewLogApi.retry(id);
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: reviewLogKeys.lists() });
        },
    });
}

export function useDeleteReviewLog() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (id: number) => {
            await reviewLogApiExtra.delete(id);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: reviewLogKeys.lists() });
        },
    });
}
