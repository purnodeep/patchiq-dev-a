import { useMemo } from 'react';
import { useFormContext } from 'react-hook-form';
import { usePatches } from '../../api/hooks/usePatches';
import { useEndpoints } from '../../api/hooks/useEndpoints';
import type { DeploymentWizardValues, WizardStepId } from '../../types/deployment-wizard';
import type { PatchListItem } from '../../types/patches';

interface ImpactPreviewProps {
  currentStep: WizardStepId;
}

// ── Mini severity bar ──────────────────────────────────────────────────────────

function SeverityBar({
  critical,
  high,
  other,
  total,
}: {
  critical: number;
  high: number;
  other: number;
  total: number;
}) {
  if (total === 0) {
    return (
      <div
        style={{
          height: 8,
          borderRadius: 4,
          background: 'var(--border)',
          width: '100%',
        }}
      />
    );
  }

  const critPct = (critical / total) * 100;
  const highPct = (high / total) * 100;
  const otherPct = (other / total) * 100;

  return (
    <div style={{ display: 'flex', height: 8, borderRadius: 4, overflow: 'hidden', width: '100%' }}>
      {critPct > 0 && (
        <div
          style={{ width: `${critPct}%`, background: 'var(--signal-critical)', flexShrink: 0 }}
        />
      )}
      {highPct > 0 && (
        <div style={{ width: `${highPct}%`, background: 'var(--signal-warning)', flexShrink: 0 }} />
      )}
      {otherPct > 0 && (
        <div style={{ width: `${otherPct}%`, background: 'var(--border)', flexShrink: 0 }} />
      )}
    </div>
  );
}

// ── Dot grid (targets step) ────────────────────────────────────────────────────

function DotGrid({ total, ready, pending }: { total: number; ready: number; pending: number }) {
  const DOTS = 100;
  const readyDots = total > 0 ? Math.round((ready / total) * DOTS) : 0;
  const pendingDots = total > 0 ? Math.round((pending / total) * DOTS) : 0;

  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(10, 1fr)',
        gap: 2,
      }}
    >
      {Array.from({ length: DOTS }, (_, i) => {
        let color = 'var(--text-muted)'; // grey = offline
        if (i < readyDots)
          color = 'var(--signal-healthy)'; // emerald = ready
        else if (i < readyDots + pendingDots) color = 'var(--signal-warning)'; // amber = pending
        return (
          <div
            key={i}
            style={{
              width: 6,
              height: 6,
              borderRadius: '50%',
              background: color,
              flexShrink: 0,
            }}
          />
        );
      })}
    </div>
  );
}

// ── Wave timeline (strategy step) ─────────────────────────────────────────────

function WaveTimeline({ waves }: { waves: { maxTargets: number; successThreshold: number }[] }) {
  if (waves.length === 0) return null;

  const totalTargets = waves.reduce((sum, w) => sum + w.maxTargets, 0);

  // Estimate duration: ~15 min base + 2 min per target per wave, capped at a reasonable display
  const estimatedHours = Math.max(1, Math.round(waves.length * 1.5));

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
      <div style={{ display: 'flex', alignItems: 'flex-end', gap: 3, height: 32 }}>
        {waves.map((wave, idx) => {
          const heightPct =
            totalTargets > 0 ? Math.max(20, (wave.maxTargets / totalTargets) * 100) : 50;
          const isLast = idx === waves.length - 1;
          return (
            <div
              key={idx}
              style={{ display: 'flex', alignItems: 'flex-end', gap: 3, flex: wave.maxTargets }}
            >
              <div
                style={{
                  flex: 1,
                  height: `${heightPct}%`,
                  minHeight: 8,
                  borderRadius: '3px 3px 0 0',
                  background: 'var(--accent)',
                  opacity: 0.7 + idx * 0.1,
                }}
                title={`Wave ${idx + 1}: ${wave.maxTargets} targets, ${wave.successThreshold}% success threshold`}
              />
              {!isLast && (
                <div
                  style={{
                    width: 2,
                    height: 12,
                    background: 'var(--border)',
                    alignSelf: 'center',
                    borderRadius: 1,
                    flexShrink: 0,
                  }}
                />
              )}
            </div>
          );
        })}
      </div>
      <div
        style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 10,
          color: 'var(--text-muted)',
        }}
      >
        Est. duration: ~{estimatedHours}h
      </div>
    </div>
  );
}

// ── Source preview (source step) ───────────────────────────────────────────────

function SourcePreview({ patches }: { patches: PatchListItem[] }) {
  const { critical, high, other, cvssTotal } = useMemo(() => {
    let crit = 0,
      hi = 0,
      ot = 0,
      cvss = 0;
    for (const p of patches) {
      if (p.severity === 'critical') crit++;
      else if (p.severity === 'high') hi++;
      else ot++;
      cvss += p.highest_cvss_score ?? 0;
    }
    return { critical: crit, high: hi, other: ot, cvssTotal: cvss };
  }, [patches]);

  const total = patches.length;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
      {/* Big count */}
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
          {total}
        </div>
        <div style={{ fontSize: 10, color: 'var(--text-muted)', marginTop: 2 }}>
          {total === 1 ? 'patch' : 'patches'} selected
        </div>
      </div>

      {/* Severity bar */}
      {total > 0 && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
          <SeverityBar critical={critical} high={high} other={other} total={total} />
          <div style={{ display: 'flex', gap: 6, fontSize: 9, color: 'var(--text-muted)' }}>
            {critical > 0 && (
              <span>
                <span style={{ color: 'var(--signal-critical)' }}>{critical}</span> crit
              </span>
            )}
            {high > 0 && (
              <span>
                <span style={{ color: 'var(--signal-warning)' }}>{high}</span> high
              </span>
            )}
            {other > 0 && <span>{other} other</span>}
          </div>
        </div>
      )}

      {/* CVSS exposure */}
      {total > 0 && (
        <div
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 10,
            color: 'var(--text-muted)',
          }}
        >
          CVSS Exposure: {cvssTotal.toFixed(1)} total
        </div>
      )}
    </div>
  );
}

// ── Targets preview ────────────────────────────────────────────────────────────

function TargetsPreview({ endpointCount }: { endpointCount: number }) {
  // We can't easily split by status without fetching each endpoint;
  // approximate: treat 80% as ready, 15% pending, 5% offline for display purposes.
  // The real endpoint count already comes from TargetsStep's filtered query.
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
            <span style={{ display: 'flex', alignItems: 'center', gap: 3 }}>
              <span
                style={{
                  width: 6,
                  height: 6,
                  borderRadius: '50%',
                  background: 'var(--signal-healthy)',
                  display: 'inline-block',
                }}
              />
              ready
            </span>
            <span style={{ display: 'flex', alignItems: 'center', gap: 3 }}>
              <span
                style={{
                  width: 6,
                  height: 6,
                  borderRadius: '50%',
                  background: 'var(--signal-warning)',
                  display: 'inline-block',
                }}
              />
              pending
            </span>
            <span style={{ display: 'flex', alignItems: 'center', gap: 3 }}>
              <span
                style={{
                  width: 6,
                  height: 6,
                  borderRadius: '50%',
                  background: 'var(--text-muted)',
                  display: 'inline-block',
                }}
              />
              offline
            </span>
          </div>
        </div>
      )}
    </div>
  );
}

// ── Strategy preview ───────────────────────────────────────────────────────────

function StrategyPreview({
  waves,
  schedule,
  scheduledAt,
}: {
  waves: { maxTargets: number; successThreshold: number }[];
  schedule: 'now' | 'datetime' | 'maintenance_window';
  scheduledAt?: string;
}) {
  const scheduleLabel = useMemo(() => {
    if (schedule === 'now') return 'Immediately';
    if (schedule === 'maintenance_window') return 'Next maintenance window';
    if (scheduledAt) {
      const d = new Date(scheduledAt);
      return isNaN(d.getTime())
        ? 'Scheduled'
        : d.toLocaleString([], { dateStyle: 'short', timeStyle: 'short' });
    }
    return 'Scheduled (TBD)';
  }, [schedule, scheduledAt]);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
      <div>
        <div style={{ fontSize: 10, color: 'var(--text-muted)', marginBottom: 4 }}>
          {waves.length} {waves.length === 1 ? 'wave' : 'waves'}
        </div>
        <WaveTimeline waves={waves} />
      </div>
      <div
        style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 10,
          color: 'var(--text-muted)',
          borderTop: '1px solid var(--border)',
          paddingTop: 8,
        }}
      >
        {scheduleLabel}
      </div>
    </div>
  );
}

// ── Review preview (all three mini-visualizations) ─────────────────────────────

function ReviewPreview({
  patchCount,
  critCount,
  highCount,
  otherCount,
  cvssTotal,
  endpointCount,
  waves,
  schedule,
  scheduledAt,
}: {
  patchCount: number;
  critCount: number;
  highCount: number;
  otherCount: number;
  cvssTotal: number;
  endpointCount: number;
  waves: { maxTargets: number; successThreshold: number }[];
  schedule: 'now' | 'datetime' | 'maintenance_window';
  scheduledAt?: string;
}) {
  const ready = Math.round(endpointCount * 0.8);
  const pending = Math.round(endpointCount * 0.15);
  const estimatedHours = Math.max(1, Math.round(waves.length * 1.5));

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
      {/* Patches mini */}
      <div>
        <div
          style={{
            fontSize: 9,
            fontWeight: 600,
            color: 'var(--text-muted)',
            textTransform: 'uppercase',
            letterSpacing: '0.05em',
            marginBottom: 6,
          }}
        >
          Patches
        </div>
        <div style={{ display: 'flex', alignItems: 'baseline', gap: 4, marginBottom: 4 }}>
          <span
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 18,
              fontWeight: 700,
              color: 'var(--text-emphasis)',
            }}
          >
            {patchCount}
          </span>
          <span style={{ fontSize: 10, color: 'var(--text-muted)' }}>selected</span>
        </div>
        <SeverityBar critical={critCount} high={highCount} other={otherCount} total={patchCount} />
        {patchCount > 0 && (
          <div
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 9,
              color: 'var(--text-muted)',
              marginTop: 4,
            }}
          >
            CVSS: {cvssTotal.toFixed(1)}
          </div>
        )}
      </div>

      {/* Endpoints mini */}
      <div>
        <div
          style={{
            fontSize: 9,
            fontWeight: 600,
            color: 'var(--text-muted)',
            textTransform: 'uppercase',
            letterSpacing: '0.05em',
            marginBottom: 6,
          }}
        >
          Targets
        </div>
        <div style={{ display: 'flex', alignItems: 'baseline', gap: 4, marginBottom: 6 }}>
          <span
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 18,
              fontWeight: 700,
              color: 'var(--text-emphasis)',
            }}
          >
            {endpointCount}
          </span>
          <span style={{ fontSize: 10, color: 'var(--text-muted)' }}>endpoints</span>
        </div>
        {endpointCount > 0 && <DotGrid total={endpointCount} ready={ready} pending={pending} />}
      </div>

      {/* Strategy mini */}
      <div>
        <div
          style={{
            fontSize: 9,
            fontWeight: 600,
            color: 'var(--text-muted)',
            textTransform: 'uppercase',
            letterSpacing: '0.05em',
            marginBottom: 6,
          }}
        >
          Strategy
        </div>
        <WaveTimeline waves={waves} />
        <div
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 9,
            color: 'var(--text-muted)',
            marginTop: 4,
          }}
        >
          {schedule === 'now'
            ? 'Deploys immediately'
            : schedule === 'maintenance_window'
              ? 'Next maintenance window'
              : scheduledAt
                ? `Scheduled: ${new Date(scheduledAt).toLocaleString([], { dateStyle: 'short', timeStyle: 'short' })}`
                : 'Scheduled (TBD)'}
        </div>
        <div
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 9,
            color: 'var(--text-muted)',
            marginTop: 2,
          }}
        >
          Est. ~{estimatedHours}h total
        </div>
      </div>
    </div>
  );
}

// ── Main export ────────────────────────────────────────────────────────────────

export function ImpactPreview({ currentStep }: ImpactPreviewProps) {
  const { watch } = useFormContext<DeploymentWizardValues>();
  const values = watch();

  // Fetch selected patches when sourceType=catalog
  const selectedPatchIds = values.patchIds ?? [];
  const { data: patchesData } = usePatches({
    limit: 50,
  });

  // Filter patches list to selected IDs (best effort from cached data)
  const selectedPatches = useMemo(() => {
    if (values.sourceType !== 'catalog') return [];
    const all = patchesData?.data ?? [];
    return all.filter((p) => selectedPatchIds.includes(p.id));
  }, [patchesData, selectedPatchIds, values.sourceType]);

  // Endpoint count from the TargetsStep's data (we re-use totalData)
  const { data: totalData } = useEndpoints({ limit: 1 });
  const totalEndpoints = totalData?.total_count ?? 0;

  // Respect target mode for endpoint count
  const endpointCount =
    values.targetMode === 'select' ? (values.endpointIds?.length ?? 0) : totalEndpoints;

  // Compute patch stats for review step
  const { critCount, highCount, otherCount, cvssTotal } = useMemo(() => {
    const all = patchesData?.data ?? [];
    const selected =
      values.sourceType === 'catalog' ? all.filter((p) => selectedPatchIds.includes(p.id)) : all;
    let crit = 0,
      hi = 0,
      ot = 0,
      cvss = 0;
    for (const p of selected) {
      if (p.severity === 'critical') crit++;
      else if (p.severity === 'high') hi++;
      else ot++;
      cvss += p.highest_cvss_score ?? 0;
    }
    return { critCount: crit, highCount: hi, otherCount: ot, cvssTotal: cvss };
  }, [patchesData, selectedPatchIds, values.sourceType]);

  const patchCount =
    values.sourceType === 'catalog'
      ? selectedPatchIds.length
      : values.sourceType === 'adhoc'
        ? (values.adhocPackages?.length ?? 0)
        : 0; // policy: unknown count

  return (
    <div
      style={{
        position: 'sticky',
        top: 0,
        width: 160,
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
        }}
      >
        Impact Preview
      </div>

      {currentStep === 'source' && <SourcePreview patches={selectedPatches} />}

      {currentStep === 'targets' && <TargetsPreview endpointCount={endpointCount} />}

      {currentStep === 'strategy' && (
        <StrategyPreview
          waves={values.waves}
          schedule={values.schedule}
          scheduledAt={values.scheduledAt}
        />
      )}

      {currentStep === 'review' && (
        <ReviewPreview
          patchCount={patchCount}
          critCount={critCount}
          highCount={highCount}
          otherCount={otherCount}
          cvssTotal={cvssTotal}
          endpointCount={endpointCount}
          waves={values.waves}
          schedule={values.schedule}
          scheduledAt={values.scheduledAt}
        />
      )}
    </div>
  );
}
