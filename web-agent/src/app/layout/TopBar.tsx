import { Sun, Moon } from 'lucide-react';
import { useTheme } from '@patchiq/ui';
import { useAgentStatus } from '../../api/hooks/useStatus';

export const TopBar = () => {
  const { data } = useAgentStatus();
  const { resolvedMode, setMode } = useTheme();
  const isConnected = data?.enrollment_status === 'enrolled';
  const hostname = data?.hostname;

  return (
    <>
      <style>{`
        @keyframes pulse-dot {
          0%, 100% { opacity: 1; }
          50% { opacity: 0.4; }
        }
        .conn-dot-pulse {
          animation: pulse-dot 2s ease-in-out infinite;
        }
      `}</style>
      <header
        style={{
          position: 'fixed',
          top: 0,
          left: 0,
          right: 0,
          zIndex: 100,
          height: '48px',
          background: 'var(--bg-topbar, var(--bg-card))',
          borderBottom: '1px solid var(--border)',
          display: 'flex',
          alignItems: 'center',
          padding: '0 16px',
        }}
      >
        {/* Left: brand */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            minWidth: '180px',
          }}
        >
          <img
            src="/infraon-logo.png"
            alt="Infraon"
            style={{
              height: 28,
              objectFit: 'contain',
            }}
          />
        </div>

        {/* Center: hostname */}
        <div
          style={{
            flex: 1,
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
          }}
        >
          {hostname && (
            <span
              style={{
                fontFamily: 'var(--font-mono, "Geist Mono", monospace)',
                fontSize: '12px',
                color: 'var(--text-muted)',
                letterSpacing: '0.02em',
              }}
            >
              {hostname}
            </span>
          )}
        </div>

        {/* Right: theme toggle + connection status */}
        <div
          style={{
            minWidth: '180px',
            display: 'flex',
            justifyContent: 'flex-end',
            alignItems: 'center',
            gap: '6px',
          }}
        >
          {/* Theme toggle */}
          <button
            type="button"
            onClick={() => setMode(resolvedMode === 'dark' ? 'light' : 'dark')}
            aria-label="Toggle theme"
            style={{
              width: 32,
              height: 32,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              borderRadius: 6,
              background: 'transparent',
              border: 'none',
              cursor: 'pointer',
              color: 'var(--text-muted)',
              transition: 'color 0.1s',
            }}
            onMouseEnter={(e) => {
              (e.currentTarget as HTMLButtonElement).style.color = 'var(--text-secondary)';
            }}
            onMouseLeave={(e) => {
              (e.currentTarget as HTMLButtonElement).style.color = 'var(--text-muted)';
            }}
          >
            {resolvedMode === 'dark' ? (
              <Sun style={{ width: 16, height: 16, strokeWidth: 1.5 }} />
            ) : (
              <Moon style={{ width: 16, height: 16, strokeWidth: 1.5 }} />
            )}
          </button>

          {/* Connection dot */}
          <div
            className={isConnected ? 'conn-dot-pulse' : ''}
            style={{
              width: '6px',
              height: '6px',
              borderRadius: '50%',
              background: isConnected ? 'var(--signal-healthy)' : 'var(--signal-critical)',
              boxShadow: isConnected
                ? '0 0 6px color-mix(in srgb, var(--signal-healthy) 60%, transparent)'
                : 'none',
            }}
          />
          <span
            style={{
              fontSize: '12px',
              color: 'var(--text-faint)',
            }}
          >
            {isConnected ? 'Connected' : (data?.enrollment_status ?? 'Connecting...')}
          </span>
        </div>
      </header>
    </>
  );
};
