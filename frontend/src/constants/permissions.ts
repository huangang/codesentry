export const ROLES = {
  ADMIN: 'admin',
  DEVELOPER: 'developer',
  USER: 'user',
} as const;

export type Role = typeof ROLES[keyof typeof ROLES];

export const ADMIN_ONLY_ROUTES = [
  '/admin/llm-models',
  '/admin/im-bots',
  '/admin/daily-reports',
  '/admin/git-credentials',
  '/admin/users',
  '/admin/sys-logs',
  '/admin/settings',
] as const;

export const isAdminOnlyRoute = (path: string): boolean => {
  return ADMIN_ONLY_ROUTES.some(route => path.startsWith(route));
};

// Check if a role has write access (admin or developer)
export const hasWriteAccess = (role: string): boolean => {
  return role === ROLES.ADMIN || role === ROLES.DEVELOPER;
};
