import { useMemo } from 'react';
import { useAuthStore } from '../stores/authStore';
import { ROLES, ADMIN_ONLY_ROUTES, hasWriteAccess } from '../constants';

export interface UsePermissionReturn {
  isAdmin: boolean;
  isDeveloper: boolean;
  canAccess: (route: string) => boolean;
  canWrite: boolean;
}

export function usePermission(): UsePermissionReturn {
  const user = useAuthStore((state) => state.user);

  const isAdmin = useMemo(() => user?.role === ROLES.ADMIN, [user?.role]);
  const isDeveloper = useMemo(() => user?.role === ROLES.DEVELOPER, [user?.role]);

  const canAccess = useMemo(() => {
    return (route: string): boolean => {
      if (isAdmin) return true;
      return !ADMIN_ONLY_ROUTES.some((adminRoute) => route.startsWith(adminRoute));
    };
  }, [isAdmin]);

  const canWrite = useMemo(() => hasWriteAccess(user?.role || ''), [user?.role]);

  return { isAdmin, isDeveloper, canAccess, canWrite };
}
