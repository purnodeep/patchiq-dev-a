import { Star, Calendar, CheckCircle2, XCircle } from 'lucide-react';
import { useLicenseStatus } from '../../api/hooks/useLicense';

const featureLabels: Record<string, string> = {
  endpoint_management: 'Endpoint Management',
  patch_deployment: 'Patch Deployment',
  visual_workflow_builder: 'Workflow Engine',
  compliance_frameworks: 'Compliance Frameworks',
  multi_wave_deployments: 'Multi-Wave Deployments',
  api_access: 'API Access',
  custom_rbac: 'RBAC',
  sso_oidc: 'SSO Integration',
  ai_assistant: 'AI Assistant',
  multi_site: 'Multi-Site Deployment',
  ha_dr: 'HA/DR',
  third_party_patching: '3rd Party Patching',
  vulnerability_integration: 'Vulnerability Integration',
};

const TIER_COLORS: Record<string, string> = {
  free: 'var(--text-muted)',
  standard: 'var(--accent)',
  enterprise: 'var(--signal-warning)',
};

export function LicenseSettingsPage() {
  const { data: license, isLoading, error } = useLicenseStatus();

  const usagePercent =
    license && license.endpoint_usage.limit > 0
      ? Math.min((license.endpoint_usage.current / license.endpoint_usage.limit) * 100, 100)
      : 0;

  const slotsRemaining =
    license && license.endpoint_usage.limit > 0
      ? license.endpoint_usage.limit - license.endpoint_usage.current
      : null;

  const tierColor = license
    ? (TIER_COLORS[license.tier.toLowerCase()] ?? 'var(--text-muted)')
    : 'var(--text-muted)';

  if (isLoading) {
    return (
      <div style={{ padding: '28px 40px 80px', maxWidth: 680 }}>
        <p style={{ fontSize: 13, color: 'var(--text-muted)', fontFamily: 'var(--font-sans)' }}>
          Loading...
        </p>
      </div>
    );
  }

  if (error) {
    return (
      <div style={{ padding: '28px 40px 80px', maxWidth: 680 }}>
        <p
          style={{ fontSize: 13, color: 'var(--signal-critical)', fontFamily: 'var(--font-sans)' }}
        >
          Failed to load license status.
        </p>
      </div>
    );
  }

  if (!license) return null;

  const featureEntries = Object.entries(featureLabels);

  return (
    <div style={{ padding: '28px 40px 80px', maxWidth: 680 }}>
      {/* Section header */}
      <div style={{ paddingBottom: 16, borderBottom: '1px solid var(--border)', marginBottom: 24 }}>
        <h2
          style={{
            fontSize: 18,
            fontWeight: 600,
            color: 'var(--text-emphasis)',
            letterSpacing: '-0.02em',
            margin: 0,
          }}
        >
          License
        </h2>
        <p
          style={{
            fontSize: 12,
            color: 'var(--text-muted)',
            marginTop: 4,
            margin: '4px 0 0',
          }}
        >
          Current tier, endpoint capacity, and feature entitlements.
        </p>
      </div>

      {/* License card — compound display */}
      <div
        style={{
          background: 'var(--bg-card)',
          border: '1px solid var(--border)',
          borderRadius: 8,
          display: 'flex',
          overflow: 'hidden',
        }}
      >
        {/* Tier badge area */}
        <div
          style={{
            minWidth: 100,
            padding: 16,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            gap: 4,
            background:
              'linear-gradient(135deg, color-mix(in srgb, var(--accent) 12%, transparent), color-mix(in srgb, var(--accent) 4%, transparent))',
            borderRight: '1px solid color-mix(in srgb, var(--accent) 20%, transparent)',
            borderRadius: 8,
          }}
        >
          <Star style={{ width: 20, height: 20, color: tierColor }} />
          <span
            style={{
              fontSize: 13,
              fontWeight: 700,
              color: tierColor,
              textTransform: 'uppercase',
              letterSpacing: '0.04em',
              fontFamily: 'var(--font-sans)',
            }}
          >
            {license.tier}
          </span>
          <span
            style={{
              fontSize: 10,
              fontFamily: 'var(--font-mono)',
              color: 'var(--text-faint)',
              textTransform: 'uppercase',
              letterSpacing: '0.1em',
            }}
          >
            TIER
          </span>
        </div>

        {/* Details area */}
        <div
          style={{
            flex: 1,
            padding: '16px 20px',
            display: 'flex',
            flexDirection: 'column',
            justifyContent: 'center',
            gap: 12,
          }}
        >
          {/* Usage bar */}
          <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            <div
              style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline' }}
            >
              <span
                style={{
                  fontSize: 12,
                  color: 'var(--text-secondary)',
                  fontFamily: 'var(--font-sans)',
                }}
              >
                Endpoint Usage
              </span>
              <span
                style={{
                  fontSize: 12,
                  fontFamily: 'var(--font-mono)',
                  color: 'var(--text-primary)',
                  fontWeight: 600,
                }}
              >
                {license.endpoint_usage.current}
                {' / '}
                {license.endpoint_usage.limit === 0 ? '\u221e' : license.endpoint_usage.limit}
              </span>
            </div>
            {license.endpoint_usage.limit > 0 && (
              <>
                <div
                  style={{
                    height: 6,
                    background: 'var(--bg-inset)',
                    borderRadius: 3,
                    overflow: 'hidden',
                  }}
                >
                  <div
                    style={{
                      height: '100%',
                      width: `${usagePercent}%`,
                      background:
                        usagePercent > 85
                          ? 'var(--signal-critical)'
                          : usagePercent > 60
                            ? 'var(--signal-warning)'
                            : 'var(--signal-healthy)',
                      borderRadius: 3,
                      transition: 'width 0.3s',
                    }}
                  />
                </div>
                {slotsRemaining !== null && (
                  <p
                    style={{
                      fontSize: 11,
                      color: 'var(--text-faint)',
                      fontFamily: 'var(--font-sans)',
                      margin: 0,
                    }}
                  >
                    {slotsRemaining} slots remaining
                  </p>
                )}
              </>
            )}
          </div>

          {/* Expiry row */}
          {license.days_remaining > 0 ? (
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <Calendar style={{ width: 13, height: 13, color: 'var(--text-faint)' }} />
              <span
                style={{
                  fontSize: 12,
                  fontFamily: 'var(--font-mono)',
                  color: 'var(--text-primary)',
                }}
              >
                {new Date(license.expires_at).toLocaleDateString('en-US', {
                  month: 'short',
                  day: 'numeric',
                  year: 'numeric',
                })}
              </span>
              <span
                style={{
                  fontSize: 11,
                  color: 'var(--signal-healthy)',
                  fontFamily: 'var(--font-sans)',
                }}
              >
                {license.days_remaining} days remaining
              </span>
            </div>
          ) : (
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <Calendar style={{ width: 13, height: 13, color: 'var(--text-faint)' }} />
              <span
                style={{
                  fontSize: 12,
                  color: 'var(--text-muted)',
                  fontFamily: 'var(--font-sans)',
                }}
              >
                No expiration — community license
              </span>
            </div>
          )}
        </div>
      </div>

      {/* Feature Entitlements */}
      <div style={{ marginTop: 24 }}>
        <h3
          style={{
            fontSize: 14,
            fontWeight: 600,
            color: 'var(--text-emphasis)',
            fontFamily: 'var(--font-sans)',
            margin: '0 0 12px',
          }}
        >
          Feature Entitlements
        </h3>

        <div
          style={{
            display: 'grid',
            gridTemplateColumns: '1fr 1fr',
            border: '1px solid var(--border)',
            borderRadius: 8,
            overflow: 'hidden',
          }}
        >
          {featureEntries.map(([key, label], idx) => {
            const enabled = Boolean(license.features[key]);
            const isOdd = idx % 2 === 0; // left column (0-indexed even = left)
            const isLastRow = idx >= featureEntries.length - 2;

            return (
              <div
                key={key}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                  padding: '10px 14px',
                  borderBottom: isLastRow ? 'none' : '1px solid var(--border)',
                  borderRight: isOdd ? '1px solid var(--border)' : 'none',
                  background: 'var(--bg-card)',
                }}
              >
                {enabled ? (
                  <div
                    style={{
                      width: 16,
                      height: 16,
                      borderRadius: 4,
                      background: 'color-mix(in srgb, var(--signal-healthy) 10%, transparent)',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      flexShrink: 0,
                    }}
                  >
                    <CheckCircle2
                      style={{ width: 12, height: 12, color: 'var(--signal-healthy)' }}
                    />
                  </div>
                ) : (
                  <div
                    style={{
                      width: 16,
                      height: 16,
                      borderRadius: 4,
                      background: 'color-mix(in srgb, var(--signal-critical) 1%, transparent)',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      flexShrink: 0,
                    }}
                  >
                    <XCircle style={{ width: 12, height: 12, color: 'var(--text-faint)' }} />
                  </div>
                )}
                <span
                  style={{
                    fontSize: 12,
                    color: enabled ? 'var(--text-primary)' : 'var(--text-faint)',
                    fontFamily: 'var(--font-sans)',
                  }}
                >
                  {label}
                </span>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
