import type { AdminRole } from '@/admin/roles';

export interface AdminActor {
  id: string;
  role: AdminRole;
}

export interface AdminAuditEntry {
  action: string;
  actor: AdminActor;
  ip: string;
  device: string;
  timestamp: string;
}

export function getClientDevice(): string {
  if (typeof navigator === 'undefined') {
    return 'unknown-device';
  }

  return navigator.userAgent;
}

export function logAdminAction(
  action: string,
  actor: AdminActor,
  ip: string,
  device: string,
): AdminAuditEntry {
  const entry: AdminAuditEntry = {
    action,
    actor,
    ip,
    device,
    timestamp: new Date().toISOString(),
  };

  // TODO: replace with backend audit endpoint once API is wired.
  console.info('[admin-audit]', entry);

  return entry;
}
