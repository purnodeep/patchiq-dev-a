import { AlertBanner } from '@patchiq/ui';

interface DashboardAlertBannerProps {
  overdueCount: number;
  onDismiss: () => void;
}

export function DashboardAlertBanner({ overdueCount, onDismiss }: DashboardAlertBannerProps) {
  if (overdueCount === 0) {
    return null;
  }

  return (
    <AlertBanner
      severity="warning"
      message={`${overdueCount} deployment${overdueCount === 1 ? '' : 's'} overdue SLA — immediate attention required`}
      onDismiss={onDismiss}
    />
  );
}
