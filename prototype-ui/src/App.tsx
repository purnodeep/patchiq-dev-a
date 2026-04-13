import { BrowserRouter, Routes, Route, Navigate } from 'react-router';
import { ThemeProvider } from '@/components/shell/ThemeProvider';
import { AppShell } from '@/components/shell/AppShell';
import Dashboard from '@/pages/pm/Dashboard';
import Endpoints from '@/pages/pm/Endpoints';
import Patches from '@/pages/pm/Patches';
import CVEs from '@/pages/pm/CVEs';
import Policies from '@/pages/pm/Policies';
import Deployments from '@/pages/pm/Deployments';
import Workflows from '@/pages/pm/Workflows';
import Compliance from '@/pages/pm/Compliance';
import Audit from '@/pages/pm/Audit';
import Settings from '@/pages/pm/Settings';
import Roles from '@/pages/pm/Roles';
import Notifications from '@/pages/pm/Notifications';

export default function App() {
  return (
    <ThemeProvider>
      <BrowserRouter>
        <Routes>
          <Route element={<AppShell />}>
            <Route index element={<Navigate to="/pm/dashboard" replace />} />
            <Route path="/pm/dashboard" element={<Dashboard />} />
            <Route path="/pm/endpoints" element={<Endpoints />} />
            <Route path="/pm/patches" element={<Patches />} />
            <Route path="/pm/cves" element={<CVEs />} />
            <Route path="/pm/policies" element={<Policies />} />
            <Route path="/pm/deployments" element={<Deployments />} />
            <Route path="/pm/workflows" element={<Workflows />} />
            <Route path="/pm/compliance" element={<Compliance />} />
            <Route path="/pm/audit" element={<Audit />} />
            <Route path="/pm/settings" element={<Settings />} />
            <Route path="/pm/roles" element={<Roles />} />
            <Route path="/pm/notifications" element={<Notifications />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </ThemeProvider>
  );
}
