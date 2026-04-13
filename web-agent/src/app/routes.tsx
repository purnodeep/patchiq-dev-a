import { createBrowserRouter, Navigate } from 'react-router';
import { AppLayout } from './layout/AppLayout';
import { StatusPage } from '../pages/status/StatusPage';
import { PendingPatchesPage } from '../pages/pending/PendingPatchesPage';
import { HistoryPage } from '../pages/history/HistoryPage';
import { LogsPage } from '../pages/logs/LogsPage';
import { SettingsPage } from '../pages/settings/SettingsPage';
import { HardwarePage } from '../pages/hardware/HardwarePage';
import { SoftwarePage } from '../pages/software/SoftwarePage';
import { ServicesPage } from '../pages/services/ServicesPage';

export const router = createBrowserRouter([
  {
    element: <AppLayout />,
    children: [
      { path: '/', element: <StatusPage /> },
      { path: '/pending', element: <PendingPatchesPage /> },
      { path: '/patches', element: <Navigate to="/pending" replace /> },
      { path: '/hardware', element: <HardwarePage /> },
      { path: '/software', element: <SoftwarePage /> },
      { path: '/services', element: <ServicesPage /> },
      { path: '/history', element: <HistoryPage /> },
      { path: '/logs', element: <LogsPage /> },
      { path: '/settings', element: <SettingsPage /> },
    ],
  },
]);
