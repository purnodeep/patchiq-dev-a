import { createBrowserRouter, Navigate } from 'react-router';
import { AppLayout } from './layout/AppLayout';
import { LoginPage } from '../pages/login';
import { DashboardPage } from '../pages/dashboard/DashboardPage';
import { CatalogPage } from '../pages/catalog/CatalogPage';
import { CatalogDetailPage } from '../pages/catalog/CatalogDetailPage';
import { FeedsPage } from '../pages/feeds/FeedsPage';
import { FeedDetailPage } from '../pages/feeds/FeedDetailPage';
import { LicensesPage } from '../pages/licenses/LicensesPage';
import { LicenseDetailPage } from '../pages/licenses/LicenseDetailPage';
import { ClientsPage } from '../pages/clients/ClientsPage';
import { ClientDetailPage } from '../pages/clients/ClientDetailPage';
import { SettingsPage } from '../pages/settings/SettingsPage';
import { GeneralSettingsPage } from '../pages/settings/GeneralSettingsPage';
import { IAMSettingsPage } from '../pages/settings/IAMSettingsPage';
import { FeedConfigSettingsPage } from '../pages/settings/FeedConfigSettingsPage';
import { APIWebhookSettingsPage } from '../pages/settings/APIWebhookSettingsPage';
import { DeploymentsPage } from '../pages/deployments/DeploymentsPage';

export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  {
    element: <AppLayout />,
    children: [
      { path: '/', element: <DashboardPage /> },
      { path: '/catalog', element: <CatalogPage /> },
      { path: '/catalog/:id', element: <CatalogDetailPage /> },
      { path: '/feeds', element: <FeedsPage /> },
      { path: '/feeds/:id', element: <FeedDetailPage /> },
      { path: '/licenses', element: <LicensesPage /> },
      { path: '/licenses/:id', element: <LicenseDetailPage /> },
      { path: '/clients', element: <ClientsPage /> },
      { path: '/clients/:id', element: <ClientDetailPage /> },
      { path: '/deployments', element: <DeploymentsPage /> },
      {
        path: '/settings',
        element: <SettingsPage />,
        children: [
          { index: true, element: <Navigate to="/settings/general" replace /> },
          { path: 'general', element: <GeneralSettingsPage /> },
          { path: 'iam', element: <IAMSettingsPage /> },
          { path: 'feeds', element: <FeedConfigSettingsPage /> },
          { path: 'api', element: <APIWebhookSettingsPage /> },
        ],
      },
    ],
  },
]);
