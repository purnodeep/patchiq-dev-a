import { createBrowserRouter } from 'react-router';
import { AppLayout } from './layout/AppLayout';
import DashboardPage from '../pages/dashboard/DashboardPage';
import { EndpointsPage } from '../pages/endpoints/EndpointsPage';
import { EndpointDetailPage } from '../pages/endpoints/EndpointDetailPage';
import { PatchesPage } from '../pages/patches/PatchesPage';
import { PatchDetailPage } from '../pages/patches/PatchDetailPage';
import { CVEsPage } from '../pages/cves/CVEsPage';
import { CVEDetailPage } from '../pages/cves/CVEDetailPage';
import { PoliciesPage } from '../pages/policies/PoliciesPage';
import { CreatePolicyPage } from '../pages/policies/CreatePolicyPage';
import { PolicyDetailPage } from '../pages/policies/PolicyDetailPage';
import { EditPolicyPage } from '../pages/policies/EditPolicyPage';
import { DeploymentsPage } from '../pages/deployments/DeploymentsPage';
import { DeploymentDetailPage } from '../pages/deployments/DeploymentDetailPage';
import { AuditPage } from '../pages/audit/AuditPage';
import { ReportsPage } from '../pages/reports/ReportsPage';
import { GeneralSettingsPage } from '../pages/settings/GeneralSettingsPage';
import { IdentitySettingsPage } from '../pages/settings/IdentitySettingsPage';
import { LicenseSettingsPage } from '../pages/settings/LicenseSettingsPage';
import { AppearanceSettingsPage } from '../pages/settings/AppearanceSettingsPage';
import { PatchSourcesSettingsPage } from '../pages/settings/PatchSourcesSettingsPage';
import { AgentFleetSettingsPage } from '../pages/settings/AgentFleetSettingsPage';
import { AccountSettingsPage } from '../pages/settings/AccountSettingsPage';
import { AboutSettingsPage } from '../pages/settings/AboutSettingsPage';
import { Navigate } from 'react-router';
import { RolesPage } from '../pages/admin/roles/RolesPage';
import { RoleEditPage } from '../pages/admin/roles/RoleEditPage';
import { UserRolesPage } from '../pages/admin/users/UserRolesPage';
import { NotificationsPage } from '../pages/notifications/NotificationsPage';
import { LoginPage } from '../pages/login';
import { ForgotPasswordPage } from '../pages/login/ForgotPasswordPage';
import { RegisterPage } from '../pages/login/RegisterPage';
import { ComponentPreview } from '../pages/preview/ComponentPreview';
import { TagsPage } from '../pages/tags/TagsPage';
import { AgentDownloadsPage } from '../pages/agent-downloads/AgentDownloadsPage';
import { AlertsPage } from '../pages/alerts/AlertsPage';
import { RequirePermission } from './auth/RequirePermission';

export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  { path: '/forgot-password', element: <ForgotPasswordPage /> },
  { path: '/register', element: <RegisterPage /> },
  ...(import.meta.env.DEV ? [{ path: '/preview', element: <ComponentPreview /> }] : []),
  {
    element: <AppLayout />,
    children: [
      {
        path: '/',
        element: (
          <RequirePermission resource="endpoints" action="read">
            <DashboardPage />
          </RequirePermission>
        ),
      },
      {
        path: '/endpoints',
        element: (
          <RequirePermission resource="endpoints" action="read">
            <EndpointsPage />
          </RequirePermission>
        ),
      },
      {
        path: '/endpoints/:id',
        element: (
          <RequirePermission resource="endpoints" action="read">
            <EndpointDetailPage />
          </RequirePermission>
        ),
      },
      { path: '/tags', element: <Navigate to="/settings/tags" replace /> },
      {
        path: '/patches',
        element: (
          <RequirePermission resource="patches" action="read">
            <PatchesPage />
          </RequirePermission>
        ),
      },
      {
        path: '/patches/:id',
        element: (
          <RequirePermission resource="patches" action="read">
            <PatchDetailPage />
          </RequirePermission>
        ),
      },
      {
        path: '/cves',
        element: (
          <RequirePermission resource="patches" action="read">
            <CVEsPage />
          </RequirePermission>
        ),
      },
      {
        path: '/cves/:id',
        element: (
          <RequirePermission resource="patches" action="read">
            <CVEDetailPage />
          </RequirePermission>
        ),
      },
      {
        path: '/policies',
        element: (
          <RequirePermission resource="policies" action="read">
            <PoliciesPage />
          </RequirePermission>
        ),
      },
      {
        path: '/policies/new',
        element: (
          <RequirePermission resource="policies" action="read">
            <CreatePolicyPage />
          </RequirePermission>
        ),
      },
      {
        path: '/policies/:id',
        element: (
          <RequirePermission resource="policies" action="read">
            <PolicyDetailPage />
          </RequirePermission>
        ),
      },
      {
        path: '/policies/:id/edit',
        element: (
          <RequirePermission resource="policies" action="read">
            <EditPolicyPage />
          </RequirePermission>
        ),
      },
      {
        path: '/deployments',
        element: (
          <RequirePermission resource="deployments" action="read">
            <DeploymentsPage />
          </RequirePermission>
        ),
      },
      {
        path: '/deployments/:id',
        element: (
          <RequirePermission resource="deployments" action="read">
            <DeploymentDetailPage />
          </RequirePermission>
        ),
      },
      {
        path: '/audit',
        element: (
          <RequirePermission resource="audit" action="read">
            <AuditPage />
          </RequirePermission>
        ),
      },
      {
        path: '/reports',
        element: (
          <RequirePermission resource="reports" action="read">
            <ReportsPage />
          </RequirePermission>
        ),
      },
      {
        path: '/alerts',
        element: (
          <RequirePermission resource="alerts" action="read">
            <AlertsPage />
          </RequirePermission>
        ),
      },
      { path: '/notifications', element: <Navigate to="/settings/notifications" replace /> },
      { path: '/settings', element: <Navigate to="/settings/general" replace /> },
      {
        path: '/settings/general',
        element: (
          <RequirePermission resource="settings" action="read">
            <GeneralSettingsPage />
          </RequirePermission>
        ),
      },
      {
        path: '/settings/identity',
        element: (
          <RequirePermission resource="settings" action="read">
            <IdentitySettingsPage />
          </RequirePermission>
        ),
      },
      {
        path: '/settings/patch-sources',
        element: (
          <RequirePermission resource="settings" action="read">
            <PatchSourcesSettingsPage />
          </RequirePermission>
        ),
      },
      {
        path: '/settings/agent-fleet',
        element: (
          <RequirePermission resource="settings" action="read">
            <AgentFleetSettingsPage />
          </RequirePermission>
        ),
      },
      {
        path: '/settings/notifications',
        element: (
          <RequirePermission resource="settings" action="read">
            <NotificationsPage />
          </RequirePermission>
        ),
      },
      { path: '/settings/account', element: <AccountSettingsPage /> },
      { path: '/settings/license', element: <LicenseSettingsPage /> },
      { path: '/settings/appearance', element: <AppearanceSettingsPage /> },
      { path: '/settings/about', element: <AboutSettingsPage /> },
      { path: '/admin/roles', element: <Navigate to="/settings/roles" replace /> },
      { path: '/admin/roles/new', element: <Navigate to="/settings/roles/new" replace /> },
      { path: '/admin/users/roles', element: <Navigate to="/settings/user-roles" replace /> },
      {
        path: '/settings/tags',
        element: (
          <RequirePermission resource="endpoints" action="read">
            <TagsPage />
          </RequirePermission>
        ),
      },
      {
        path: '/settings/roles',
        element: (
          <RequirePermission resource="roles" action="read">
            <RolesPage />
          </RequirePermission>
        ),
      },
      {
        path: '/settings/roles/new',
        element: (
          <RequirePermission resource="roles" action="read">
            <RoleEditPage />
          </RequirePermission>
        ),
      },
      {
        path: '/settings/roles/:id/edit',
        element: (
          <RequirePermission resource="roles" action="read">
            <RoleEditPage />
          </RequirePermission>
        ),
      },
      {
        path: '/settings/user-roles',
        element: (
          <RequirePermission resource="roles" action="read">
            <UserRolesPage />
          </RequirePermission>
        ),
      },
      {
        path: '/agent-downloads',
        element: (
          <RequirePermission resource="endpoints" action="read">
            <AgentDownloadsPage />
          </RequirePermission>
        ),
      },
    ],
  },
]);
