import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { systemLogApi, memberApi, dailyReportApi } from '../../services';

// System Logs
export interface SystemLogFilters {
    page?: number;
    page_size?: number;
    level?: string;
    module?: string;
    action?: string;
    start_date?: string;
    end_date?: string;
    search?: string;
}

export const systemLogKeys = {
    all: ['systemLogs'] as const,
    lists: () => [...systemLogKeys.all, 'list'] as const,
    list: (filters: SystemLogFilters) => [...systemLogKeys.lists(), filters] as const,
    modules: () => [...systemLogKeys.all, 'modules'] as const,
};

export function useSystemLogs(filters: SystemLogFilters) {
    return useQuery({
        queryKey: systemLogKeys.list(filters),
        queryFn: async () => {
            const res = await systemLogApi.list(filters);
            return res.data;
        },
    });
}

export function useSystemLogModules() {
    return useQuery({
        queryKey: systemLogKeys.modules(),
        queryFn: async () => {
            const res = await systemLogApi.getModules();
            return res.data.modules;
        },
    });
}

export function useCleanupSystemLogs() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async () => {
            const res = await systemLogApi.cleanup();
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: systemLogKeys.all });
        },
    });
}

// Member Analysis
export interface MemberFilters {
    page?: number;
    page_size?: number;
    name?: string;
    project_id?: number;
    start_date?: string;
    end_date?: string;
    sort_by?: string;
    sort_order?: string;
}

export const memberKeys = {
    all: ['members'] as const,
    lists: () => [...memberKeys.all, 'list'] as const,
    list: (filters: MemberFilters) => [...memberKeys.lists(), filters] as const,
    detail: (params: { author: string; start_date?: string; end_date?: string }) => [...memberKeys.all, 'detail', params] as const,
};

export function useMembers(filters: MemberFilters) {
    return useQuery({
        queryKey: memberKeys.list(filters),
        queryFn: async () => {
            const res = await memberApi.list(filters);
            return res.data;
        },
    });
}

export function useMemberDetail(params: { author: string; start_date?: string; end_date?: string }, enabled: boolean = true) {
    return useQuery({
        queryKey: memberKeys.detail(params),
        queryFn: async () => {
            const res = await memberApi.getDetail(params);
            return res.data;
        },
        enabled: enabled && !!params.author,
    });
}

// Daily Reports
export interface DailyReportFilters {
    page?: number;
    page_size?: number;
}

export const dailyReportKeys = {
    all: ['dailyReports'] as const,
    lists: () => [...dailyReportKeys.all, 'list'] as const,
    list: (filters: DailyReportFilters) => [...dailyReportKeys.lists(), filters] as const,
    detail: (id: number) => [...dailyReportKeys.all, 'detail', id] as const,
};

export function useDailyReports(filters: DailyReportFilters) {
    return useQuery({
        queryKey: dailyReportKeys.list(filters),
        queryFn: async () => {
            const res = await dailyReportApi.list(filters);
            return res.data;
        },
    });
}

export function useDailyReport(id: number) {
    return useQuery({
        queryKey: dailyReportKeys.detail(id),
        queryFn: async () => {
            const res = await dailyReportApi.getById(id);
            return res.data;
        },
        enabled: id > 0,
    });
}

export function useGenerateDailyReport() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async () => {
            const res = await dailyReportApi.generate();
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: dailyReportKeys.all });
        },
    });
}

export function useResendDailyReport() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (id: number) => {
            const res = await dailyReportApi.resend(id);
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: dailyReportKeys.all });
        },
    });
}

// Member Stats exports (alias for compatibility)
export type MemberStatsFilters = MemberFilters;
export const useMemberStats = useMembers;
