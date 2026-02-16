import { useState } from 'react';
import { roles, permissions } from '@/data/mockData';
import { 
  Check, 
  Users, 
  Shield, 
  Edit, 
  Trash2, 
  Plus,
  ChevronRight,
  Search
} from 'lucide-react';
import { cn } from '@/lib/utils';
import type { Role, Permission } from '@/types';
import { ADMIN_PERMISSIONS } from '@/admin/permissions';
import { usePermissions } from '@/admin/usePermissions';
import { getClientDevice, logAdminAction } from '@/admin/audit';

function PermissionToggle({ 
  permission, 
  checked, 
  onChange,
  disabled,
}: { 
  permission: Permission; 
  checked: boolean;
  onChange: () => void;
  disabled: boolean;
}) {
  return (
    <div className="flex items-center justify-between p-3 rounded-lg bg-[rgba(14,19,32,0.5)] hover:bg-[rgba(123,97,255,0.05)] transition-colors">
      <div>
        <p className="text-sm font-medium text-[#F5F7FF]">{permission.name}</p>
        <p className="text-xs text-[#A7B1C8]">{permission.description}</p>
      </div>
      <button
        onClick={onChange}
        disabled={disabled}
        className={cn(
          "w-12 h-6 rounded-full transition-colors relative disabled:opacity-50 disabled:cursor-not-allowed",
          checked ? "bg-[#7B61FF]" : "bg-[rgba(123,97,255,0.2)]"
        )}
      >
        <span className={cn(
          "absolute top-1 w-4 h-4 rounded-full bg-white transition-all",
          checked ? "left-7" : "left-1"
        )} />
      </button>
    </div>
  );
}

/** "Create Role" button for TopBar when on Roles page */
export function RolesCreateRoleButton() {
  const { hasPermission, role } = usePermissions();
  const canManageRoles = hasPermission(ADMIN_PERMISSIONS.manage_roles);

  const logRoleAction = (action: string) => {
    logAdminAction(action, { id: 'current-admin', role }, '127.0.0.1', getClientDevice());
  };

  return (
    <button
      onClick={() => canManageRoles && logRoleAction('create_role')}
      disabled={!canManageRoles}
      className="btn-primary flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
    >
      <Plus className="w-4 h-4" />
      Create Role
    </button>
  );
}

export function RolesPage() {
  const [selectedRole, setSelectedRole] = useState<Role>(roles[0]);
  const [searchQuery, setSearchQuery] = useState('');
  const [rolePermissions, setRolePermissions] = useState<Set<string>>(
    new Set(selectedRole.permissions.map(p => p.id))
  );
  const { hasPermission, role } = usePermissions();
  const canManageRoles = hasPermission(ADMIN_PERMISSIONS.manage_roles);

  const togglePermission = (permissionId: string) => {
    if (!canManageRoles) {
      return;
    }

    const newSet = new Set(rolePermissions);
    if (newSet.has(permissionId)) {
      newSet.delete(permissionId);
    } else {
      newSet.add(permissionId);
    }
    setRolePermissions(newSet);
  };

  const logRoleAction = (action: string) => {
    logAdminAction(action, { id: 'current-admin', role }, '127.0.0.1', getClientDevice());
  };

  const handleRoleChange = (role: Role) => {
    setSelectedRole(role);
    setRolePermissions(new Set(role.permissions.map(p => p.id)));
  };

  const groupedPermissions = permissions.reduce((acc, permission) => {
    if (!acc[permission.category]) {
      acc[permission.category] = [];
    }
    acc[permission.category].push(permission);
    return acc;
  }, {} as Record<string, Permission[]>);

  return (
    <div className="p-6 h-[calc(100vh-64px)] animate-fade-in">
      {/* Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-6">
        {[
          { label: 'Total Roles', value: roles.length, icon: Shield, color: '#7B61FF' },
          { label: 'Total Users', value: '59', icon: Users, color: '#2DD4A8' },
          { label: 'Permissions', value: permissions.length, icon: Check, color: '#4CC9F0' },
        ].map((stat, i) => (
          <div key={i} className="glass-panel p-4 flex items-center gap-4">
            <div 
              className="w-10 h-10 rounded-lg flex items-center justify-center"
              style={{ background: `${stat.color}20` }}
            >
              <stat.icon className="w-5 h-5" style={{ color: stat.color }} />
            </div>
            <div>
              <p className="text-sm text-[#A7B1C8]">{stat.label}</p>
              <p className="text-xl font-bold text-[#F5F7FF]">{stat.value}</p>
            </div>
          </div>
        ))}
      </div>

      {/* Main Content */}
      <div className="glass-panel h-[calc(100%-140px)] overflow-hidden flex">
        {/* Roles List */}
        <div className="w-72 border-r border-[rgba(123,97,255,0.12)] flex flex-col">
          <div className="p-3 border-b border-[rgba(123,97,255,0.12)]">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[#A7B1C8]" />
              <input
                type="text"
                placeholder="Search roles..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full pl-9 pr-3 py-2 rounded-lg text-sm bg-[rgba(14,19,32,0.8)] border border-[rgba(123,97,255,0.18)] text-[#F5F7FF] placeholder:text-[rgba(167,177,200,0.6)] focus:border-[#7B61FF] focus:outline-none"
              />
            </div>
          </div>

          <div className="flex-1 overflow-y-auto scrollbar-thin">
            {roles
              .filter(role => role.name.toLowerCase().includes(searchQuery.toLowerCase()))
              .map((role) => (
                <button
                  key={role.id}
                  onClick={() => handleRoleChange(role)}
                  className={cn(
                    "w-full p-4 text-left border-b border-[rgba(123,97,255,0.08)] transition-colors",
                    selectedRole?.id === role.id
                      ? "bg-[rgba(123,97,255,0.1)]"
                      : "hover:bg-[rgba(123,97,255,0.05)]"
                  )}
                >
                  <div className="flex items-center justify-between mb-1">
                    <p className="text-sm font-medium text-[#F5F7FF]">{role.name}</p>
                    <ChevronRight className={cn(
                      "w-4 h-4 text-[#A7B1C8]",
                      selectedRole?.id === role.id && "text-[#7B61FF]"
                    )} />
                  </div>
                  <p className="text-xs text-[#A7B1C8] line-clamp-1">{role.description}</p>
                  <div className="flex items-center gap-3 mt-2">
                    <span className="flex items-center gap-1 text-xs text-[#A7B1C8]">
                      <Users className="w-3 h-3" />
                      {role.userCount} users
                    </span>
                    <span className="flex items-center gap-1 text-xs text-[#A7B1C8]">
                      <Check className="w-3 h-3" />
                      {role.permissions.length} permissions
                    </span>
                  </div>
                </button>
              ))}
          </div>
        </div>

        {/* Permissions Matrix */}
        <div className="flex-1 flex flex-col">
          {/* Role Header */}
          <div className="p-4 border-b border-[rgba(123,97,255,0.12)] flex items-center justify-between">
            <div>
              <h3 className="text-lg font-semibold text-[#F5F7FF]">{selectedRole.name}</h3>
              <p className="text-sm text-[#A7B1C8]">{selectedRole.description}</p>
            </div>
            <div className="flex items-center gap-2">
              <button
                onClick={() => canManageRoles && logRoleAction(`edit_role_${selectedRole.id}`)}
                disabled={!canManageRoles}
                className="p-2 rounded-lg text-[#A7B1C8] hover:bg-[rgba(123,97,255,0.1)] hover:text-[#F5F7FF] transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <Edit className="w-4 h-4" />
              </button>
              <button
                onClick={() => canManageRoles && logRoleAction(`delete_role_${selectedRole.id}`)}
                disabled={!canManageRoles}
                className="p-2 rounded-lg text-[#A7B1C8] hover:bg-[rgba(255,107,107,0.15)] hover:text-[#FF6B6B] transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <Trash2 className="w-4 h-4" />
              </button>
            </div>
          </div>

          {/* Permissions List */}
          <div className="flex-1 overflow-y-auto scrollbar-thin p-4">
            {Object.entries(groupedPermissions).map(([category, perms]) => (
              <div key={category} className="mb-6">
                <h4 className="text-sm font-medium text-[#A7B1C8] uppercase tracking-wider mb-3">
                  {category}
                </h4>
                <div className="space-y-2">
                  {perms.map((permission) => (
                    <PermissionToggle
                      key={permission.id}
                      permission={permission}
                      checked={rolePermissions.has(permission.id)}
                      onChange={() => togglePermission(permission.id)}
                      disabled={!canManageRoles}
                    />
                  ))}
                </div>
              </div>
            ))}
          </div>

          {/* Footer Actions */}
          <div className="p-4 border-t border-[rgba(123,97,255,0.12)] flex items-center justify-between">
            <p className="text-sm text-[#A7B1C8]">
              {rolePermissions.size} of {permissions.length} permissions enabled
            </p>
            <div className="flex items-center gap-3">
              <button
                onClick={() => {
                  if (!canManageRoles) {
                    return;
                  }
                  setRolePermissions(new Set(selectedRole.permissions.map(p => p.id)));
                  logRoleAction(`reset_role_permissions_${selectedRole.id}`);
                }}
                disabled={!canManageRoles}
                className="px-4 py-2 rounded-lg text-sm text-[#A7B1C8] hover:text-[#F5F7FF] transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Reset
              </button>
              <button
                onClick={() => canManageRoles && logRoleAction(`save_role_permissions_${selectedRole.id}`)}
                disabled={!canManageRoles}
                className="btn-primary disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Save Changes
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
