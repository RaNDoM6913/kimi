import { useMemo, useState } from 'react';
import {
  LayoutDashboard,
  Users,
  ShieldAlert,
  Heart,
  DollarSign,
  Megaphone,
  FlaskConical,
  Cpu,
  UserCog,
  Settings,
  ChevronLeft,
  ChevronRight,
  HeartPulse,
  type LucideIcon,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { ADMIN_PERMISSIONS, type AdminPermission } from '@/admin/permissions';
import { usePermissions } from '@/admin/usePermissions';

interface SidebarProps {
  activePage: string;
  onPageChange: (page: string) => void;
  collapsed?: boolean;
  onCollapsedChange?: (collapsed: boolean) => void;
}

interface NavItem {
  id: string;
  label: string;
  icon: LucideIcon;
  permission: AdminPermission;
  badge?: number;
}

const navItems: NavItem[] = [
  { id: 'overview', label: 'Overview', icon: LayoutDashboard, permission: ADMIN_PERMISSIONS.view_metrics },
  { id: 'users', label: 'Users', icon: Users, permission: ADMIN_PERMISSIONS.view_users },
  { id: 'moderation', label: 'Moderation', icon: ShieldAlert, permission: ADMIN_PERMISSIONS.view_reports, badge: 6 },
  { id: 'engagement', label: 'Engagement', icon: Heart, permission: ADMIN_PERMISSIONS.view_metrics },
  { id: 'monetization', label: 'Monetization', icon: DollarSign, permission: ADMIN_PERMISSIONS.view_payments },
  { id: 'ads', label: 'Ads', icon: Megaphone, permission: ADMIN_PERMISSIONS.view_ads_metrics },
  { id: 'experiments', label: 'Experiments', icon: FlaskConical, permission: ADMIN_PERMISSIONS.manage_experiments },
  { id: 'system', label: 'System', icon: Cpu, permission: ADMIN_PERMISSIONS.view_metrics },
  { id: 'roles', label: 'Roles & Access', icon: UserCog, permission: ADMIN_PERMISSIONS.manage_roles },
  { id: 'settings', label: 'Settings', icon: Settings, permission: ADMIN_PERMISSIONS.change_limits },
];

export function Sidebar({ activePage, onPageChange, collapsed: collapsedProp, onCollapsedChange }: SidebarProps) {
  const [internalCollapsed, setInternalCollapsed] = useState(false);
  const isControlled = onCollapsedChange != null;
  const collapsed = isControlled ? (collapsedProp ?? false) : internalCollapsed;

  const setCollapsed = (value: boolean) => {
    if (isControlled) {
      onCollapsedChange?.(value);
    } else {
      setInternalCollapsed(value);
    }
  };
  const { hasPermission, role } = usePermissions();

  const visibleNavItems = useMemo(
    () => navItems.filter((item) => hasPermission(item.permission)),
    [hasPermission],
  );

  return (
    <aside
      className={cn(
        'fixed left-0 top-0 h-screen bg-[#0E1320] border-r border-[rgba(123,97,255,0.12)] flex flex-col transition-[width] duration-300 z-50',
        collapsed ? 'w-20' : 'w-64',
      )}
    >
      {/* Logo */}
      <div className="h-16 flex items-center px-4 border-b border-[rgba(123,97,255,0.12)]">
        <div className="flex items-center gap-3">
          <div className="w-9 h-9 rounded-xl bg-gradient-to-br from-[#7B61FF] to-[#4CC9F0] flex items-center justify-center flex-shrink-0">
            <HeartPulse className="w-5 h-5 text-white" />
          </div>
          {!collapsed && (
            <div>
              <h1 className="text-lg font-bold text-[#F5F7FF]">Heartbeat</h1>
              <p className="text-[10px] text-[#A7B1C8] -mt-0.5">Admin Console</p>
            </div>
          )}
        </div>
      </div>

      {/* Navigation */}
      <nav className="flex-1 py-4 px-3 overflow-y-auto scrollbar-thin">
        {visibleNavItems.length === 0 ? (
          <div className="px-3 py-2 text-xs text-[#A7B1C8]">No accessible sections</div>
        ) : (
          <ul className="space-y-1">
            {visibleNavItems.map((item) => {
              const Icon = item.icon;
              const isActive = activePage === item.id;

              return (
                <li key={item.id}>
                  <button
                    onClick={() => onPageChange(item.id)}
                    className={cn(
                      'w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all duration-180 relative',
                      isActive
                        ? 'bg-[rgba(123,97,255,0.12)] text-[#F5F7FF]'
                        : 'text-[rgba(245,247,255,0.75)] hover:bg-[rgba(123,97,255,0.08)] hover:text-[#F5F7FF]',
                    )}
                  >
                    {isActive && (
                      <span className="absolute left-0 top-1/2 -translate-y-1/2 w-[3px] h-5 bg-[#7B61FF] rounded-r" />
                    )}
                    <Icon className={cn('w-5 h-5 flex-shrink-0', isActive && 'text-[#7B61FF]')} />
                    {!collapsed && (
                      <>
                        <span className="flex-1 text-left">{item.label}</span>
                        {item.badge && (
                          <span className="px-2 py-0.5 bg-[rgba(255,107,107,0.2)] text-[#FF6B6B] text-xs rounded-full">
                            {item.badge}
                          </span>
                        )}
                      </>
                    )}
                  </button>
                </li>
              );
            })}
          </ul>
        )}
      </nav>

      {/* Collapse Button — fixed height so nav doesn't reflow on expand */}
      <div className="flex-shrink-0 p-3 border-t border-[rgba(123,97,255,0.12)]">
        <button
          onClick={() => setCollapsed(!collapsed)}
          aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          className="w-full flex items-center justify-center p-2 rounded-lg text-[#A7B1C8] hover:bg-[rgba(123,97,255,0.08)] hover:text-[#F5F7FF] transition-colors"
        >
          {collapsed ? <ChevronRight className="w-5 h-5" /> : <ChevronLeft className="w-5 h-5" />}
        </button>
      </div>

      {/* Admin Profile — min-height so height doesn't change on expand, nav doesn't jump */}
      <div
        className={cn(
          'flex-shrink-0 min-h-[73px] border-t border-[rgba(123,97,255,0.12)] flex items-center p-3',
          collapsed ? 'justify-center' : 'pl-5 pr-3',
        )}
      >
        <div className={cn('flex items-center gap-3', collapsed && 'justify-center')}>
          <div className="relative flex-shrink-0">
            <img
              src="https://i.pravatar.cc/150?u=admin"
              alt="Admin"
              className="w-10 h-10 rounded-full border-2 border-[rgba(123,97,255,0.25)]"
            />
            <span className="absolute bottom-0 right-0 w-3 h-3 bg-[#2DD4A8] rounded-full border-2 border-[#0E1320]" />
          </div>
          {!collapsed && (
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium text-[#F5F7FF] truncate">Alex Morgan</p>
              <p className="text-xs text-[#A7B1C8] truncate">{role}</p>
            </div>
          )}
        </div>
      </div>
    </aside>
  );
}
