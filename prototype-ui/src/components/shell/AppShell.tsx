import { Outlet, useLocation } from 'react-router';
import { AnimatePresence, motion } from 'framer-motion';
import { Sidebar } from './Sidebar';
import { TopBar } from './TopBar';

export function AppShell() {
  const location = useLocation();

  return (
    <div className="relative h-screen w-screen overflow-hidden bg-background">
      {/* Ambient blobs — these create material for the glass blur */}
      <div
        className="pointer-events-none absolute -top-[100px] right-[10%] h-[500px] w-[500px] rounded-full"
        style={{
          background: 'radial-gradient(circle, var(--color-blob-1) 0%, transparent 70%)',
          filter: 'blur(80px)',
          animation: 'float-blob-1 15s ease-in-out infinite',
        }}
      />
      <div
        className="pointer-events-none absolute -bottom-[120px] left-[5%] h-[450px] w-[450px] rounded-full"
        style={{
          background: 'radial-gradient(circle, var(--color-blob-2) 0%, transparent 70%)',
          filter: 'blur(80px)',
          animation: 'float-blob-2 18s ease-in-out infinite',
        }}
      />
      <div
        className="pointer-events-none absolute left-[35%] top-[40%] h-[350px] w-[350px] rounded-full"
        style={{
          background: 'radial-gradient(circle, var(--color-blob-3) 0%, transparent 70%)',
          filter: 'blur(70px)',
          animation: 'float-blob-3 20s ease-in-out infinite',
        }}
      />

      {/* App layout */}
      <div className="relative z-10 flex h-full gap-4 p-4">
        <Sidebar />
        <main className="flex min-w-0 flex-1 flex-col gap-3">
          <TopBar />
          <div className="flex-1 overflow-y-auto overflow-x-hidden rounded-2xl pr-1">
            <AnimatePresence mode="wait">
              <motion.div
                key={location.pathname}
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                transition={{ duration: 0.15 }}
                className="flex-1"
              >
                <Outlet />
              </motion.div>
            </AnimatePresence>
          </div>
        </main>
      </div>
    </div>
  );
}
