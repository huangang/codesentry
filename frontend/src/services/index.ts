import api from './api';
import type { 
  LoginResponse, 
  AuthConfig, 
  User,
  Project,
  ReviewLog,
  LLMConfig,
  IMBot,
  PromptTemplate,
  PaginatedResponse,
  DashboardResponse 
} from '../types';

// Auth
export const authApi = {
  login: (username: string, password: string, authType: string = 'local') =>
    api.post<LoginResponse>('/auth/login', { username, password, auth_type: authType }),
  
  getConfig: () => api.get<AuthConfig>('/auth/config'),
  
  getCurrentUser: () => api.get<User>('/auth/me'),
  
  logout: () => api.post('/auth/logout'),
};

// Dashboard
export const dashboardApi = {
  getStats: (startDate?: string, endDate?: string) =>
    api.get<DashboardResponse>('/dashboard/stats', { params: { start_date: startDate, end_date: endDate } }),
};

// Projects
export const projectApi = {
  list: (params?: { page?: number; page_size?: number; name?: string; platform?: string }) =>
    api.get<PaginatedResponse<Project>>('/projects', { params }),
  
  getById: (id: number) => api.get<Project>(`/projects/${id}`),
  
  create: (data: Partial<Project> & { access_token?: string; webhook_secret?: string }) =>
    api.post<Project>('/projects', data),
  
  update: (id: number, data: Partial<Project> & { access_token?: string; webhook_secret?: string }) =>
    api.put<Project>(`/projects/${id}`, data),
  
  delete: (id: number) => api.delete(`/projects/${id}`),
  
  getDefaultPrompt: () => api.get<{ prompt: string }>('/projects/default-prompt'),
};

// Review Logs
export const reviewLogApi = {
  list: (params?: {
    page?: number;
    page_size?: number;
    event_type?: string;
    project_id?: number;
    author?: string;
    start_date?: string;
    end_date?: string;
    search_text?: string;
  }) => api.get<PaginatedResponse<ReviewLog>>('/review-logs', { params }),
  
  getById: (id: number) => api.get<ReviewLog>(`/review-logs/${id}`),
};

// LLM Configs
export const llmConfigApi = {
  list: (params?: { page?: number; page_size?: number; name?: string; provider?: string; is_active?: boolean }) =>
    api.get<PaginatedResponse<LLMConfig>>('/llm-configs', { params }),
  
  getById: (id: number) => api.get<LLMConfig>(`/llm-configs/${id}`),
  
  create: (data: Partial<LLMConfig> & { api_key: string }) =>
    api.post<LLMConfig>('/llm-configs', data),
  
  update: (id: number, data: Partial<LLMConfig> & { api_key?: string }) =>
    api.put<LLMConfig>(`/llm-configs/${id}`, data),
  
  delete: (id: number) => api.delete(`/llm-configs/${id}`),
};

// IM Bots
export const imBotApi = {
  list: (params?: { page?: number; page_size?: number; name?: string; type?: string; is_active?: boolean }) =>
    api.get<PaginatedResponse<IMBot>>('/im-bots', { params }),
  
  getById: (id: number) => api.get<IMBot>(`/im-bots/${id}`),
  
  getActive: () => api.get<IMBot[]>('/im-bots/active'),
  
  create: (data: Partial<IMBot> & { secret?: string }) =>
    api.post<IMBot>('/im-bots', data),
  
  update: (id: number, data: Partial<IMBot> & { secret?: string }) =>
    api.put<IMBot>(`/im-bots/${id}`, data),
  
  delete: (id: number) => api.delete(`/im-bots/${id}`),
};

// Prompts
export const promptApi = {
  list: (params?: { page?: number; page_size?: number; name?: string; is_system?: boolean }) =>
    api.get<PaginatedResponse<PromptTemplate>>('/prompts', { params }),
  
  getById: (id: number) => api.get<PromptTemplate>(`/prompts/${id}`),
  
  getDefault: () => api.get<PromptTemplate>('/prompts/default'),
  
  getActive: () => api.get<PromptTemplate[]>('/prompts/active'),
  
  create: (data: Partial<PromptTemplate>) =>
    api.post<PromptTemplate>('/prompts', data),
  
  update: (id: number, data: Partial<PromptTemplate>) =>
    api.put<PromptTemplate>(`/prompts/${id}`, data),
  
  delete: (id: number) => api.delete(`/prompts/${id}`),
  
  setDefault: (id: number) => api.post(`/prompts/${id}/set-default`),
};

// Members
export const memberApi = {
  list: (params?: {
    page?: number;
    page_size?: number;
    name?: string;
    project_id?: number;
    start_date?: string;
    end_date?: string;
    sort_by?: string;
    sort_order?: string;
  }) => api.get<{ total: number; page: number; page_size: number; items: any[] }>('/members', { params }),

  getDetail: (params: { author: string; start_date?: string; end_date?: string }) =>
    api.get<any>('/members/detail', { params }),
};

// System Logs
export interface SystemLog {
  id: number;
  level: string;
  module: string;
  action: string;
  message: string;
  user_id?: number;
  ip: string;
  user_agent: string;
  extra: string;
  created_at: string;
}

export const systemLogApi = {
  list: (params?: {
    page?: number;
    page_size?: number;
    level?: string;
    module?: string;
    action?: string;
    start_date?: string;
    end_date?: string;
    search?: string;
  }) => api.get<{ total: number; page: number; page_size: number; items: SystemLog[] }>('/system-logs', { params }),

  getModules: () => api.get<{ modules: string[] }>('/system-logs/modules'),
};
