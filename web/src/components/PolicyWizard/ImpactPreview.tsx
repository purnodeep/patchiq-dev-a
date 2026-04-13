import { useMemo } from 'react';
import { useFormContext } from 'react-hook-form';
import { useValidateSelector } from '../../api/hooks/useTagSelector';
import type { Selector } from '../../types/targeting';
import type { PolicyWizardValues, PolicyWizardStepId } from './types';

// ── Dot grid — exact copy from DeploymentWizard ImpactPreview ─────────────────

function DotGrid({ total, ready, pending }: { total: number; ready: number; pending: number }) {
  const DOTS = 100;
  const readyDots = total > 0 ? Math.round((ready / total) * DOTS) : 0;
  const pendingDots = total > 0 ? Math.round((pending / total) * DOTS) : 0;

  return (
    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(10, 1fr)', gap: 2 }}>
      {Array.from({ length: DOTS }, (_, i) => {
        let color = 'var(--border)';
        if (i < readyDots) color = 'var(--signal-healthy)';
        else if (i < readyDots + pendingDots) color = 'var(--signal-warning)';
        return (
          <div
            key={i}
            style={{ width: 6, height: 6, borderRadius: '50%', background: color, flexShrink: 0 }}
          />
        );
      })}
    </div>
  );
}

// ── Severity bar — exact copy from DeploymentWizard ImpactPreview ─────────────

function SeverityBar({
  mode,
  minSeverity,
}: {
  mode: PolicyWizardValues['selection_mode'];
  minSeverity?: PolicyWizardValues['min_severity'];
}) {
  const { critical, high, other } = useMemo(() => {
    if (mode === 'all_available') return { critical: 20, high: 30, other: 50 };
    if (mode === 'by_severity') {
      if (minSeverity === 'critical') return { critical: 100, high: 0, other: 0 };
      if (minSeverity === 'high') return { critical: 40, high: 60, other: 0 };
      if (minSeverity === 'medium') return { critical: 25, high: 35, other: 40 };
    }
    return { critical: 35, high: 35, other: 30 };
  }, [mode, minSeverity]);

  const total = critical + high + other;
  if (total === 0)
    return (
      <div style={{ height: 8, borderRadius: 4, background: 'var(--border)', width: '100%' }} />
    );

  return (
    <div style={{ display: 'flex', height: 8, borderRadius: 4, overflow: 'hidden', width: '100%' }}>
      {critical > 0 && (
        <div
          style={{
            width: `${(critical / total) * 100}%`,
            background: 'var(--signal-critical)',
            flexShrink: 0,
          }}
        />
      )}
      {high > 0 && (
        <div
          style={{
            width: `${(high / total) * 100}%`,
            background: 'var(--signal-warning)',
            flexShrink: 0,
          }}
        />
      )}
      {other > 0 && (
        <div
          style={{ width: `${(other / total) * 100}%`, background: 'var(--border)', flexShrink: 0 }}
        />
      )}
    </div>
  );
}

// ── Week strip for cron schedule ───────────────────────────────────────────────

function WeekStrip({ dow }: { dow: number }) {
  const ABBR = ['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa'];
  const today = new Date().getDay();
  return (
    <div style={{ display: 'flex', gap: 3, marginTop: 6 }}>
      {ABBR.map((d, i) => (
        <div
          key={i}
          style={{
            flex: 1,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: 2,
          }}
        >
          <div
            style={{
              fontSize: 8,
              color: i === today ? 'var(--accent)' : 'var(--text-muted)',
              fontWeight: i === today ? 600 : 400,
            }}
          >
            {d}
          </div>
          <div
            style={{
              width: 14,
              height: 14,
              borderRadius: '50%',
              background: i === dow ? 'var(--accent)' : 'transparent',
              border: `1px solid ${i === dow ? 'var(--accent)' : 'var(--border)'}`,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            {i === dow && (
              <div
                style={{
                  width: 4,
                  height: 4,
                  borderRadius: '50%',
                  background: 'var(--btn-accent-text, #000)',
                }}
              />
            )}
          </div>
        </div>
      ))}
    </div>
  );
}

function parseCron(cron: string) {
  const parts = cron.trim().split(/\s+/);
  if (parts.length < 5) return null;
  const h = parseInt(parts[1], 10);
  const m = parseInt(parts[0], 10);
  if (isNaN(h) || isNaN(m)) return null;
  const dow = parts[4] === '*' ? -1 : parseInt(parts[4], 10);
  return { h, m, dow };
}

// ── Section label inside preview panel ───────────────────────────────────────

const PL: React.CSSProperties = {
  fontSize: 9,
  fontWeight: 600,
  textTransform: 'uppercase',
  letterSpacing: '0.06em',
  color: 'var(--text-muted)',
  fontFamily: 'var(--font-mono)',
  marginBottom: 6,
};

// ── Step previews ─────────────────────────────────────────────────────────────

function BasicsPreview({ mode, name }: { mode: string; name: string }) {
  const colors: Record<string, string> = {
    automatic: 'var(--signal-healthy)',
    manual: 'var(--accent)',
    advisory: 'var(--text-muted)',
  };
  const color = colors[mode] ?? 'var(--text-muted)';
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <div>
        <div style={PL}>Mode</div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <div style={{ width: 8, height: 8, borderRadius: '50%', background: color }} />
          <span style={{ fontSize: 12, fontWeight: 700, color, fontFamily: 'var(--font-mono)' }}>
            {mode}
          </span>
        </div>
      </div>
      {name && (
        <div>
          <div style={PL}>Name</div>
          <div
            style={{
              fontSize: 11,
              color: 'var(--text-primary)',
              fontWeight: 500,
              wordBreak: 'break-word',
            }}
          >
            {name}
          </div>
        </div>
      )}
    </div>
  );
}

function TargetsPreview({ endpointCount }: { endpointCount: number }) {
  const ready = Math.round(endpointCount * 0.8);
  const pending = Math.round(endpointCount * 0.15);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
      <div>
        <div
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 28,
            fontWeight: 700,
            color: 'var(--text-emphasis)',
            lineHeight: 1,
          }}
        >
          {endpointCount}
        </div>
        <div style={{ fontSize: 10, color: 'var(--text-muted)', marginTop: 2 }}>
          {endpointCount === 1 ? 'endpoint' : 'endpoints'} targeted
        </div>
      </div>
      {endpointCount > 0 && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
          <DotGrid total={endpointCount} ready={ready} pending={pending} />
          <div style={{ display: 'flex', gap: 8, fontSize: 9, color: 'var(--text-muted)' }}>
            {[
              ['var(--signal-healthy)', 'ready'],
              ['var(--signal-warning)', 'pending'],
              ['var(--border)', 'offline'],
            ].map(([bg, label]) => (
              <span key={label} style={{ display: 'flex', alignItems: 'center', gap: 3 }}>
                <span
                  style={{
                    width: 6,
                    height: 6,
                    borderRadius: '50%',
                    background: bg,
                    display: 'inline-block',
                  }}
                />
                {label}
              </span>
            ))}
          </div>
        </div>
      )}
      {endpointCount === 0 && (
        <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>Select groups above</div>
      )}
    </div>
  );
}

function PatchesPreview({
  mode,
  minSeverity,
  scheduleType,
  scheduleCron,
  cveIds,
}: {
  mode: PolicyWizardValues['selection_mode'];
  minSeverity?: PolicyWizardValues['min_severity'];
  scheduleType: string;
  scheduleCron: string;
  cveIds: string[];
}) {
  const patchLabel = useMemo(() => {
    if (mode === 'all_available') return 'All patches';
    if (mode === 'by_severity') return `Min ${minSeverity ?? '?'} severity`;
    if (mode === 'by_cve_list') return `${cveIds.length} CVE${cveIds.length !== 1 ? 's' : ''}`;
    if (mode === 'by_regex') return 'By regex';
    return mode;
  }, [mode, minSeverity, cveIds]);

  const parsed = scheduleCron ? parseCron(scheduleCron) : null;
  const timeStr = parsed
    ? `${String(parsed.h).padStart(2, '0')}:${String(parsed.m).padStart(2, '0')} UTC`
    : null;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <div>
        <div style={PL}>Patch Scope</div>
        <SeverityBar mode={mode} minSeverity={minSeverity} />
        <div style={{ fontSize: 10, color: 'var(--text-muted)', marginTop: 4 }}>{patchLabel}</div>
      </div>
      <div>
        <div style={PL}>Schedule</div>
        {scheduleType === 'manual' && (
          <div style={{ fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--text-muted)' }}>
            On demand
          </div>
        )}
        {scheduleType === 'maintenance_window' && (
          <div style={{ fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--text-muted)' }}>
            Next maint. window
          </div>
        )}
        {scheduleType === 'recurring' && parsed && (
          <div>
            <div
              style={{ fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--text-primary)' }}
            >
              {parsed.dow >= 0 ? `Weekly @ ${timeStr}` : `Daily @ ${timeStr}`}
            </div>
            {parsed.dow >= 0 && parsed.dow <= 6 && <WeekStrip dow={parsed.dow} />}
          </div>
        )}
        {scheduleType === 'recurring' && !parsed && (
          <div style={{ fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--text-muted)' }}>
            Enter cron expression
          </div>
        )}
      </div>
    </div>
  );
}

function ReviewPreview({
  endpointCount,
  mode,
  selectionMode,
  minSeverity,
  scheduleType,
  scheduleCron,
}: {
  endpointCount: number;
  mode: string;
  selectionMode: PolicyWizardValues['selection_mode'];
  minSeverity?: PolicyWizardValues['min_severity'];
  scheduleType: string;
  scheduleCron: string;
}) {
  const modeColors: Record<string, string> = {
    automatic: 'var(--signal-healthy)',
    manual: 'var(--accent)',
    advisory: 'var(--text-muted)',
  };
  const ready = Math.round(endpointCount * 0.8);
  const pending = Math.round(endpointCount * 0.15);
  const parsed = scheduleCron ? parseCron(scheduleCron) : null;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
      {/* Mode */}
      <div>
        <div style={PL}>Mode</div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 5 }}>
          <div
            style={{
              width: 7,
              height: 7,
              borderRadius: '50%',
              background: modeColors[mode] ?? 'var(--text-muted)',
            }}
          />
          <span
            style={{
              fontSize: 11,
              fontWeight: 600,
              color: modeColors[mode] ?? 'var(--text-muted)',
              fontFamily: 'var(--font-mono)',
            }}
          >
            {mode}
          </span>
        </div>
      </div>

      {/* Targets */}
      <div>
        <div style={PL}>Targets</div>
        <div
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 18,
            fontWeight: 700,
            color: 'var(--text-emphasis)',
            lineHeight: 1,
          }}
        >
          {endpointCount}
        </div>
        <div style={{ fontSize: 10, color: 'var(--text-muted)', marginTop: 2, marginBottom: 8 }}>
          endpoints
        </div>
        {endpointCount > 0 && <DotGrid total={endpointCount} ready={ready} pending={pending} />}
      </div>

      {/* Patches */}
      <div>
        <div style={PL}>Patches</div>
        <SeverityBar mode={selectionMode} minSeverity={minSeverity} />
      </div>

      {/* Strategy */}
      <div>
        <div style={PL}>Schedule</div>
        <div style={{ fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--text-muted)' }}>
          {scheduleType === 'manual'
            ? 'On demand'
            : scheduleType === 'maintenance_window'
              ? 'Next maint. window'
              : parsed
                ? parsed.dow >= 0
                  ? `Weekly @ ${String(parsed.h).padStart(2, '0')}:${String(parsed.m).padStart(2, '0')}`
                  : `Daily @ ${String(parsed.h).padStart(2, '0')}:${String(parsed.m).padStart(2, '0')}`
                : 'Recurring (TBD)'}
        </div>
      </div>
    </div>
  );
}

// ── Main export ────────────────────────────────────────────────────────────────

export function ImpactPreview({ currentStep }: { currentStep: PolicyWizardStepId }) {
  const { watch } = useFormContext<PolicyWizardValues>();
  const values = watch();
  const selector = (values.target_selector ?? null) as Selector | null;
  const validation = useValidateSelector(selector);
  const totalEndpoints = validation.data?.matched_count ?? 0;

  return (
    <div
      style={{
        position: 'sticky',
        top: 0,
        width: 150,
        flexShrink: 0,
        alignSelf: 'flex-start',
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: 8,
        padding: '14px 12px',
        display: 'flex',
        flexDirection: 'column',
        gap: 4,
      }}
    >
      {/* Header — identical to DeploymentWizard ImpactPreview header */}
      <div
        style={{
          fontSize: 9,
          fontWeight: 600,
          textTransform: 'uppercase',
          letterSpacing: '0.08em',
          color: 'var(--text-muted)',
          marginBottom: 10,
          paddingBottom: 8,
          borderBottom: '1px solid var(--border)',
          fontFamily: 'var(--font-mono)',
        }}
      >
        Impact Preview
      </div>

      {currentStep === 'basics' && <BasicsPreview mode={values.mode} name={values.name} />}

      {currentStep === 'targets' && <TargetsPreview endpointCount={totalEndpoints} />}

      {currentStep === 'patches' && (
        <PatchesPreview
          mode={values.selection_mode}
          minSeverity={values.min_severity}
          scheduleType={values.schedule_type}
          scheduleCron={values.schedule_cron}
          cveIds={values.cve_ids ?? []}
        />
      )}

      {currentStep === 'review' && (
        <ReviewPreview
          endpointCount={totalEndpoints}
          mode={values.mode}
          selectionMode={values.selection_mode}
          minSeverity={values.min_severity}
          scheduleType={values.schedule_type}
          scheduleCron={values.schedule_cron}
        />
      )}
    </div>
  );
}
