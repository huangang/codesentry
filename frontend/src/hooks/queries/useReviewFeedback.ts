import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { reviewFeedbackApi } from '../../services';

const reviewFeedbackKeys = {
    all: ['reviewFeedbacks'] as const,
    byReview: (reviewLogId: number) => [...reviewFeedbackKeys.all, 'byReview', reviewLogId] as const,
    detail: (id: number) => [...reviewFeedbackKeys.all, 'detail', id] as const,
};

export function useReviewFeedbacks(reviewLogId: number) {
    return useQuery({
        queryKey: reviewFeedbackKeys.byReview(reviewLogId),
        queryFn: async () => {
            const res = await reviewFeedbackApi.listByReview(reviewLogId);
            return res.data;
        },
        enabled: !!reviewLogId,
        refetchInterval: (query) => {
            // Auto-refresh if any feedback is still processing
            const data = query.state.data;
            if (data?.some(f => f.process_status === 'pending' || f.process_status === 'processing')) {
                return 2000; // Poll every 2 seconds
            }
            return false;
        },
    });
}

export function useReviewFeedback(id: number) {
    return useQuery({
        queryKey: reviewFeedbackKeys.detail(id),
        queryFn: async () => {
            const res = await reviewFeedbackApi.getById(id);
            return res.data;
        },
        enabled: !!id,
    });
}

export function useCreateReviewFeedback() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (data: Parameters<typeof reviewFeedbackApi.create>[0]) => {
            const res = await reviewFeedbackApi.create(data);
            return res.data;
        },
        onSuccess: (_, variables) => {
            queryClient.invalidateQueries({ queryKey: reviewFeedbackKeys.byReview(variables.review_log_id) });
        },
    });
}
