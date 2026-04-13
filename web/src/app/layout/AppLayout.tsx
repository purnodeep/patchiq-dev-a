import { useState, useEffect } from 'react';
import { Outlet } from 'react-router';
import { ThemeProvider, RouteErrorBoundary } from '@patchiq/ui';
import { Toaster } from 'sonner';
import { AuthProvider } from '../auth/AuthContext';
import { AppSidebar } from './AppSidebar';
import { TopBar } from './TopBar';
import { CommandPalette } from '@/pages/dashboard/CommandPalette';

export const AppLayout = () => {
  const [cmdPaletteOpen, setCmdPaletteOpen] = useState(false);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        setCmdPaletteOpen((prev) => !prev);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

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
            <TopBar onOpenCommandPalette={() => setCmdPaletteOpen(true)} />
            <main style={{ flex: 1, overflowY: 'auto', background: 'var(--bg-page)' }}>
              <RouteErrorBoundary>
                <Outlet />
              </RouteErrorBoundary>
            </main>
          </div>
        </div>
        <CommandPalette open={cmdPaletteOpen} onOpenChange={setCmdPaletteOpen} />
        <Toaster richColors position="bottom-right" hotkey={[]} />
      </AuthProvider>
    </ThemeProvider>
  );
};
