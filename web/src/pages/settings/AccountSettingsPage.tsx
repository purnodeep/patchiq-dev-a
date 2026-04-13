import { Shield, Clock } from 'lucide-react';
import { useAuth } from '../../app/auth/AuthContext';

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

function DetailRow({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div
      style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        padding: '10px 0',
        borderBottom: '1px solid var(--border)',
      }}
    >
      <span
        style={{
          fontSize: 12,
          color: 'var(--text-muted)',
          fontFamily: 'var(--font-sans)',
        }}
      >
        {label}
      </span>
      <span
        style={{
          fontSize: 12,
          color: 'var(--text-primary)',
          fontFamily: mono ? 'var(--font-mono)' : 'var(--font-sans)',
          fontWeight: 500,
          maxWidth: '60%',
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          whiteSpace: 'nowrap',
          textAlign: 'right',
        }}
      >
        {value}
      </span>
    </div>
  );
}

export function AccountSettingsPage() {
  const { user } = useAuth();

  const initials = (user.name ?? user.email ?? 'U')
    .split(' ')
    .map((w) => w[0])
    .join('')
    .toUpperCase()
    .slice(0, 2);

  const displayName = user.name ?? 'Not set';
  const displayEmail = user.email ?? 'Not set';
  const displayUsername =
    ((user as unknown as Record<string, unknown>).preferred_username as string | undefined) ??
    user.email ??
    '—';
  const displayRole =
    Array.isArray(user.roles) && user.roles.length > 0
      ? user.roles.join(', ')
      : (((user as unknown as Record<string, unknown>).role as string | undefined) ?? 'User');

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
          My Account
        </h2>
        <p
          style={{
            fontSize: 12,
            color: 'var(--text-muted)',
            margin: '4px 0 0',
          }}
        >
          Your profile, role, and session information.
        </p>
      </div>

      {/* Profile Card */}
      <div
        style={{
          background: 'var(--bg-card)',
          border: '1px solid var(--border)',
          borderRadius: 10,
          padding: 20,
          display: 'flex',
          alignItems: 'center',
          gap: 16,
        }}
      >
        <div
          style={{
            width: 52,
            height: 52,
            borderRadius: '50%',
            background: 'var(--avatar-bg)',
            border: '2px solid var(--avatar-border)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: 16,
            fontWeight: 700,
            color: 'var(--avatar-text)',
            letterSpacing: '0.02em',
            flexShrink: 0,
          }}
        >
          {initials}
        </div>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div
            style={{
              fontSize: 15,
              fontWeight: 600,
              color: 'var(--text-emphasis)',
              fontFamily: 'var(--font-sans)',
            }}
          >
            {displayName}
          </div>
          <div
            style={{
              fontSize: 12,
              color: 'var(--text-muted)',
              fontFamily: 'var(--font-sans)',
              marginTop: 2,
            }}
          >
            {displayEmail}
          </div>
          <div
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 4,
              marginTop: 6,
              padding: '3px 8px',
              background: 'color-mix(in srgb, var(--accent) 8%, transparent)',
              border: '1px solid color-mix(in srgb, var(--accent) 20%, transparent)',
              borderRadius: 4,
              fontSize: 10,
              fontWeight: 600,
              color: 'var(--accent)',
              textTransform: 'uppercase',
              letterSpacing: '0.04em',
              fontFamily: 'var(--font-sans)',
            }}
          >
            <Shield style={{ width: 10, height: 10 }} />
            {displayRole}
          </div>
        </div>
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
          Profile settings including name, email, and password are managed through your identity
          provider (Zitadel). Role assignments are managed by your administrator.
        </span>
      </div>

      {/* Account Details */}
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
          Account Details
        </span>
        <div
          style={{
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            padding: '4px 14px',
          }}
        >
          <DetailRow label="Name" value={displayName} />
          <DetailRow label="Email" value={displayEmail} />
          <DetailRow label="Username" value={displayUsername} />
          <DetailRow label="User ID" value={user.user_id} mono />
          {user.tenant_id && <DetailRow label="Tenant ID" value={user.tenant_id} mono />}
          <DetailRow label="Role" value={displayRole} />
        </div>
      </div>

      {/* Divider */}
      <div style={{ height: 1, background: 'var(--border)' }} />

      {/* Session */}
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
          Active Session
        </span>
        <div
          style={{
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            padding: 14,
            display: 'flex',
            alignItems: 'center',
            gap: 10,
          }}
        >
          <div
            style={{
              width: 8,
              height: 8,
              borderRadius: '50%',
              background: 'var(--signal-healthy)',
              boxShadow: '0 0 0 3px color-mix(in srgb, var(--signal-healthy) 12%, transparent)',
              flexShrink: 0,
            }}
          />
          <div style={{ flex: 1 }}>
            <div
              style={{
                fontSize: 12,
                fontWeight: 500,
                color: 'var(--text-primary)',
                fontFamily: 'var(--font-sans)',
              }}
            >
              Current Session
            </div>
            <div
              style={{
                fontSize: 11,
                color: 'var(--text-faint)',
                fontFamily: 'var(--font-sans)',
                marginTop: 2,
              }}
            >
              Authenticated via Zitadel SSO &middot; JWT + HTTP-only cookie
            </div>
          </div>
          <Clock style={{ width: 14, height: 14, color: 'var(--text-faint)', strokeWidth: 1.5 }} />
        </div>
      </div>
    </div>
  );
}
