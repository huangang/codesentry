export const REVIEW_STATUS = {
  PENDING: 'pending',
  PROCESSING: 'processing',
  COMPLETED: 'completed',
  FAILED: 'failed',
} as const;

export type ReviewStatus = typeof REVIEW_STATUS[keyof typeof REVIEW_STATUS];

export const PLATFORMS = {
  GITHUB: 'github',
  GITLAB: 'gitlab',
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
  CUSTOM: 'custom',
} as const;

export type IMBotType = typeof IM_BOT_TYPES[keyof typeof IM_BOT_TYPES];

export const LLM_PROVIDERS = {
  OPENAI: 'openai',
  AZURE: 'azure',
  ANTHROPIC: 'anthropic',
  OTHER: 'other',
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

export function getStatusColor(status: string): 'success' | 'error' | 'processing' | 'default' {
  switch (status) {
    case REVIEW_STATUS.COMPLETED:
      return 'success';
    case REVIEW_STATUS.FAILED:
      return 'error';
    case REVIEW_STATUS.PROCESSING:
    case REVIEW_STATUS.PENDING:
      return 'processing';
    default:
      return 'default';
  }
}
