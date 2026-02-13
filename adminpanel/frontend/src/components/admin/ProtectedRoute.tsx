import type { ReactNode } from 'react';
import type { AdminPermission } from '@/admin/permissions';
import { usePermissions } from '@/admin/usePermissions';

interface ProtectedRouteProps {
  permission: AdminPermission;
  children: ReactNode;
}

export function ProtectedRoute({ permission, children }: ProtectedRouteProps) {
  const { hasPermission } = usePermissions();

  if (!hasPermission(permission)) {
    return (
      <div className="p-6">
        <div className="glass-panel p-8 text-center">
          <p className="text-2xl font-bold text-[#F5F7FF]">Access denied</p>
          <p className="mt-2 text-sm text-[#A7B1C8]">You do not have permission to access this section.</p>
        </div>
      </div>
    );
  }

  return <>{children}</>;
}
