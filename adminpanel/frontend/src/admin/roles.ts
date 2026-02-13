export const ADMIN_ROLES = {
  OWNER: 'OWNER',
  ADMIN: 'ADMIN',
  MODERATOR: 'MODERATOR',
  SUPPORT: 'SUPPORT',
  AD_MANAGER: 'AD_MANAGER',
} as const;

export type AdminRole = (typeof ADMIN_ROLES)[keyof typeof ADMIN_ROLES];

export const DEFAULT_ADMIN_ROLE: AdminRole = ADMIN_ROLES.ADMIN;

export function isAdminRole(value: string | null | undefined): value is AdminRole {
  if (!value) {
    return false;
  }

  return Object.values(ADMIN_ROLES).includes(value as AdminRole);
}
