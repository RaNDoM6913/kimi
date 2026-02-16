import { ADMIN_ROLES, type AdminRole } from '@/admin/roles';

export const ADMIN_PERMISSIONS = {
  view_users: 'view_users',
  edit_users: 'edit_users',
  ban_users: 'ban_users',
  delete_users: 'delete_users',
  view_private_data: 'view_private_data',
  view_reports: 'view_reports',
  moderate_profiles: 'moderate_profiles',
  approve_profiles: 'approve_profiles',
  reject_profiles: 'reject_profiles',
  view_metrics: 'view_metrics',
  export_metrics: 'export_metrics',
  view_payments: 'view_payments',
  refund_payments: 'refund_payments',
  view_ads_metrics: 'view_ads_metrics',
  export_ads_metrics: 'export_ads_metrics',
  manage_flags: 'manage_flags',
  change_limits: 'change_limits',
  run_recalculation: 'run_recalculation',
  manage_roles: 'manage_roles',
  view_audit_logs: 'view_audit_logs',
  view_owner_logs: 'view_owner_logs',
  impersonate_admin: 'impersonate_admin',
  disable_admin_2fa: 'disable_admin_2fa',
} as const;

export type AdminPermission = (typeof ADMIN_PERMISSIONS)[keyof typeof ADMIN_PERMISSIONS];

const ALL_PERMISSIONS: AdminPermission[] = Object.values(ADMIN_PERMISSIONS);

export const ROLE_PERMISSIONS: Record<AdminRole, AdminPermission[]> = {
  [ADMIN_ROLES.OWNER]: ALL_PERMISSIONS,
  [ADMIN_ROLES.ADMIN]: [
    ADMIN_PERMISSIONS.view_users,
    ADMIN_PERMISSIONS.edit_users,
    ADMIN_PERMISSIONS.ban_users,
    ADMIN_PERMISSIONS.delete_users,
    ADMIN_PERMISSIONS.view_private_data,
    ADMIN_PERMISSIONS.view_reports,
    ADMIN_PERMISSIONS.moderate_profiles,
    ADMIN_PERMISSIONS.approve_profiles,
    ADMIN_PERMISSIONS.reject_profiles,
    ADMIN_PERMISSIONS.view_metrics,
    ADMIN_PERMISSIONS.export_metrics,
    ADMIN_PERMISSIONS.view_payments,
    ADMIN_PERMISSIONS.refund_payments,
    ADMIN_PERMISSIONS.view_ads_metrics,
    ADMIN_PERMISSIONS.export_ads_metrics,
    ADMIN_PERMISSIONS.manage_flags,
    ADMIN_PERMISSIONS.change_limits,
    ADMIN_PERMISSIONS.run_recalculation,
    ADMIN_PERMISSIONS.manage_roles,
    ADMIN_PERMISSIONS.view_audit_logs,
  ],
  [ADMIN_ROLES.MODERATOR]: [
    ADMIN_PERMISSIONS.view_users,
    ADMIN_PERMISSIONS.ban_users,
    ADMIN_PERMISSIONS.view_private_data,
    ADMIN_PERMISSIONS.view_reports,
    ADMIN_PERMISSIONS.moderate_profiles,
    ADMIN_PERMISSIONS.approve_profiles,
    ADMIN_PERMISSIONS.reject_profiles,
    ADMIN_PERMISSIONS.manage_flags,
    ADMIN_PERMISSIONS.view_metrics,
  ],
  [ADMIN_ROLES.SUPPORT]: [
    ADMIN_PERMISSIONS.view_users,
    ADMIN_PERMISSIONS.edit_users,
    ADMIN_PERMISSIONS.view_private_data,
    ADMIN_PERMISSIONS.view_reports,
    ADMIN_PERMISSIONS.view_metrics,
    ADMIN_PERMISSIONS.view_payments,
    ADMIN_PERMISSIONS.refund_payments,
    ADMIN_PERMISSIONS.change_limits,
    ADMIN_PERMISSIONS.view_audit_logs,
  ],
  [ADMIN_ROLES.AD_MANAGER]: [
    ADMIN_PERMISSIONS.view_metrics,
    ADMIN_PERMISSIONS.export_metrics,
    ADMIN_PERMISSIONS.view_ads_metrics,
    ADMIN_PERMISSIONS.export_ads_metrics,
  ],
};

export function getPermissionsForRole(role: AdminRole): AdminPermission[] {
  return ROLE_PERMISSIONS[role] ?? [];
}
