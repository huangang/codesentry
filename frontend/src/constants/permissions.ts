export const ROLES = {
  ADMIN: 'admin',
  USER: 'user',
} as const;

export type Role = typeof ROLES[keyof typeof ROLES];

export const ADMIN_ONLY_ROUTES = [
  '/admin/llm-models',
  '/admin/prompts',
  '/admin/im-bots',
  '/admin/git-credentials',
  '/admin/sys-logs',
  '/admin/settings',
] as const;

export const isAdminOnlyRoute = (path: string): boolean => {
  return ADMIN_ONLY_ROUTES.some(route => path.startsWith(route));
};
