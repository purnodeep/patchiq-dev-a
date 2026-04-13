import { Outlet } from 'react-router';
import { ThemeProvider, RouteErrorBoundary } from '@patchiq/ui';
import { Toaster } from 'sonner';
import { AuthProvider } from '../auth/AuthContext';
import { AppSidebar } from './AppSidebar';
import { TopBar } from './TopBar';

export const AppLayout = () => {
  return (
    <ThemeProvider>
      <AuthProvider>
        <div style={{ display: 'flex', height: '100vh', overflow: 'hidden' }}>
          <AppSidebar />
          <div
            style={{
              flex: 1,
              display: 'flex',
              flexDirection: 'column',
              overflow: 'hidden',
              minWidth: 0,
            }}
          >
            <TopBar />
            <main style={{ flex: 1, overflowY: 'auto', background: 'var(--bg-page)' }}>
              <RouteErrorBoundary>
                <Outlet />
              </RouteErrorBoundary>
            </main>
          </div>
        </div>
        <Toaster richColors position="bottom-right" hotkey={[]} />
      </AuthProvider>
    </ThemeProvider>
  );
};
