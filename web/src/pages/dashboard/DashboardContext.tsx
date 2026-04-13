import { createContext, useContext } from 'react';
import type { DashboardSummary } from '@/api/hooks/useDashboard';

const DashboardDataContext = createContext<DashboardSummary | null>(null);

export function DashboardDataProvider({
  data,
  children,
}: {
  data: DashboardSummary;
  children: React.ReactNode;
}) {
  return <DashboardDataContext.Provider value={data}>{children}</DashboardDataContext.Provider>;
}

export function useDashboardData(): DashboardSummary {
  const ctx = useContext(DashboardDataContext);
  if (!ctx) throw new Error('useDashboardData must be used within DashboardDataProvider');
  return ctx;
}
