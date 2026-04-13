import { useState } from 'react';
import { useCan } from '../../app/auth/AuthContext';
import { Loader2, Copy, Check, Plus, Monitor, Server } from 'lucide-react';
import { Skeleton } from '@patchiq/ui';
import { toast } from 'sonner';
import { useAgentBinaries } from '../../api/hooks/useAgentBinaries';
import type { AgentBinaryInfo } from '../../api/hooks/useAgentBinaries';
import { useCreateRegistration } from '../../api/hooks/useEndpoints';

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
}

function osLabel(os: string): string {
  switch (os) {
    case 'linux':
      return 'Linux';
    case 'darwin':
      return 'macOS';
    case 'windows':
      return 'Windows';
    default:
      return os;
  }
}

function archLabel(arch: string): string {
  switch (arch) {
    case 'amd64':
      return 'x86_64';
    case 'arm64':
      return 'ARM64';
    default:
      return arch;
  }
}

function OsIcon({ os }: { os: string }) {
  const color =
    os === 'linux'
      ? 'var(--signal-warning)'
      : os === 'darwin'
        ? 'var(--text-muted)'
        : os === 'windows'
          ? 'var(--signal-info)'
          : 'var(--text-muted)';
  return <Monitor style={{ width: 14, height: 14, color, strokeWidth: 1.5 }} />;
}

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);

  function handleCopy() {
    navigator.clipboard.writeText(text).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }

  return (
    <button
      type="button"
      onClick={handleCopy}
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 4,
        height: 26,
        padding: '0 8px',
        background: 'transparent',
        border: '1px solid var(--border)',
        borderRadius: 5,
        fontSize: 10,
        color: copied ? 'var(--signal-healthy)' : 'var(--text-muted)',
        cursor: 'pointer',
        fontFamily: 'var(--font-mono)',
        transition: 'color 150ms',
      }}
    >
      {copied ? (
        <Check style={{ width: 10, height: 10 }} />
      ) : (
        <Copy style={{ width: 10, height: 10 }} />
      )}
      {copied ? 'Copied' : 'Copy'}
    </button>
  );
}

function BinaryRow({ binary }: { binary: AgentBinaryInfo }) {
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 10,
        padding: '10px 14px',
        borderBottom: '1px solid var(--border)',
      }}
    >
      <OsIcon os={binary.os} />
      <div style={{ flex: 1, minWidth: 0 }}>
        <div
          style={{
            fontSize: 12,
            fontWeight: 500,
            color: 'var(--text-primary)',
            fontFamily: 'var(--font-sans)',
          }}
        >
          {osLabel(binary.os)} {archLabel(binary.arch)}
        </div>
        <div
          style={{
            fontSize: 10,
            color: 'var(--text-faint)',
            fontFamily: 'var(--font-mono)',
            marginTop: 2,
          }}
        >
          {binary.filename} &middot; {formatBytes(binary.size)}
        </div>
      </div>
      <a
        href={binary.url}
        download
        style={{
          fontSize: 10,
          fontFamily: 'var(--font-mono)',
          color: 'var(--text-secondary)',
          background: 'transparent',
          border: '1px solid var(--border)',
          borderRadius: 5,
          padding: '4px 10px',
          textDecoration: 'none',
          cursor: 'pointer',
        }}
      >
        Download
      </a>
    </div>
  );
}

export function AgentFleetSettingsPage() {
  const can = useCan();
  const { data: binaries, isLoading: binariesLoading } = useAgentBinaries();
  const createRegistration = useCreateRegistration();
  const [lastToken, setLastToken] = useState<string | null>(null);

  function handleCreateToken() {
    createRegistration.mutate(undefined, {
      onSuccess: (reg) => {
        setLastToken(reg.registration_token);
        toast.success('Registration token created');
      },
      onError: (err) => {
        toast.error(`Failed to create token: ${err.message}`);
      },
    });
  }

  if (binariesLoading) {
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
          <Skeleton className="h-6 w-40" />
          <Skeleton className="h-4 w-72 mt-2" />
        </div>
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-40 w-full" />
      </div>
    );
  }

  // Group binaries by OS
  const grouped: Record<string, AgentBinaryInfo[]> = {};
  for (const b of binaries ?? []) {
    (grouped[b.os] ??= []).push(b);
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
          Agent Fleet
        </h2>
        <p
          style={{
            fontSize: 12,
            color: 'var(--text-muted)',
            margin: '4px 0 0',
          }}
        >
          Agent enrollment, downloads, and fleet configuration.
        </p>
      </div>

      {/* Enrollment Token Section */}
      <div>
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
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
            Enrollment Token
          </span>
          <button
            type="button"
            onClick={handleCreateToken}
            disabled={createRegistration.isPending || !can('endpoints', 'create')}
            title={!can('endpoints', 'create') ? "You don't have permission" : undefined}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 6,
              height: 30,
              padding: '0 12px',
              background: 'transparent',
              border: '1px solid var(--border)',
              borderRadius: 6,
              fontSize: 11,
              fontWeight: 500,
              color: 'var(--text-secondary)',
              cursor: createRegistration.isPending ? 'not-allowed' : 'pointer',
              fontFamily: 'var(--font-sans)',
              opacity: createRegistration.isPending ? 0.6 : 1,
            }}
          >
            {createRegistration.isPending ? (
              <Loader2 style={{ width: 12, height: 12 }} className="animate-spin" />
            ) : (
              <Plus style={{ width: 12, height: 12 }} />
            )}
            Generate Token
          </button>
        </div>

        <p
          style={{
            fontSize: 11,
            color: 'var(--text-faint)',
            fontFamily: 'var(--font-sans)',
            margin: '0 0 10px',
          }}
        >
          Generate a one-time enrollment token for new agent installations. Tokens are single-use
          and expire after registration.
        </p>

        {lastToken && (
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 10,
              padding: '10px 14px',
              background: 'color-mix(in srgb, var(--signal-healthy) 6%, transparent)',
              border: '1px solid color-mix(in srgb, var(--signal-healthy) 15%, transparent)',
              borderRadius: 8,
            }}
          >
            <div style={{ flex: 1, minWidth: 0 }}>
              <div
                style={{
                  fontSize: 10,
                  fontWeight: 600,
                  color: 'var(--signal-healthy)',
                  fontFamily: 'var(--font-sans)',
                  textTransform: 'uppercase',
                  letterSpacing: '0.05em',
                  marginBottom: 4,
                }}
              >
                New Token
              </div>
              <div
                style={{
                  fontSize: 12,
                  fontFamily: 'var(--font-mono)',
                  color: 'var(--text-primary)',
                  wordBreak: 'break-all',
                }}
              >
                {lastToken}
              </div>
            </div>
            <CopyButton text={lastToken} />
          </div>
        )}

        {!lastToken && (
          <div
            style={{
              padding: '20px',
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              borderRadius: 8,
              textAlign: 'center',
            }}
          >
            <div
              style={{ fontSize: 12, color: 'var(--text-faint)', fontFamily: 'var(--font-sans)' }}
            >
              No token generated this session. Click <strong>Generate Token</strong> to create one.
            </div>
          </div>
        )}
      </div>

      {/* Divider */}
      <div style={{ height: 1, background: 'var(--border)' }} />

      {/* Agent Binaries */}
      <div>
        <span
          style={{
            fontSize: 13,
            fontWeight: 600,
            color: 'var(--text-emphasis)',
            fontFamily: 'var(--font-sans)',
            display: 'block',
            marginBottom: 4,
          }}
        >
          Agent Downloads
        </span>
        <p
          style={{
            fontSize: 11,
            color: 'var(--text-faint)',
            fontFamily: 'var(--font-sans)',
            margin: '0 0 12px',
          }}
        >
          Pre-compiled agent binaries for supported platforms.
        </p>

        {(binaries ?? []).length > 0 ? (
          <div
            style={{
              borderRadius: 8,
              border: '1px solid var(--border)',
              overflow: 'hidden',
            }}
          >
            {(binaries ?? []).map((b) => (
              <BinaryRow key={`${b.os}-${b.arch}`} binary={b} />
            ))}
          </div>
        ) : (
          <div
            style={{
              padding: '24px',
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              borderRadius: 8,
              textAlign: 'center',
            }}
          >
            <Server
              style={{
                width: 24,
                height: 24,
                color: 'var(--text-faint)',
                strokeWidth: 1.5,
                margin: '0 auto 8px',
              }}
            />
            <div
              style={{ fontSize: 12, color: 'var(--text-faint)', fontFamily: 'var(--font-sans)' }}
            >
              No agent binaries available. Build agents with{' '}
              <code style={{ fontFamily: 'var(--font-mono)', fontSize: 11 }}>
                make build-agents
              </code>
              .
            </div>
          </div>
        )}
      </div>

      {/* Divider */}
      <div style={{ height: 1, background: 'var(--border)' }} />

      {/* Install Command */}
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
          Quick Install
        </span>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '10px 14px',
            background: 'var(--bg-inset)',
            border: '1px solid var(--border)',
            borderRadius: 8,
          }}
        >
          <code
            style={{
              fontSize: 11,
              fontFamily: 'var(--font-mono)',
              color: 'var(--text-secondary)',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
          >
            {lastToken
              ? `curl -sSL <server>/install.sh | sudo bash -s -- --token ${lastToken}`
              : 'curl -sSL <server>/install.sh | sudo bash -s -- --token <TOKEN>'}
          </code>
          <CopyButton
            text={
              lastToken
                ? `curl -sSL ${window.location.origin}/install.sh | sudo bash -s -- --token ${lastToken}`
                : `curl -sSL ${window.location.origin}/install.sh | sudo bash -s -- --token <TOKEN>`
            }
          />
        </div>
        <p
          style={{
            fontSize: 11,
            color: 'var(--text-faint)',
            fontFamily: 'var(--font-sans)',
            marginTop: 6,
          }}
        >
          Run on the target machine. Requires root/admin privileges.
        </p>
      </div>
    </div>
  );
}
