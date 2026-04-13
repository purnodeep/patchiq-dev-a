import { Outlet } from 'react-router';
import { ThemeProvider, RouteErrorBoundary } from '@patchiq/ui';
import { TopBar } from './TopBar';
import { IconRail } from './IconRail';

export const AppLayout = () => (
  <ThemeProvider>
    <div style={{ background: 'var(--bg-page)', minHeight: '100vh' }}>
      <TopBar />
      <IconRail />
      <main
        style={{
          position: 'fixed',
          top: '48px',
          left: '48px',
          right: 0,
          bottom: 0,
          overflowY: 'auto',
          padding: '20px 24px',
        }}
      >
        <RouteErrorBoundary>
          <Outlet />
        </RouteErrorBoundary>
      </main>
    </div>
  </ThemeProvider>
);
