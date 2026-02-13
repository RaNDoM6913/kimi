import { useState, type ReactElement } from 'react';
import { Sidebar } from '@/components/layout/Sidebar';
import { TopBar } from '@/components/layout/TopBar';
import { OverviewPage } from '@/pages/OverviewPage';
import { UsersPage } from '@/pages/UsersPage';
import { EngagementPage } from '@/pages/EngagementPage';
import { MonetizationPage } from '@/pages/MonetizationPage';
import { AdsPage } from '@/pages/AdsPage';
import { ModerationPage } from '@/pages/ModerationPage';
import { ExperimentsPage } from '@/pages/ExperimentsPage';
import { SystemPage } from '@/pages/SystemPage';
import { RolesPage } from '@/pages/RolesPage';
import { SettingsPage } from '@/pages/SettingsPage';
import { ADMIN_PERMISSIONS, type AdminPermission } from '@/admin/permissions';
import { DEFAULT_ADMIN_ROLE, isAdminRole, type AdminRole } from '@/admin/roles';
import { PermissionsProvider } from '@/admin/usePermissions';
import { ProtectedRoute } from '@/components/admin/ProtectedRoute';

const pageTitles: Record<string, string> = {
  overview: 'Overview',
  users: 'Users',
  moderation: 'Moderation',
  engagement: 'Engagement',
  monetization: 'Monetization',
  ads: 'Ads',
  experiments: 'Experiments',
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
  experiments: ADMIN_PERMISSIONS.manage_experiments,
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
      case 'experiments':
        return renderProtectedPage(pagePermissions.experiments, <ExperimentsPage />);
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
      <Sidebar activePage={activePage} onPageChange={setActivePage} />

      {/* Main Content */}
      <div className="flex-1 ml-64 flex flex-col">
        {/* Top Bar */}
        <TopBar pageTitle={pageTitles[activePage] ?? 'Dashboard'} />

        {/* Page Content */}
        <main className="flex-1 overflow-auto scrollbar-thin">{renderPage()}</main>
      </div>
    </div>
  );
}

function App() {
  const [currentRole] = useState<AdminRole>(resolveInitialRole);

  return (
    <PermissionsProvider role={currentRole}>
      <AppShell />
    </PermissionsProvider>
  );
}

export default App;
