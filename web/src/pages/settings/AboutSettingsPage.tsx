import { ShieldCheck, Activity, Database, ExternalLink } from 'lucide-react';
import { Skeleton } from '@patchiq/ui';
import { useServerHealth } from '../../api/hooks/useServerHealth';

function StatusDot({ ok }: { ok: boolean }) {
  return (
    <div
      style={{
        width: 8,
        height: 8,
        borderRadius: '50%',
        background: ok ? 'var(--signal-healthy)' : 'var(--signal-critical)',
        boxShadow: ok
          ? '0 0 0 3px color-mix(in srgb, var(--signal-healthy) 12%, transparent)'
          : '0 0 0 3px color-mix(in srgb, var(--signal-critical) 1%, transparent)',
        flexShrink: 0,
      }}
    />
  );
}

function ServiceRow({ label, status, detail }: { label: string; status: boolean; detail: string }) {
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 10,
        padding: '10px 0',
        borderBottom: '1px solid var(--border)',
      }}
    >
      <StatusDot ok={status} />
      <div style={{ flex: 1, minWidth: 0 }}>
        <div
          style={{
            fontSize: 12,
            fontWeight: 500,
            color: 'var(--text-primary)',
            fontFamily: 'var(--font-sans)',
          }}
        >
          {label}
        </div>
      </div>
      <span
        style={{
          fontSize: 11,
          color: 'var(--text-muted)',
          fontFamily: 'var(--font-mono)',
        }}
      >
        {detail}
      </span>
    </div>
  );
}

function LinkRow({ label, href }: { label: string; href: string }) {
  return (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        padding: '10px 0',
        borderBottom: '1px solid var(--border)',
        textDecoration: 'none',
        cursor: 'pointer',
      }}
    >
      <span
        style={{
          fontSize: 12,
          color: 'var(--text-primary)',
          fontFamily: 'var(--font-sans)',
        }}
      >
        {label}
      </span>
      <ExternalLink
        style={{ width: 12, height: 12, color: 'var(--text-faint)', strokeWidth: 1.5 }}
      />
    </a>
  );
}

export function AboutSettingsPage() {
  const { data: health, isLoading, error } = useServerHealth();
  const isHealthy = !error && health?.status === 'ok';

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
        <div>
          <Skeleton className="h-6 w-32" />
          <Skeleton className="h-4 w-64 mt-2" />
        </div>
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-32 w-full" />
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
          About
        </h2>
        <p
          style={{
            fontSize: 12,
            color: 'var(--text-muted)',
            margin: '4px 0 0',
          }}
        >
          Platform information, system health, and resources.
        </p>
      </div>

      {/* Platform Card */}
      <div
        style={{
          background: 'var(--bg-card)',
          border: '1px solid var(--border)',
          borderRadius: 10,
          padding: 20,
          display: 'flex',
          alignItems: 'center',
          gap: 14,
        }}
      >
        <div
          style={{
            width: 48,
            height: 48,
            borderRadius: 10,
            background:
              'linear-gradient(135deg, var(--brand), color-mix(in srgb, var(--accent) 60%, transparent))',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            flexShrink: 0,
          }}
        >
          <ShieldCheck
            style={{ width: 24, height: 24, color: 'var(--text-on-color, #fff)', strokeWidth: 1.5 }}
          />
        </div>
        <div style={{ flex: 1 }}>
          <div
            style={{
              fontSize: 16,
              fontWeight: 700,
              color: 'var(--text-emphasis)',
              fontFamily: 'var(--font-sans)',
              letterSpacing: '-0.02em',
            }}
          >
            PatchIQ Patch Manager
          </div>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              marginTop: 4,
            }}
          >
            <span
              style={{
                fontSize: 11,
                fontFamily: 'var(--font-mono)',
                color: 'var(--text-muted)',
              }}
            >
              {health?.version ?? 'unknown'}
            </span>
            <span
              style={{
                fontSize: 9,
                fontWeight: 600,
                textTransform: 'uppercase',
                letterSpacing: '0.06em',
                padding: '2px 6px',
                borderRadius: 3,
                background: 'color-mix(in srgb, var(--signal-warning) 10%, transparent)',
                color: 'var(--signal-warning)',
                border: '1px solid color-mix(in srgb, var(--signal-warning) 20%, transparent)',
                fontFamily: 'var(--font-mono)',
              }}
            >
              POC
            </span>
          </div>
        </div>
      </div>

      {/* System Health */}
      <div>
        <span
          style={{
            fontSize: 13,
            fontWeight: 600,
            color: 'var(--text-emphasis)',
            fontFamily: 'var(--font-sans)',
            display: 'block',
            marginBottom: 8,
          }}
        >
          System Health
        </span>
        <div
          style={{
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            padding: '4px 14px',
          }}
        >
          <ServiceRow
            label="Application Server"
            status={isHealthy}
            detail={isHealthy ? 'Healthy' : error ? 'Unreachable' : 'Unknown'}
          />
          <ServiceRow label="Uptime" status={true} detail={health?.uptime ?? '—'} />
          <ServiceRow
            label="Database"
            status={isHealthy}
            detail={isHealthy ? 'Connected' : 'Unknown'}
          />
        </div>
      </div>

      {/* Divider */}
      <div style={{ height: 1, background: 'var(--border)' }} />

      {/* Architecture */}
      <div>
        <span
          style={{
            fontSize: 13,
            fontWeight: 600,
            color: 'var(--text-emphasis)',
            fontFamily: 'var(--font-sans)',
            display: 'block',
            marginBottom: 8,
          }}
        >
          Architecture
        </span>
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: '1fr 1fr',
            gap: 10,
          }}
        >
          {[
            { label: 'Backend', value: 'Go + chi/v5', icon: Activity },
            { label: 'Database', value: 'PostgreSQL 16', icon: Database },
            { label: 'Frontend', value: 'React 19 + Vite', icon: Activity },
            { label: 'Auth', value: 'Zitadel OIDC', icon: ShieldCheck },
          ].map((item) => (
            <div
              key={item.label}
              style={{
                background: 'var(--bg-card)',
                border: '1px solid var(--border)',
                borderRadius: 8,
                padding: '10px 14px',
                display: 'flex',
                alignItems: 'center',
                gap: 8,
              }}
            >
              <item.icon
                style={{ width: 14, height: 14, color: 'var(--text-faint)', strokeWidth: 1.5 }}
              />
              <div>
                <div
                  style={{
                    fontSize: 10,
                    color: 'var(--text-faint)',
                    fontFamily: 'var(--font-sans)',
                    textTransform: 'uppercase',
                    letterSpacing: '0.04em',
                    fontWeight: 600,
                  }}
                >
                  {item.label}
                </div>
                <div
                  style={{
                    fontSize: 12,
                    color: 'var(--text-primary)',
                    fontFamily: 'var(--font-mono)',
                    marginTop: 1,
                  }}
                >
                  {item.value}
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Divider */}
      <div style={{ height: 1, background: 'var(--border)' }} />

      {/* Resources */}
      <div>
        <span
          style={{
            fontSize: 13,
            fontWeight: 600,
            color: 'var(--text-emphasis)',
            fontFamily: 'var(--font-sans)',
            display: 'block',
            marginBottom: 8,
          }}
        >
          Resources
        </span>
        <div
          style={{
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            padding: '4px 14px',
          }}
        >
          <LinkRow label="API Documentation" href="/api/v1/docs" />
          <LinkRow label="Release Notes" href="https://github.com/herambskanda/patchiq/releases" />
          <LinkRow label="Report an Issue" href="https://github.com/herambskanda/patchiq/issues" />
        </div>
      </div>

      {/* Footer */}
      <div
        style={{
          fontSize: 11,
          color: 'var(--text-faint)',
          fontFamily: 'var(--font-sans)',
          textAlign: 'center',
          paddingTop: 8,
        }}
      >
        &copy; {new Date().getFullYear()} SkenzerIQ. All rights reserved.
      </div>
    </div>
  );
}
