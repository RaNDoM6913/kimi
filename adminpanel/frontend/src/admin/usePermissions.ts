import { createContext, createElement, useContext, useMemo, type ReactNode } from 'react';
import { getPermissionsForRole, type AdminPermission } from '@/admin/permissions';
import type { AdminRole } from '@/admin/roles';

interface PermissionsContextValue {
  role: AdminRole;
  hasPermission: (permission: AdminPermission) => boolean;
}

const PermissionsContext = createContext<PermissionsContextValue | undefined>(undefined);

interface PermissionsProviderProps {
  role: AdminRole;
  children: ReactNode;
}

export function PermissionsProvider({ role, children }: PermissionsProviderProps) {
  const permissionSet = useMemo(() => new Set(getPermissionsForRole(role)), [role]);

  const value = useMemo<PermissionsContextValue>(() => ({
    role,
    hasPermission: (permission: AdminPermission) => permissionSet.has(permission),
  }), [role, permissionSet]);

  return createElement(PermissionsContext.Provider, { value }, children);
}

export function usePermissions() {
  const context = useContext(PermissionsContext);

  if (!context) {
    throw new Error('usePermissions must be used within PermissionsProvider');
  }

  return context;
}
