import { ShieldCheck } from 'lucide-react';

interface AuthLayoutProps {
  children: React.ReactNode;
}

export function AuthLayout({ children }: AuthLayoutProps) {
  return (
    <div
      style={{
        display: 'flex',
        minHeight: '100vh',
        background: 'var(--bg-page)',
      }}
    >
      {/* Left panel — branding, hidden on mobile */}
      <div
        style={{
          padding: '3rem',
          background: 'var(--bg-inset)',
          borderRight: '1px solid var(--border)',
          position: 'relative',
          overflow: 'hidden',
          width: '50%',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
        }}
        className="hidden lg:flex"
      >
        {/* Subtle grid pattern */}
        <div
          style={{
            position: 'absolute',
            inset: 0,
            backgroundImage:
              'linear-gradient(var(--border) 1px, transparent 1px), linear-gradient(90deg, var(--border) 1px, transparent 1px)',
            backgroundSize: '32px 32px',
            opacity: 0.4,
          }}
        />
        <div
          style={{
            position: 'relative',
            zIndex: 1,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: '1.5rem',
            maxWidth: '360px',
            textAlign: 'center',
          }}
        >
          <div
            style={{
              display: 'flex',
              height: '64px',
              width: '64px',
              alignItems: 'center',
              justifyContent: 'center',
              borderRadius: '16px',
              background: 'var(--accent)',
              boxShadow: '0 0 0 8px color-mix(in srgb, var(--accent) 12%, transparent)',
            }}
          >
            <ShieldCheck
              style={{ height: '32px', width: '32px', color: 'var(--text-on-color, #fff)' }}
            />
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
            <h1
              style={{
                fontSize: '28px',
                fontWeight: 700,
                letterSpacing: '-0.025em',
                color: 'var(--text-emphasis)',
                fontFamily: 'var(--font-sans)',
                margin: 0,
              }}
            >
              PatchIQ Hub
            </h1>
            <p
              style={{
                fontSize: '15px',
                color: 'var(--text-secondary)',
                lineHeight: 1.6,
                margin: 0,
              }}
            >
              Centralized patch catalog, feed aggregation, and fleet management for your PatchIQ
              deployment.
            </p>
          </div>
          <div
            style={{
              display: 'flex',
              flexDirection: 'column',
              gap: '12px',
              width: '100%',
              marginTop: '8px',
            }}
          >
            {[
              'Multi-source feed aggregation',
              'Centralized patch catalog',
              'Fleet license management',
            ].map((feature) => (
              <div
                key={feature}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '10px',
                  background: 'var(--bg-card)',
                  border: '1px solid var(--border)',
                  borderRadius: '8px',
                  padding: '10px 14px',
                  textAlign: 'left',
                }}
              >
                <div
                  style={{
                    width: '6px',
                    height: '6px',
                    borderRadius: '50%',
                    background: 'var(--accent)',
                    flexShrink: 0,
                  }}
                />
                <span style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>{feature}</span>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Right panel — form content */}
      <div
        style={{
          flex: 1,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          padding: '2rem',
        }}
      >
        <div style={{ width: '100%', maxWidth: '420px' }}>
          {/* Mobile logo */}
          <div
            style={{
              alignItems: 'center',
              justifyContent: 'center',
              gap: 8,
              marginBottom: 32,
            }}
            className="flex lg:hidden"
          >
            <div
              style={{
                display: 'flex',
                height: '32px',
                width: '32px',
                alignItems: 'center',
                justifyContent: 'center',
                borderRadius: '8px',
                background: 'var(--accent)',
              }}
            >
              <ShieldCheck
                style={{ height: '18px', width: '18px', color: 'var(--text-on-color, #fff)' }}
              />
            </div>
            <span
              style={{
                fontSize: '18px',
                fontWeight: 700,
                color: 'var(--text-emphasis)',
                fontFamily: 'var(--font-sans)',
              }}
            >
              PatchIQ Hub
            </span>
          </div>
          {children}
        </div>
      </div>
    </div>
  );
}
