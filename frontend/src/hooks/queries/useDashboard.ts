import { useQuery } from '@tanstack/react-query';
import { dashboardApi } from '../../services';

export interface DashboardFilters {
    start_date?: string;
    end_date?: string;
    project_limit?: number;
    author_limit?: number;
}

// Query keys
export const dashboardKeys = {
    all: ['dashboard'] as const,
    stats: (filters: DashboardFilters) => [...dashboardKeys.all, 'stats', filters] as const,
};

// Queries
export function useDashboardStats(filters: DashboardFilters = {}) {
    return useQuery({
        queryKey: dashboardKeys.stats(filters),
        queryFn: async () => {
            const res = await dashboardApi.getStats(filters);
            return res.data;
        },
    });
}
