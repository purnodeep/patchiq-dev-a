import { MonoTag } from '@patchiq/ui';

export function SourceBadge({ source }: { source: string | null | undefined }) {
  if (!source) return <span style={{ fontSize: '12px', color: 'var(--text-faint)' }}>--</span>;
  const s = source.toUpperCase();
  let label = source;
  if (s.includes('NVD')) label = 'NVD';
  else if (s.includes('KEV')) label = 'KEV';
  else if (s.includes('MSRC')) label = 'MSRC';
  else if (s.includes('USN')) label = 'USN';
  return <MonoTag>{label}</MonoTag>;
}
