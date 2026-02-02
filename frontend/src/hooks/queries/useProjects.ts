import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { projectApi, imBotApi, promptApi, llmConfigApi } from '../../services';

export interface ProjectFilters {
    page?: number;
    page_size?: number;
    name?: string;
    platform?: string;
}

// Query keys
export const projectKeys = {
    all: ['projects'] as const,
    lists: () => [...projectKeys.all, 'list'] as const,
    list: (filters: ProjectFilters) => [...projectKeys.lists(), filters] as const,
    details: () => [...projectKeys.all, 'detail'] as const,
    detail: (id: number) => [...projectKeys.details(), id] as const,
    defaultPrompt: () => [...projectKeys.all, 'defaultPrompt'] as const,
};

// Queries
export function useProjects(filters: ProjectFilters) {
    return useQuery({
        queryKey: projectKeys.list(filters),
        queryFn: async () => {
            const res = await projectApi.list(filters);
            return res.data;
        },
    });
}

export function useProject(id: number) {
    return useQuery({
        queryKey: projectKeys.detail(id),
        queryFn: async () => {
            const res = await projectApi.getById(id);
            return res.data;
        },
        enabled: id > 0,
    });
}

export function useDefaultPrompt() {
    return useQuery({
        queryKey: projectKeys.defaultPrompt(),
        queryFn: async () => {
            const res = await projectApi.getDefaultPrompt();
            return res.data.prompt;
        },
    });
}

// Related data queries
export function useActiveImBots() {
    return useQuery({
        queryKey: ['imBots', 'active'],
        queryFn: async () => {
            const res = await imBotApi.getActive();
            return res.data;
        },
    });
}

export function useActivePromptTemplates() {
    return useQuery({
        queryKey: ['promptTemplates', 'active'],
        queryFn: async () => {
            const res = await promptApi.getActive();
            return res.data;
        },
    });
}

export function useActiveLLMConfigs() {
    return useQuery({
        queryKey: ['llmConfigs', 'active'],
        queryFn: async () => {
            const res = await llmConfigApi.getActive();
            return res.data;
        },
    });
}

// Mutations
export function useCreateProject() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (data: Parameters<typeof projectApi.create>[0]) => {
            const res = await projectApi.create(data);
            return res.data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: projectKeys.lists() });
        },
    });
}

export function useUpdateProject() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async ({ id, data }: { id: number; data: Parameters<typeof projectApi.update>[1] }) => {
            const res = await projectApi.update(id, data);
            return res.data;
        },
        onSuccess: (_, variables) => {
            queryClient.invalidateQueries({ queryKey: projectKeys.lists() });
            queryClient.invalidateQueries({ queryKey: projectKeys.detail(variables.id) });
        },
    });
}

export function useDeleteProject() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (id: number) => {
            await projectApi.delete(id);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: projectKeys.lists() });
        },
    });
}
