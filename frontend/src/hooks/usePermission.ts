import { useMemo } from 'react';
import { useAuthStore } from '../stores/authStore';
import { ROLES, ADMIN_ONLY_ROUTES } from '../constants';

export interface UsePermissionReturn {
  isAdmin: boolean;
  canAccess: (route: string) => boolean;
  canWrite: boolean;
}

export function usePermission(): UsePermissionReturn {
  const user = useAuthStore((state) => state.user);

  const isAdmin = useMemo(() => user?.role === ROLES.ADMIN, [user?.role]);

  const canAccess = useMemo(() => {
    return (route: string): boolean => {
      if (isAdmin) return true;
      return !ADMIN_ONLY_ROUTES.some((adminRoute) => route.startsWith(adminRoute));
    };
  }, [isAdmin]);

  const canWrite = isAdmin;

  return { isAdmin, canAccess, canWrite };
}
