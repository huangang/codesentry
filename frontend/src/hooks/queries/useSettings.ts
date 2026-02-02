import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { systemConfigApi, type DailyReportConfig, type ChunkedReviewConfig, type FileContextConfig } from '../../services';
import type { LDAPConfig } from '../../types';

// Query keys
export const settingsKeys = {
    all: ['settings'] as const,
    ldap: () => [...settingsKeys.all, 'ldap'] as const,
    dailyReport: () => [...settingsKeys.all, 'dailyReport'] as const,
    chunkedReview: () => [...settingsKeys.all, 'chunkedReview'] as const,
    fileContext: () => [...settingsKeys.all, 'fileContext'] as const,
    activeLLMs: () => [...settingsKeys.all, 'activeLLMs'] as const,
    activeIMBots: () => [...settingsKeys.all, 'activeIMBots'] as const,
};

// Queries
export function useLDAPConfig() {
    return useQuery({
        queryKey: settingsKeys.ldap(),
        queryFn: async () => {
            const res = await systemConfigApi.getLDAPConfig();
            return res.data;
        },
    });
}

export function useDailyReportConfig() {
    return useQuery({
        queryKey: settingsKeys.dailyReport(),
        queryFn: async () => {
            const res = await systemConfigApi.getDailyReportConfig();
            return res.data;
        },
    });
}

export function useChunkedReviewConfig() {
    return useQuery({
        queryKey: settingsKeys.chunkedReview(),
        queryFn: async () => {
            const res = await systemConfigApi.getChunkedReviewConfig();
            return res.data;
        },
    });
}

export function useFileContextConfig() {
    return useQuery({
        queryKey: settingsKeys.fileContext(),
        queryFn: async () => {
            const res = await systemConfigApi.getFileContextConfig();
            return res.data;
        },
    });
}

// Note: useActiveLLMConfigs and useActiveIMBots are exported from useProjects.ts

// Mutations
export function useUpdateLDAPConfig() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (data: Partial<LDAPConfig>) => {
            const res = await systemConfigApi.updateLDAPConfig(data);
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: settingsKeys.ldap() });
        },
    });
}

export function useUpdateDailyReportConfig() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (data: Partial<DailyReportConfig>) => {
            const res = await systemConfigApi.updateDailyReportConfig(data);
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: settingsKeys.dailyReport() });
        },
    });
}

export function useUpdateChunkedReviewConfig() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (data: Partial<ChunkedReviewConfig>) => {
            const res = await systemConfigApi.updateChunkedReviewConfig(data);
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: settingsKeys.chunkedReview() });
        },
    });
}

export function useUpdateFileContextConfig() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (data: Partial<FileContextConfig>) => {
            const res = await systemConfigApi.updateFileContextConfig(data);
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: settingsKeys.fileContext() });
        },
    });
}
