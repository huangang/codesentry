// API Types

export interface User {
  id: number;
  username: string;
  email: string;
  nickname: string;
  avatar: string;
  role: string;
  auth_type: string;
  is_active: boolean;
  last_login: string | null;
  created_at: string;
  updated_at: string;
}

export interface Project {
  id: number;
  name: string;
  url: string;
  platform: 'github' | 'gitlab' | 'bitbucket';
  file_extensions: string;
  ignore_patterns: string;
  branch_filter: string;
  review_events: string;
  ai_enabled: boolean;
  ai_prompt: string;
  ai_prompt_id: number | null;
  llm_config_id: number | null;
  im_enabled: boolean;
  im_bot_id: number | null;
  comment_enabled: boolean;
  created_by: number;
  created_at: string;
  updated_at: string;
  min_score: number;
}

export interface ReviewLog {
  id: number;
  project_id: number;
  project?: Project;
  event_type: 'push' | 'merge_request';
  commit_hash: string;
  commit_url: string;
  branch: string;
  author: string;
  author_email: string;
  author_avatar: string;
  author_url: string;
  commit_message: string;
  files_changed: number;
  additions: number;
  deletions: number;
  score: number | null;
  original_score: number | null;
  score_override_reason: string;
  review_result: string;
  review_status: 'pending' | 'processing' | 'analyzing' | 'completed' | 'failed' | 'skipped';
  error_message: string;
  retry_count: number;
  mr_number: number | null;
  mr_url: string;
  created_at: string;
  updated_at: string;
}

export interface LLMConfig {
  id: number;
  name: string;
  provider: string;
  base_url: string;
  api_key_mask: string;
  model: string;
  max_tokens: number;
  temperature: number;
  is_default: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface IMBot {
  id: number;
  name: string;
  type: 'wechat_work' | 'dingtalk' | 'feishu' | 'slack' | 'discord' | 'teams' | 'telegram';
  webhook: string;
  extra: string;
  is_active: boolean;
  error_notify: boolean;
  daily_report_enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface PromptTemplate {
  id: number;
  name: string;
  description: string;
  content: string;
  variables: string;
  is_default: boolean;
  is_system: boolean;
  created_by: number;
  created_at: string;
  updated_at: string;
}

export interface SystemLog {
  id: number;
  level: 'info' | 'warning' | 'error';
  module: string;
  action: string;
  message: string;
  user_id: number | null;
  ip: string;
  user_agent: string;
  extra: string;
  created_at: string;
}

export interface GitCredential {
  id: number;
  name: string;
  platform: 'github' | 'gitlab' | 'bitbucket';
  base_url: string;
  access_token_mask: string;
  webhook_secret_set: boolean;
  auto_create: boolean;
  default_enabled: boolean;
  file_extensions: string;
  review_events: string;
  ignore_patterns: string;
  is_active: boolean;
  created_by: number;
  created_at: string;
  updated_at: string;
}

// API Response Types
export interface PaginatedResponse<T> {
  total: number;
  page: number;
  page_size: number;
  items: T[];
}

export interface LoginResponse {
  token: string;
  user: User;
  expire_at: string;
}

export interface AuthConfig {
  ldap_enabled: boolean;
}

export interface DashboardStats {
  active_projects: number;
  contributors: number;
  total_commits: number;
  average_score: number;
}

export interface ProjectStats {
  project_id: number;
  project_name: string;
  commit_count: number;
  avg_score: number;
  additions: number;
  deletions: number;
}

export interface AuthorStats {
  author: string;
  commit_count: number;
  avg_score: number;
  additions: number;
  deletions: number;
}

export interface DashboardResponse {
  stats: DashboardStats;
  project_stats: ProjectStats[];
  author_stats: AuthorStats[];
}

export interface LDAPConfig {
  enabled: boolean;
  host: string;
  port: number;
  base_dn: string;
  bind_dn: string;
  bind_password: string;
  user_filter: string;
  use_ssl: boolean;
}
