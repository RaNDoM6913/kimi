import { useState, type ReactElement } from 'react';
import { Sidebar } from '@/components/layout/Sidebar';
import { TopBar } from '@/components/layout/TopBar';
import { OverviewPage } from '@/pages/OverviewPage';
import { UsersPage } from '@/pages/UsersPage';
import { EngagementPage, EngagementExportButton } from '@/pages/EngagementPage';
import { MonetizationPage, MonetizationExportButton } from '@/pages/MonetizationPage';
import { AdsPage, AdsNewCampaignButton } from '@/pages/AdsPage';
import { ModerationPage, ModerationPendingBadge, useModerationPendingCount } from '@/pages/ModerationPage';
import { SystemPage } from '@/pages/SystemPage';
import { RolesPage, RolesCreateRoleButton } from '@/pages/RolesPage';
import { SettingsPage } from '@/pages/SettingsPage';
import { LoginPage } from '@/pages/LoginPage';
import { ADMIN_PERMISSIONS, type AdminPermission } from '@/admin/permissions';
import { DEFAULT_ADMIN_ROLE, isAdminRole, type AdminRole } from '@/admin/roles';
import { PermissionsProvider } from '@/admin/usePermissions';
import { ProtectedRoute } from '@/components/admin/ProtectedRoute';
import { cn } from '@/lib/utils';

const pageTitles: Record<string, string> = {
  overview: 'Overview',
  users: 'Users',
  moderation: 'Moderation',
  engagement: 'Engagement',
  monetization: 'Monetization',
  ads: 'Ads',
  system: 'System',
  roles: 'Roles & Access',
  settings: 'Settings',
};

const pagePermissions: Record<string, AdminPermission> = {
  overview: ADMIN_PERMISSIONS.view_metrics,
  users: ADMIN_PERMISSIONS.view_users,
  moderation: ADMIN_PERMISSIONS.view_reports,
  engagement: ADMIN_PERMISSIONS.view_metrics,
  monetization: ADMIN_PERMISSIONS.view_payments,
  ads: ADMIN_PERMISSIONS.view_ads_metrics,
  system: ADMIN_PERMISSIONS.view_metrics,
  roles: ADMIN_PERMISSIONS.manage_roles,
  settings: ADMIN_PERMISSIONS.change_limits,
};

function resolveInitialRole(): AdminRole {
  if (typeof window === 'undefined') {
    return DEFAULT_ADMIN_ROLE;
  }

  const storedRole = window.localStorage.getItem('adminRole');
  if (isAdminRole(storedRole)) {
    return storedRole;
  }

  return DEFAULT_ADMIN_ROLE;
}

function AppShell() {
  const [activePage, setActivePage] = useState('overview');
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const moderationPendingCount = useModerationPendingCount();

  const renderProtectedPage = (permission: AdminPermission, page: ReactElement) => (
    <ProtectedRoute permission={permission}>{page}</ProtectedRoute>
  );

  const renderPage = () => {
    switch (activePage) {
      case 'overview':
        return renderProtectedPage(pagePermissions.overview, <OverviewPage />);
      case 'users':
        return renderProtectedPage(pagePermissions.users, <UsersPage />);
      case 'moderation':
        return renderProtectedPage(pagePermissions.moderation, <ModerationPage />);
      case 'engagement':
        return renderProtectedPage(pagePermissions.engagement, <EngagementPage />);
      case 'monetization':
        return renderProtectedPage(pagePermissions.monetization, <MonetizationPage />);
      case 'ads':
        return renderProtectedPage(pagePermissions.ads, <AdsPage />);
      case 'system':
        return renderProtectedPage(pagePermissions.system, <SystemPage />);
      case 'roles':
        return renderProtectedPage(pagePermissions.roles, <RolesPage />);
      case 'settings':
        return renderProtectedPage(pagePermissions.settings, <SettingsPage />);
      default:
        return renderProtectedPage(pagePermissions.overview, <OverviewPage />);
    }
  };

  return (
    <div className="min-h-screen bg-[#070B14] flex">
      {/* Noise Overlay */}
      <div className="noise-overlay" />

      {/* Sidebar */}
      <Sidebar
        activePage={activePage}
        onPageChange={setActivePage}
        collapsed={sidebarCollapsed}
        onCollapsedChange={setSidebarCollapsed}
        moderationPendingCount={moderationPendingCount}
      />

      {/* Main Content — margin matches sidebar width (w-20 = 80px, w-60 = 240px) */}
      <div
        className={cn(
          'flex-1 flex flex-col transition-[margin] duration-300',
          sidebarCollapsed ? 'ml-20' : 'ml-60',
        )}
      >
        {/* Top Bar */}
        <TopBar
          pageTitle={pageTitles[activePage] ?? 'Dashboard'}
          leftSlot={
            activePage === 'moderation' ? <ModerationPendingBadge /> :
            activePage === 'engagement' ? <EngagementExportButton /> :
            activePage === 'monetization' ? <MonetizationExportButton /> :
            activePage === 'ads' ? <AdsNewCampaignButton /> :
            activePage === 'roles' ? <RolesCreateRoleButton /> :
            undefined
          }
        />

        {/* Page Content */}
        <main className="flex-1 overflow-auto scrollbar-thin">{renderPage()}</main>
      </div>
    </div>
  );
}

function App() {
  const [currentRole] = useState<AdminRole>(resolveInitialRole);
  const [isAuthenticated, setIsAuthenticated] = useState(false);

  const handleLogin = () => {
    setIsAuthenticated(true);
  };

  if (!isAuthenticated) {
    return <LoginPage onLogin={handleLogin} />;
  }

  return (
    <PermissionsProvider role={currentRole}>
      <AppShell />
    </PermissionsProvider>
  );
}

export default App;
