import { useQuery } from '@tanstack/react-query';
import { aiUsageApi } from '../../services';

export interface AIUsageFilters {
    start_date?: string;
    end_date?: string;
    project_id?: number;
}

export const aiUsageKeys = {
    all: ['aiUsage'] as const,
    stats: (filters: AIUsageFilters) => [...aiUsageKeys.all, 'stats', filters] as const,
    trend: (filters: AIUsageFilters) => [...aiUsageKeys.all, 'trend', filters] as const,
    providers: (filters: Omit<AIUsageFilters, 'project_id'>) => [...aiUsageKeys.all, 'providers', filters] as const,
};

export function useAIUsageStats(filters: AIUsageFilters) {
    return useQuery({
        queryKey: aiUsageKeys.stats(filters),
        queryFn: async () => {
            const res = await aiUsageApi.getStats(filters);
            return res.data;
        },
    });
}

export function useAIUsageTrend(filters: AIUsageFilters) {
    return useQuery({
        queryKey: aiUsageKeys.trend(filters),
        queryFn: async () => {
            const res = await aiUsageApi.getDailyTrend(filters);
            return res.data;
        },
    });
}

export function useAIUsageProviders(filters: Omit<AIUsageFilters, 'project_id'>) {
    return useQuery({
        queryKey: aiUsageKeys.providers(filters),
        queryFn: async () => {
            const res = await aiUsageApi.getProviderBreakdown(filters);
            return res.data;
        },
    });
}
