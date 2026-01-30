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
  DashboardResponse,
  GitCredential,
  LDAPConfig
} from '../types';

// Auth
export const authApi = {
  login: (username: string, password: string, authType: string = 'local') =>
    api.post<LoginResponse>('/auth/login', { username, password, auth_type: authType }),
  
  getConfig: () => api.get<AuthConfig>('/auth/config'),
  
  getCurrentUser: () => api.get<User>('/auth/me'),
  
  logout: () => api.post('/auth/logout'),

  changePassword: (oldPassword: string, newPassword: string) =>
    api.post<{ message: string }>('/auth/change-password', { old_password: oldPassword, new_password: newPassword }),
};

// Dashboard
export const dashboardApi = {
  getStats: (params?: { start_date?: string; end_date?: string; project_limit?: number; author_limit?: number }) =>
    api.get<DashboardResponse>('/dashboard/stats', { params }),
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

  retry: (id: number) => api.post<{ message: string }>(`/review-logs/${id}/retry`),
};

// LLM Configs
export const llmConfigApi = {
  list: (params?: { page?: number; page_size?: number; name?: string; provider?: string; is_active?: boolean }) =>
    api.get<PaginatedResponse<LLMConfig>>('/llm-configs', { params }),
  
  getById: (id: number) => api.get<LLMConfig>(`/llm-configs/${id}`),
  
  getActive: () => api.get<LLMConfig[]>('/llm-configs/active'),
  
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

  getRetentionDays: () => api.get<{ retention_days: number }>('/system-logs/retention'),

  setRetentionDays: (days: number) => api.put<{ retention_days: number }>('/system-logs/retention', { days }),

  cleanup: (days?: number) => api.post<{ deleted: number; retention_days: number }>('/system-logs/cleanup', { days }),
};

export const gitCredentialApi = {
  list: (params?: { page?: number; page_size?: number; name?: string; platform?: string; is_active?: boolean }) =>
    api.get<PaginatedResponse<GitCredential>>('/git-credentials', { params }),
  
  getById: (id: number) => api.get<GitCredential>(`/git-credentials/${id}`),
  
  getActive: () => api.get<GitCredential[]>('/git-credentials/active'),
  
  create: (data: Partial<GitCredential> & { access_token?: string; webhook_secret?: string }) =>
    api.post<GitCredential>('/git-credentials', data),
  
  update: (id: number, data: Partial<GitCredential> & { access_token?: string; webhook_secret?: string }) =>
    api.put<GitCredential>(`/git-credentials/${id}`, data),
  
  delete: (id: number) => api.delete(`/git-credentials/${id}`),
};

export const systemConfigApi = {
  getLDAPConfig: () => api.get<LDAPConfig>('/system-config/ldap'),
  
  updateLDAPConfig: (data: Partial<LDAPConfig>) =>
    api.put<LDAPConfig>('/system-config/ldap', data),

  getDailyReportConfig: () => api.get<DailyReportConfig>('/system-config/daily-report'),

  updateDailyReportConfig: (data: Partial<DailyReportConfig>) =>
    api.put<DailyReportConfig>('/system-config/daily-report', data),

  getChunkedReviewConfig: () => api.get<ChunkedReviewConfig>('/system-config/chunked-review'),

  updateChunkedReviewConfig: (data: Partial<ChunkedReviewConfig>) =>
    api.put<ChunkedReviewConfig>('/system-config/chunked-review', data),

  getFileContextConfig: () => api.get<FileContextConfig>('/system-config/file-context'),

  updateFileContextConfig: (data: Partial<FileContextConfig>) =>
    api.put<FileContextConfig>('/system-config/file-context', data),
};

export interface DailyReportConfig {
  enabled: boolean;
  time: string;
  timezone: string;
  low_score: number;
  llm_config_id: number;
  im_bot_ids: number[];
}

export interface ChunkedReviewConfig {
  enabled: boolean;
  threshold: number;
  max_tokens_per_batch: number;
}

export interface FileContextConfig {
  enabled: boolean;
  max_file_size: number;
  max_files: number;
}

export const userApi = {
  list: (params?: { page?: number; page_size?: number; username?: string; role?: string; auth_type?: string }) =>
    api.get<{ items: User[]; total: number; page: number; page_size: number }>('/users', { params }),
  
  update: (id: number, data: { role?: string; is_active?: boolean; nickname?: string }) =>
    api.put<User>(`/users/${id}`, data),
  
  delete: (id: number) => api.delete(`/users/${id}`),
};

export const reviewLogApiExtra = {
  delete: (id: number) => api.delete(`/review-logs/${id}`),
};

// Daily Reports
export interface DailyReport {
  id: number;
  report_date: string;
  report_type: string;
  total_projects: number;
  total_commits: number;
  total_authors: number;
  total_additions: number;
  total_deletions: number;
  average_score: number;
  passed_count: number;
  failed_count: number;
  pending_count: number;
  top_projects: string;
  top_authors: string;
  low_score_reviews: string;
  ai_analysis: string;
  ai_model_used: string;
  notified_at?: string;
  notify_error?: string;
  created_at: string;
}

export const dailyReportApi = {
  list: (params?: { page?: number; page_size?: number }) =>
    api.get<{ total: number; page: number; page_size: number; items: DailyReport[] }>('/daily-reports', { params }),

  getById: (id: number) => api.get<DailyReport>(`/daily-reports/${id}`),

  generate: () => api.post<{ message: string }>('/daily-reports/generate'),

  resend: (id: number) => api.post<{ message: string }>(`/daily-reports/${id}/resend`),
};
