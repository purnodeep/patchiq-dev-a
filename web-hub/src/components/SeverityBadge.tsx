import { SeverityText } from '@patchiq/ui';

export function SeverityBadge({ severity, large }: { severity: string; large?: boolean }) {
  return (
    <span style={{ fontSize: large ? '14px' : '12px', fontWeight: large ? 600 : 500 }}>
      <SeverityText severity={severity} />
    </span>
  );
}
