// eslint-disable-next-line @typescript-eslint/no-explicit-any -- PolicyDetail schema not yet in generated types
type PolicyDetail = any;

interface PatchScopeTabProps {
  policy: PolicyDetail;
}

const CARD: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: 8,
  boxShadow: 'var(--shadow-sm)',
  padding: '16px 20px',
};

const LABEL: React.CSSProperties = {
  fontFamily: 'var(--font-mono)',
  fontSize: 9,
  fontWeight: 600,
  textTransform: 'uppercase',
  letterSpacing: '0.07em',
  color: 'var(--text-muted)',
  marginBottom: 14,
};

const selectionModeLabels: Record<string, string> = {
  all_available: 'All Available Patches',
  by_severity: 'By Severity',
  by_cve_list: 'By CVE List',
  by_regex: 'By Package Regex',
};

// Pipeline node for the visual selection flow
function PipelineStage({
  step,
  label,
  value,
  color,
  isLast = false,
}: {
  step: number;
  label: string;
  value: string;
  color: string;
  isLast?: boolean;
}) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', flex: 1, minWidth: 0 }}>
      <div
        style={{
          flex: 1,
          background: 'var(--bg-inset)',
          border: `1px solid color-mix(in srgb, ${color} 13%, transparent)`,
          borderRadius: 6,
          padding: '12px 14px',
          position: 'relative',
        }}
      >
        {/* Step badge */}
        <div
          style={{
            position: 'absolute',
            top: -8,
            left: 12,
            width: 16,
            height: 16,
            borderRadius: '50%',
            background: color,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: 8,
            fontWeight: 700,
            color: 'var(--btn-accent-text, #000)',
            fontFamily: 'var(--font-mono)',
          }}
        >
          {step}
        </div>
        <div
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 9,
            fontWeight: 600,
            textTransform: 'uppercase',
            letterSpacing: '0.06em',
            color,
            marginBottom: 6,
            marginTop: 4,
          }}
        >
          {label}
        </div>
        <div
          style={{
            fontSize: 11,
            color: 'var(--text-primary)',
            fontWeight: 500,
            lineHeight: 1.4,
          }}
        >
          {value}
        </div>
      </div>
      {!isLast && (
        <div
          style={{
            padding: '0 8px',
            fontSize: 14,
            color: 'var(--text-faint)',
            flexShrink: 0,
          }}
        >
          →
        </div>
      )}
    </div>
  );
}

export const PatchScopeTab = ({ policy }: PatchScopeTabProps) => {
  const mode = policy.selection_mode ?? 'all_available';
  const modeLabel = selectionModeLabels[mode] ?? mode;

  const criteria: { key: string; value: string }[] = [];
  if (policy.selection_mode) criteria.push({ key: 'Selection Mode', value: modeLabel });
  if (policy.min_severity)
    criteria.push({ key: 'Min Severity', value: `≥ ${policy.min_severity}` });
  if (policy.package_regex) criteria.push({ key: 'Package Regex', value: policy.package_regex });
  if (policy.cve_ids?.length > 0)
    criteria.push({ key: 'CVE IDs', value: policy.cve_ids.join(', ') });
  if (policy.exclude_packages?.length > 0)
    criteria.push({ key: 'Excluded Packages', value: policy.exclude_packages.join(', ') });
  if (criteria.length === 0) criteria.push({ key: 'Mode', value: modeLabel });

  const pipelineStages = [
    { step: 1, label: 'Source', value: 'All Available Patches', color: 'var(--accent)' },
    { step: 2, label: 'Mode', value: modeLabel, color: 'var(--text-secondary)' },
    ...(policy.min_severity
      ? [
          {
            step: 3,
            label: 'Severity Filter',
            value: `≥ ${policy.min_severity}`,
            color: 'var(--signal-warning)',
          },
        ]
      : []),
    ...(policy.package_regex
      ? [
          {
            step: policy.min_severity ? 4 : 3,
            label: 'Regex Filter',
            value: policy.package_regex,
            color: 'var(--signal-warning)',
          },
        ]
      : []),
    {
      step: criteria.length + 1,
      label: 'Result',
      value: 'Matched Patches',
      color: 'var(--accent)',
    },
  ];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
      {/* Visual pipeline hero */}
      <div style={CARD}>
        <div style={LABEL}>Selection Pipeline</div>
        <div
          style={{
            display: 'flex',
            alignItems: 'stretch',
            gap: 0,
            overflowX: 'auto',
            paddingTop: 10,
          }}
        >
          {pipelineStages.map((stage, i) => (
            <PipelineStage
              key={stage.label}
              step={stage.step}
              label={stage.label}
              value={stage.value}
              color={stage.color}
              isLast={i === pipelineStages.length - 1}
            />
          ))}
        </div>
      </div>

      {/* Details grid */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'minmax(260px, 1fr) minmax(260px, 2fr)',
          gap: 14,
        }}
      >
        {/* Filter criteria card */}
        <div style={CARD}>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              marginBottom: 14,
            }}
          >
            <div style={LABEL}>Filter Criteria</div>
            <span
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 9,
                fontWeight: 600,
                color: 'var(--accent)',
                background: 'color-mix(in srgb, var(--accent) 8%, transparent)',
                border: '1px solid color-mix(in srgb, var(--accent) 20%, transparent)',
                borderRadius: 3,
                padding: '2px 7px',
              }}
            >
              Live
            </span>
          </div>
          {criteria.map((c, i) => (
            <div
              key={c.key}
              style={{
                display: 'flex',
                alignItems: 'flex-start',
                gap: 12,
                padding: '9px 0',
                borderBottom: i < criteria.length - 1 ? '1px solid var(--border)' : undefined,
              }}
            >
              <div
                style={{
                  fontSize: 10,
                  color: 'var(--text-muted)',
                  width: 110,
                  flexShrink: 0,
                  paddingTop: 1,
                }}
              >
                {c.key}
              </div>
              <div
                style={{
                  fontSize: 12,
                  color: 'var(--text-primary)',
                  fontWeight: 500,
                  fontFamily:
                    c.key === 'Package Regex' || c.key === 'CVE IDs'
                      ? 'var(--font-mono)'
                      : 'inherit',
                  wordBreak: 'break-all',
                }}
              >
                {c.value}
              </div>
            </div>
          ))}
        </div>

        {/* Matched patches result */}
        <div style={CARD}>
          <div style={LABEL}>Matched Patches</div>
          <div
            style={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              minHeight: 140,
              border: '1px dashed var(--border)',
              borderRadius: 6,
              background: 'color-mix(in srgb, white 1%, transparent)',
              gap: 8,
            }}
          >
            <div
              style={{
                width: 32,
                height: 32,
                borderRadius: '50%',
                background: 'var(--bg-inset)',
                border: '1px solid var(--border)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: 16,
                color: 'var(--text-faint)',
              }}
            >
              ◎
            </div>
            <div style={{ fontSize: 13, color: 'var(--text-secondary)', fontWeight: 500 }}>
              No evaluation data
            </div>
            <div
              style={{
                fontSize: 11,
                color: 'var(--text-muted)',
                textAlign: 'center',
                maxWidth: 280,
              }}
            >
              Run an evaluation to see which patches this policy selects.
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
