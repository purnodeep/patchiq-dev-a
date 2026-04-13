import { useState } from 'react';
import { useCan } from '../../app/auth/AuthContext';
import { Eye, EyeOff, Loader2 } from 'lucide-react';
import { Skeleton } from '@patchiq/ui';
import { toast } from 'sonner';
import { useIAMSettings, useTestIAMConnection } from '../../api/hooks/useIAMSettings';
import type { RoleMapping } from '../../api/hooks/useIAMSettings';

const inputStyle: React.CSSProperties = {
  width: '100%',
  height: 36,
  padding: '0 10px',
  background: 'var(--bg-input, var(--bg-card))',
  border: '1px solid var(--border)',
  borderRadius: 6,
  fontSize: 13,
  color: 'var(--text-primary)',
  fontFamily: 'var(--font-mono)',
  outline: 'none',
  boxSizing: 'border-box',
  cursor: 'default',
};

const readOnlyBadgeStyle: React.CSSProperties = {
  background: 'color-mix(in srgb, var(--signal-info) 6%, transparent)',
  color: 'var(--signal-info)',
  border: '1px solid color-mix(in srgb, var(--signal-info) 15%, transparent)',
  borderRadius: 4,
  padding: '2px 7px',
  fontSize: 9,
  fontWeight: 600,
  letterSpacing: '0.05em',
  textTransform: 'uppercase',
  fontFamily: 'var(--font-mono)',
};

const labelStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: 6,
  fontSize: 10,
  fontWeight: 600,
  textTransform: 'uppercase',
  letterSpacing: '0.06em',
  color: 'var(--text-muted)',
  fontFamily: 'var(--font-mono)',
  marginBottom: 6,
};

function StatusPill({ status }: { status: string }) {
  const isConnected = status === 'connected';
  const isError = status === 'error';

  const pillStyle: React.CSSProperties = isConnected
    ? {
        background: 'color-mix(in srgb, var(--signal-healthy) 8%, transparent)',
        color: 'var(--signal-healthy)',
        border: '1px solid color-mix(in srgb, var(--signal-healthy) 20%, transparent)',
      }
    : isError
      ? {
          background: 'color-mix(in srgb, var(--signal-critical) 1%, transparent)',
          color: 'var(--signal-critical)',
          border: '1px solid color-mix(in srgb, var(--signal-critical) 1%, transparent)',
        }
      : {
          background: 'color-mix(in srgb, var(--text-muted) 8%, transparent)',
          color: 'var(--text-muted)',
          border: '1px solid color-mix(in srgb, var(--text-muted) 20%, transparent)',
        };

  return (
    <span
      style={{
        ...pillStyle,
        borderRadius: 20,
        padding: '4px 10px',
        fontSize: 11,
        fontWeight: 500,
        fontFamily: 'var(--font-sans)',
        whiteSpace: 'nowrap',
      }}
    >
      {isConnected ? 'Connected' : isError ? 'Error' : 'Not Configured'}
    </span>
  );
}

function InfoIcon() {
  return (
    <svg
      width="14"
      height="14"
      viewBox="0 0 24 24"
      fill="none"
      stroke="var(--signal-info)"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      style={{ flexShrink: 0 }}
    >
      <circle cx="12" cy="12" r="10" />
      <line x1="12" y1="16" x2="12" y2="12" />
      <line x1="12" y1="8" x2="12.01" y2="8" />
    </svg>
  );
}

function RoleMappingRow({ mapping }: { mapping: RoleMapping }) {
  return (
    <tr>
      <td
        style={{
          padding: '8px 12px',
          fontSize: 12,
          fontFamily: 'var(--font-mono)',
          color: 'var(--text-secondary)',
          borderBottom: '1px solid var(--border)',
        }}
      >
        {mapping.external_role}
      </td>
      <td
        style={{
          padding: '8px 8px',
          fontSize: 12,
          color: 'var(--text-faint)',
          borderBottom: '1px solid var(--border)',
        }}
      >
        &rarr;
      </td>
      <td
        style={{
          padding: '8px 12px',
          fontSize: 12,
          color: 'var(--text-primary)',
          fontFamily: 'var(--font-sans)',
          borderBottom: '1px solid var(--border)',
        }}
      >
        {mapping.role_name}
      </td>
      <td
        style={{
          padding: '8px 12px',
          borderBottom: '1px solid var(--border)',
        }}
      >
        <span
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: 4,
            fontSize: 10,
            fontWeight: 600,
            color: 'var(--signal-healthy)',
            textTransform: 'uppercase',
            letterSpacing: '0.05em',
            fontFamily: 'var(--font-sans)',
          }}
        >
          <div
            style={{
              width: 5,
              height: 5,
              borderRadius: '50%',
              background: 'var(--signal-healthy)',
            }}
          />
          Active
        </span>
      </td>
    </tr>
  );
}

export function IdentitySettingsPage() {
  const can = useCan();
  const [revealClientId, setRevealClientId] = useState(false);
  const { data: settings, isLoading, error } = useIAMSettings(revealClientId);
  const testConnection = useTestIAMConnection();
  const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null);

  function handleTestConnection() {
    testConnection.mutate(undefined, {
      onSuccess: (result) => {
        if (result.success) {
          setTestResult({
            success: true,
            message: `OIDC discovery validated in ${result.latency_ms}ms`,
          });
          setTimeout(() => setTestResult(null), 4000);
        } else {
          toast.error(`Connection failed: ${result.error ?? 'Unknown error'}`);
        }
      },
      onError: (err) => {
        toast.error(`Failed to test connection: ${err.message}`);
      },
    });
  }

  if (isLoading) {
    return (
      <div
        style={{
          padding: '28px 40px 80px',
          maxWidth: 680,
          display: 'flex',
          flexDirection: 'column',
          gap: 20,
        }}
      >
        {/* Section header skeleton */}
        <div>
          <Skeleton className="h-6 w-40" />
          <Skeleton className="h-4 w-72 mt-2" />
        </div>
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-12 w-full" />
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-24 w-full" />
      </div>
    );
  }

  if (error) {
    return (
      <div
        style={{
          padding: '28px 40px 80px',
          maxWidth: 680,
          display: 'flex',
          flexDirection: 'column',
          gap: 20,
        }}
      >
        <div
          style={{ paddingBottom: 16, borderBottom: '1px solid var(--border)', marginBottom: 4 }}
        >
          <h2
            style={{
              fontSize: 18,
              fontWeight: 600,
              color: 'var(--text-emphasis)',
              letterSpacing: '-0.02em',
              margin: 0,
            }}
          >
            Identity & Access
          </h2>
          <p
            style={{
              fontSize: 12,
              color: 'var(--text-muted)',
              margin: '4px 0 0',
            }}
          >
            Single sign-on via Zitadel OIDC. Connection testing and role mapping.
          </p>
        </div>
        <p
          style={{ fontSize: 13, color: 'var(--signal-critical)', fontFamily: 'var(--font-sans)' }}
        >
          Failed to load IAM settings. Please try again.
        </p>
      </div>
    );
  }

  return (
    <div
      style={{
        padding: '28px 40px 80px',
        maxWidth: 680,
        display: 'flex',
        flexDirection: 'column',
        gap: 20,
      }}
    >
      {/* Section header */}
      <div style={{ paddingBottom: 16, borderBottom: '1px solid var(--border)', marginBottom: 4 }}>
        <h2
          style={{
            fontSize: 18,
            fontWeight: 600,
            color: 'var(--text-emphasis)',
            letterSpacing: '-0.02em',
            margin: 0,
          }}
        >
          Identity & Access
        </h2>
        <p
          style={{
            fontSize: 12,
            color: 'var(--text-muted)',
            margin: '4px 0 0',
          }}
        >
          Single sign-on via Zitadel OIDC. Connection testing and role mapping.
        </p>
      </div>

      {settings ? (
        <>
          {/* Provider card */}
          <div
            style={{
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              borderRadius: 10,
              padding: 16,
              display: 'flex',
              alignItems: 'center',
              gap: 12,
            }}
          >
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                width: 40,
                height: 40,
                borderRadius: 8,
                background:
                  'linear-gradient(135deg, var(--signal-warning), var(--signal-critical))',
                fontSize: 16,
                fontWeight: 800,
                color: 'var(--text-on-color, #fff)',
                flexShrink: 0,
              }}
            >
              Z
            </div>
            <div style={{ flex: 1, minWidth: 0 }}>
              <div
                style={{
                  fontSize: 13,
                  fontWeight: 600,
                  color: 'var(--text-primary)',
                  fontFamily: 'var(--font-sans)',
                }}
              >
                Zitadel
              </div>
              <div
                style={{
                  fontSize: 11,
                  color: 'var(--text-muted)',
                  fontFamily: 'var(--font-mono)',
                  marginTop: 2,
                }}
              >
                OIDC + PKCE &middot; SSO Provider
              </div>
            </div>
            <StatusPill status={settings.connection_status} />
          </div>

          {/* Info banner */}
          <div
            style={{
              display: 'flex',
              alignItems: 'flex-start',
              gap: 10,
              background: 'color-mix(in srgb, var(--signal-info) 6%, transparent)',
              border: '1px solid color-mix(in srgb, var(--signal-info) 15%, transparent)',
              borderRadius: 8,
              padding: '12px 14px',
              fontSize: 12,
              color: 'var(--text-secondary)',
              fontFamily: 'var(--font-sans)',
              lineHeight: 1.5,
            }}
          >
            <InfoIcon />
            <span>
              Identity settings are managed through the Zitadel admin console. Fields below are
              read-only. Use Test Connection to verify.
            </span>
          </div>

          {/* SSO URL */}
          <div>
            <div style={labelStyle}>
              <span>SSO URL</span>
              <span style={readOnlyBadgeStyle}>READ-ONLY</span>
            </div>
            <input value={settings.sso_url} readOnly style={inputStyle} />
          </div>

          {/* Client ID */}
          <div>
            <div style={labelStyle}>
              <span>Client ID</span>
              <span style={readOnlyBadgeStyle}>READ-ONLY</span>
            </div>
            <div style={{ display: 'flex', gap: 8 }}>
              <input
                value={settings.client_id}
                readOnly
                style={{
                  ...inputStyle,
                  flex: 1,
                  letterSpacing: revealClientId ? 'normal' : '0.15em',
                }}
              />
              <button
                type="button"
                onClick={() => setRevealClientId(!revealClientId)}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 5,
                  height: 36,
                  padding: '0 12px',
                  background: 'transparent',
                  border: '1px solid var(--border)',
                  borderRadius: 6,
                  fontSize: 12,
                  color: 'var(--text-secondary)',
                  cursor: 'pointer',
                  fontFamily: 'var(--font-sans)',
                  whiteSpace: 'nowrap',
                }}
              >
                {revealClientId ? (
                  <>
                    <EyeOff style={{ width: 13, height: 13 }} />
                    Hide
                  </>
                ) : (
                  <>
                    <Eye style={{ width: 13, height: 13 }} />
                    Reveal
                  </>
                )}
              </button>
            </div>
          </div>

          {/* Organization ID */}
          {settings.zitadel_org_id && (
            <div>
              <div style={labelStyle}>
                <span>Organization ID</span>
                <span style={readOnlyBadgeStyle}>READ-ONLY</span>
              </div>
              <input value={settings.zitadel_org_id} readOnly style={inputStyle} />
            </div>
          )}

          {/* User Sync */}
          <div>
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                marginBottom: 6,
              }}
            >
              <div style={labelStyle}>
                <span>User Sync</span>
              </div>
            </div>
            <div
              style={{
                background: 'var(--bg-card)',
                border: '1px solid var(--border)',
                borderRadius: 8,
                padding: '12px 14px',
                display: 'flex',
                flexDirection: 'column',
                gap: 10,
              }}
            >
              <div
                style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}
              >
                <div>
                  <div
                    style={{
                      fontSize: 12,
                      fontWeight: 500,
                      color: 'var(--text-primary)',
                      fontFamily: 'var(--font-sans)',
                    }}
                  >
                    Automatic user provisioning
                  </div>
                  <div
                    style={{
                      fontSize: 11,
                      color: 'var(--text-faint)',
                      fontFamily: 'var(--font-sans)',
                      marginTop: 2,
                    }}
                  >
                    Sync users from Zitadel every {settings.user_sync_interval_minutes} minutes
                  </div>
                </div>
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 6,
                    fontSize: 11,
                    fontFamily: 'var(--font-mono)',
                    color: settings.user_sync_enabled
                      ? 'var(--signal-healthy)'
                      : 'var(--text-faint)',
                  }}
                >
                  <div
                    style={{
                      width: 7,
                      height: 7,
                      borderRadius: '50%',
                      background: settings.user_sync_enabled
                        ? 'var(--signal-healthy)'
                        : 'var(--border-strong)',
                    }}
                  />
                  {settings.user_sync_enabled ? 'Enabled' : 'Disabled'}
                </div>
              </div>
              {settings.user_sync_enabled && (
                <div
                  style={{
                    display: 'flex',
                    gap: 16,
                    fontSize: 11,
                    color: 'var(--text-muted)',
                    fontFamily: 'var(--font-sans)',
                    borderTop: '1px solid var(--border)',
                    paddingTop: 10,
                  }}
                >
                  <span>
                    Interval:{' '}
                    <strong
                      style={{ fontFamily: 'var(--font-mono)', color: 'var(--text-secondary)' }}
                    >
                      {settings.user_sync_interval_minutes}m
                    </strong>
                  </span>
                </div>
              )}
            </div>
          </div>

          {/* Test Connection */}
          <div>
            <button
              type="button"
              onClick={handleTestConnection}
              disabled={testConnection.isPending || !can('settings', 'write')}
              title={!can('settings', 'write') ? "You don't have permission" : undefined}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 6,
                height: 34,
                padding: '0 14px',
                background: 'transparent',
                border: '1px solid var(--border)',
                borderRadius: 6,
                fontSize: 12,
                fontWeight: 500,
                color: 'var(--text-secondary)',
                cursor: testConnection.isPending ? 'not-allowed' : 'pointer',
                fontFamily: 'var(--font-sans)',
                opacity: testConnection.isPending ? 0.6 : 1,
              }}
            >
              {testConnection.isPending ? (
                <>
                  <Loader2 style={{ width: 12, height: 12 }} className="animate-spin" />
                  Testing...
                </>
              ) : (
                'Test Connection'
              )}
            </button>
          </div>

          {/* Last tested timestamp */}
          {settings.last_tested_at && !testResult && (
            <div
              style={{
                fontSize: 11,
                color: 'var(--text-faint)',
                fontFamily: 'var(--font-sans)',
                marginTop: -12,
              }}
            >
              Last tested:{' '}
              {new Date(settings.last_tested_at).toLocaleString(undefined, {
                month: 'short',
                day: 'numeric',
                hour: '2-digit',
                minute: '2-digit',
              })}
            </div>
          )}

          {/* Test result banner */}
          {testResult && (
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                background: 'color-mix(in srgb, var(--signal-healthy) 6%, transparent)',
                border: '1px solid color-mix(in srgb, var(--signal-healthy) 15%, transparent)',
                borderRadius: 8,
                padding: '10px 14px',
                fontSize: 12,
                color: 'var(--signal-healthy)',
                fontFamily: 'var(--font-sans)',
              }}
            >
              <svg
                width="14"
                height="14"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
                style={{ flexShrink: 0 }}
              >
                <path d="M20 6L9 17l-5-5" />
              </svg>
              {testResult.message}
            </div>
          )}

          {/* Divider */}
          <div style={{ height: 1, background: 'var(--border)' }} />

          {/* Role Mappings */}
          <div>
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                marginBottom: 12,
              }}
            >
              <span
                style={{
                  fontSize: 13,
                  fontWeight: 600,
                  color: 'var(--text-emphasis)',
                  fontFamily: 'var(--font-sans)',
                }}
              >
                Role Mappings
              </span>
              <span
                style={{
                  background: 'color-mix(in srgb, var(--signal-info) 6%, transparent)',
                  color: 'var(--signal-info)',
                  border: '1px solid color-mix(in srgb, var(--signal-info) 15%, transparent)',
                  borderRadius: 4,
                  padding: '2px 7px',
                  fontSize: 9,
                  fontWeight: 600,
                  letterSpacing: '0.05em',
                  textTransform: 'uppercase',
                  fontFamily: 'var(--font-mono)',
                }}
              >
                MANAGED
              </span>
            </div>
            {settings.role_mappings.length > 0 ? (
              <div
                style={{
                  borderRadius: 6,
                  border: '1px solid var(--border)',
                  overflow: 'hidden',
                }}
              >
                <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                  <thead style={{ background: 'var(--bg-inset)' }}>
                    <tr>
                      {['Zitadel Role', '', 'PatchIQ Role', 'Status'].map((h, i) => (
                        <th
                          key={i}
                          style={{
                            padding: '8px 12px',
                            fontSize: 10,
                            fontWeight: 600,
                            textTransform: 'uppercase',
                            letterSpacing: '0.06em',
                            color: 'var(--text-muted)',
                            textAlign: 'left',
                            fontFamily: 'var(--font-sans)',
                            borderBottom: '1px solid var(--border)',
                          }}
                        >
                          {h}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {settings.role_mappings.map((m) => (
                      <RoleMappingRow key={m.external_role} mapping={m} />
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <p
                style={{
                  fontSize: 12,
                  color: 'var(--text-faint)',
                  fontFamily: 'var(--font-sans)',
                }}
              >
                No role mappings configured.
              </p>
            )}
          </div>
        </>
      ) : (
        <p style={{ fontSize: 13, color: 'var(--text-muted)', fontFamily: 'var(--font-sans)' }}>
          IAM settings are not configured. Contact your administrator.
        </p>
      )}
    </div>
  );
}
