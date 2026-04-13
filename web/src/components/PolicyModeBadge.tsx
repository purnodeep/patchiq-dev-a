type SelectionMode = 'all_available' | 'by_severity' | 'by_cve_list' | 'by_regex';

const modeConfig: Record<SelectionMode, { label: string; bg: string; color: string }> = {
  all_available: {
    label: 'All Patches',
    bg: 'color-mix(in srgb, var(--accent) 10%, transparent)',
    color: 'var(--accent)',
  },
  by_severity: {
    label: 'By Severity',
    bg: 'color-mix(in srgb, var(--signal-warning) 10%, transparent)',
    color: 'var(--signal-warning)',
  },
  by_cve_list: {
    label: 'By CVE',
    bg: 'color-mix(in srgb, var(--signal-critical) 1%, transparent)',
    color: 'var(--signal-critical)',
  },
  by_regex: {
    label: 'By Regex',
    bg: 'color-mix(in srgb, var(--text-muted) 10%, transparent)',
    color: 'var(--text-secondary)',
  },
};

interface PolicyModeBadgeProps {
  mode: SelectionMode;
}

export const PolicyModeBadge = ({ mode }: PolicyModeBadgeProps) => {
  const c = modeConfig[mode];
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        borderRadius: 4,
        padding: '2px 8px',
        fontSize: 11,
        fontWeight: 600,
        fontFamily: 'var(--font-mono)',
        background: c.bg,
        color: c.color,
      }}
    >
      {c.label}
    </span>
  );
};

/* ── Policy Mode Label (automatic / manual / advisory) ──── */

type PolicyMode = 'automatic' | 'manual' | 'advisory';

const policyModeConfig: Record<PolicyMode, { label: string; bg: string; color: string }> = {
  automatic: {
    label: 'Automatic',
    bg: 'color-mix(in srgb, var(--signal-healthy) 10%, transparent)',
    color: 'var(--signal-healthy)',
  },
  manual: {
    label: 'Manual',
    bg: 'color-mix(in srgb, var(--text-muted) 10%, transparent)',
    color: 'var(--text-secondary)',
  },
  advisory: {
    label: 'Advisory',
    bg: 'color-mix(in srgb, var(--text-muted) 10%, transparent)',
    color: 'var(--text-secondary)',
  },
};

interface PolicyModeLabelProps {
  mode: PolicyMode;
  className?: string;
}

export const PolicyModeLabel = ({ mode }: PolicyModeLabelProps) => {
  const c = policyModeConfig[mode];
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        borderRadius: 4,
        padding: '2px 8px',
        fontSize: 11,
        fontWeight: 600,
        fontFamily: 'var(--font-mono)',
        background: c.bg,
        color: c.color,
      }}
    >
      {c.label}
    </span>
  );
};
