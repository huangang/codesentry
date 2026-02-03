export const REVIEW_STATUS = {
  PENDING: 'pending',
  PROCESSING: 'processing',
  ANALYZING: 'analyzing',
  COMPLETED: 'completed',
  FAILED: 'failed',
  SKIPPED: 'skipped',
} as const;

export type ReviewStatus = typeof REVIEW_STATUS[keyof typeof REVIEW_STATUS];

export const PLATFORMS = {
  GITHUB: 'github',
  GITLAB: 'gitlab',
  BITBUCKET: 'bitbucket',
} as const;

export type Platform = typeof PLATFORMS[keyof typeof PLATFORMS];

export const EVENT_TYPES = {
  PUSH: 'push',
  MERGE_REQUEST: 'merge_request',
} as const;

export type EventType = typeof EVENT_TYPES[keyof typeof EVENT_TYPES];

export const IM_BOT_TYPES = {
  WECHAT_WORK: 'wechat_work',
  DINGTALK: 'dingtalk',
  FEISHU: 'feishu',
  SLACK: 'slack',
  DISCORD: 'discord',
  TEAMS: 'teams',
  TELEGRAM: 'telegram',
} as const;

export type IMBotType = typeof IM_BOT_TYPES[keyof typeof IM_BOT_TYPES];

export const LLM_PROVIDERS = {
  OPENAI: 'openai',
  AZURE: 'azure',
  ANTHROPIC: 'anthropic',
  OLLAMA: 'ollama',
  GEMINI: 'gemini',
} as const;

export type LLMProvider = typeof LLM_PROVIDERS[keyof typeof LLM_PROVIDERS];

export const SCORE_THRESHOLDS = {
  HIGH: 80,
  MEDIUM: 60,
} as const;

export function getScoreColor(score: number | null): 'success' | 'warning' | 'error' | 'default' {
  if (score === null) return 'default';
  if (score >= SCORE_THRESHOLDS.HIGH) return 'success';
  if (score >= SCORE_THRESHOLDS.MEDIUM) return 'warning';
  return 'error';
}

export function getStatusColor(status: string): 'success' | 'error' | 'processing' | 'default' | 'warning' {
  switch (status) {
    case REVIEW_STATUS.COMPLETED:
      return 'success';
    case REVIEW_STATUS.FAILED:
      return 'error';
    case REVIEW_STATUS.PROCESSING:
    case REVIEW_STATUS.ANALYZING:
    case REVIEW_STATUS.PENDING:
      return 'processing';
    case REVIEW_STATUS.SKIPPED:
      return 'warning';
    default:
      return 'default';
  }
}

export * from './permissions';
