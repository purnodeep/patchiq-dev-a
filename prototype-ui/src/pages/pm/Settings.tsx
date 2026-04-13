import { motion } from 'framer-motion';
import { Save, Link, Mail, Key, CheckCircle, RefreshCw } from 'lucide-react';
import { useHotkeys } from '@/hooks/useHotkeys';
import { GlassCard } from '@/components/shared/GlassCard';
import { SectionHeader } from '@/components/shared/SectionHeader';

// ── Framer Motion variants ────────────────────────────────────────────────────
const stagger = {
  hidden: {},
  show: { transition: { staggerChildren: 0.06 } },
};

const fadeUp = {
  hidden: { opacity: 0, y: 12 },
  show: { opacity: 1, y: 0, transition: { duration: 0.4, ease: 'easeOut' } },
};

// ── Shared input style ────────────────────────────────────────────────────────
const inputStyle: React.CSSProperties = {
  background: 'transparent',
  border: '1px solid var(--color-separator)',
  borderRadius: 8,
  padding: '8px 12px',
  color: 'var(--color-foreground)',
  width: '100%',
  fontSize: 13,
  outline: 'none',
  boxSizing: 'border-box',
};

const labelStyle: React.CSSProperties = {
  display: 'block',
  fontSize: 11,
  fontWeight: 600,
  color: 'var(--color-muted)',
  marginBottom: 4,
  textTransform: 'uppercase',
  letterSpacing: '0.05em',
};

const fieldStyle: React.CSSProperties = {
  display: 'flex',
  flexDirection: 'column',
  gap: 0,
};

function FormField({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div style={fieldStyle}>
      <label style={labelStyle}>{label}</label>
      {children}
    </div>
  );
}

function PrimaryButton({ children, onClick }: { children: React.ReactNode; onClick?: () => void }) {
  return (
    <button
      onClick={onClick}
      style={{
        background: 'var(--color-primary)',
        color: '#fff',
        border: 'none',
        borderRadius: 8,
        padding: '8px 16px',
        fontSize: 13,
        fontWeight: 600,
        cursor: 'pointer',
        display: 'inline-flex',
        alignItems: 'center',
        gap: 6,
      }}
    >
      {children}
    </button>
  );
}

function OutlineButton({ children, onClick }: { children: React.ReactNode; onClick?: () => void }) {
  return (
    <button
      onClick={onClick}
      style={{
        background: 'transparent',
        color: 'var(--color-muted)',
        border: '1px solid var(--color-separator)',
        borderRadius: 8,
        padding: '8px 16px',
        fontSize: 13,
        fontWeight: 600,
        cursor: 'pointer',
        display: 'inline-flex',
        alignItems: 'center',
        gap: 6,
      }}
    >
      {children}
    </button>
  );
}

// ── Feature list ─────────────────────────────────────────────────────────────
const LICENSE_FEATURES = [
  'Endpoint Management',
  'Patch Deployment',
  'Workflow Engine',
  'Compliance Frameworks',
  'Multi-Wave Deployments',
  'API Access',
  'RBAC',
  'SSO Integration',
];

// ── Role mapping ──────────────────────────────────────────────────────────────
const ROLE_MAPPINGS = [
  { from: 'admin', to: 'Administrator' },
  { from: 'dev', to: 'Operator' },
  { from: 'readonly', to: 'Read Only' },
];

// ── Settings page ─────────────────────────────────────────────────────────────
export default function Settings() {
  useHotkeys();

  return (
    <motion.div
      variants={stagger}
      initial="hidden"
      animate="show"
      style={{
        display: 'flex',
        flexDirection: 'column',
        gap: 16,
        padding: '20px 24px',
        overflowY: 'auto',
        height: '100%',
      }}
    >
      {/* Page header */}
      <motion.div variants={fadeUp}>
        <h1 style={{ fontSize: 20, fontWeight: 700, margin: 0 }}>Settings</h1>
      </motion.div>

      {/* 2-column grid */}
      <motion.div
        variants={fadeUp}
        style={{
          display: 'grid',
          gridTemplateColumns: '1fr 1fr',
          gap: 16,
          alignItems: 'start',
        }}
      >
        {/* LEFT COLUMN */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          {/* General Settings */}
          <GlassCard className="p-5" hover={false}>
            <SectionHeader title="General Settings" />
            <div style={{ marginTop: 16, display: 'flex', flexDirection: 'column', gap: 12 }}>
              <FormField label="Organization Name">
                <input style={inputStyle} type="text" defaultValue="Acme Corp" />
              </FormField>
              <FormField label="Timezone">
                <select style={inputStyle}>
                  <option>UTC</option>
                  <option>America/New_York</option>
                  <option>America/Los_Angeles</option>
                  <option>Europe/London</option>
                </select>
              </FormField>
              <FormField label="Date Format">
                <select style={inputStyle}>
                  <option>YYYY-MM-DD</option>
                  <option>MM/DD/YYYY</option>
                  <option>DD/MM/YYYY</option>
                </select>
              </FormField>
              <FormField label="Default Scan Interval">
                <select style={inputStyle}>
                  <option>6 hours</option>
                  <option>12 hours</option>
                  <option>24 hours</option>
                  <option>48 hours</option>
                </select>
              </FormField>
              <div style={{ marginTop: 4 }}>
                <PrimaryButton>
                  <Save size={13} />
                  Save Changes
                </PrimaryButton>
              </div>
            </div>
          </GlassCard>

          {/* Integrations */}
          <GlassCard className="p-5" hover={false}>
            <SectionHeader
              title="Integrations"
              action={<Link size={14} color="var(--color-muted)" />}
            />
            <div style={{ marginTop: 16, display: 'flex', flexDirection: 'column', gap: 12 }}>
              <FormField label="Webhook URL">
                <div style={{ display: 'flex', gap: 8 }}>
                  <input
                    style={{ ...inputStyle, flex: 1 }}
                    type="text"
                    placeholder="https://hooks.example.com/..."
                  />
                  <button
                    style={{
                      background: 'transparent',
                      border: '1px solid var(--color-separator)',
                      borderRadius: 8,
                      padding: '8px 12px',
                      fontSize: 12,
                      fontWeight: 600,
                      color: 'var(--color-primary)',
                      cursor: 'pointer',
                      whiteSpace: 'nowrap',
                    }}
                  >
                    Test
                  </button>
                </div>
              </FormField>
              <FormField label="Slack Webhook">
                <input style={inputStyle} type="text" defaultValue="#patchiq-alerts" />
              </FormField>

              {/* SMTP section */}
              <div
                style={{
                  paddingTop: 8,
                  borderTop: '1px solid var(--color-separator)',
                }}
              >
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 6,
                    marginBottom: 12,
                  }}
                >
                  <Mail size={13} color="var(--color-muted)" />
                  <span
                    style={{
                      fontSize: 11,
                      fontWeight: 700,
                      color: 'var(--color-muted)',
                      textTransform: 'uppercase',
                      letterSpacing: '0.05em',
                    }}
                  >
                    SMTP
                  </span>
                </div>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
                  <FormField label="Host">
                    <input style={inputStyle} type="text" defaultValue="smtp.acme.com" />
                  </FormField>
                  <FormField label="Port">
                    <input style={inputStyle} type="text" defaultValue="587" />
                  </FormField>
                  <FormField label="From Address">
                    <input style={inputStyle} type="text" placeholder="noreply@acme.com" />
                  </FormField>
                  <FormField label="Username">
                    <input style={inputStyle} type="text" placeholder="smtp-user" />
                  </FormField>
                </div>
              </div>

              <div style={{ marginTop: 4 }}>
                <PrimaryButton>
                  <Save size={13} />
                  Save Integrations
                </PrimaryButton>
              </div>
            </div>
          </GlassCard>
        </div>

        {/* RIGHT COLUMN */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          {/* License */}
          <GlassCard className="p-5" hover={false}>
            <SectionHeader title="License" />
            <div style={{ marginTop: 16, display: 'flex', flexDirection: 'column', gap: 14 }}>
              {/* Enterprise badge */}
              <div>
                <span
                  style={{
                    background:
                      'linear-gradient(135deg, var(--color-purple), var(--color-primary))',
                    color: '#fff',
                    fontSize: 11,
                    fontWeight: 700,
                    padding: '4px 12px',
                    borderRadius: 20,
                    letterSpacing: '0.08em',
                    textTransform: 'uppercase',
                  }}
                >
                  Enterprise
                </span>
              </div>

              {/* Endpoint usage */}
              <div>
                <div
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    marginBottom: 6,
                  }}
                >
                  <span style={{ fontSize: 12, color: 'var(--color-muted)' }}>Endpoints</span>
                  <span style={{ fontSize: 13, fontWeight: 600, color: 'var(--color-foreground)' }}>
                    247 / 500
                  </span>
                </div>
                <div
                  style={{
                    height: 6,
                    background: 'var(--color-separator)',
                    borderRadius: 4,
                    overflow: 'hidden',
                  }}
                >
                  <div
                    style={{
                      height: '100%',
                      width: '49%',
                      background: 'var(--color-primary)',
                      borderRadius: 4,
                      transition: 'width 0.6s ease',
                    }}
                  />
                </div>
                <div
                  style={{
                    marginTop: 4,
                    fontSize: 11,
                    color: 'var(--color-muted)',
                  }}
                >
                  49% of license used
                </div>
              </div>

              {/* Feature list */}
              <div
                style={{
                  display: 'grid',
                  gridTemplateColumns: '1fr 1fr',
                  gap: '6px 8px',
                }}
              >
                {LICENSE_FEATURES.map((feature) => (
                  <div
                    key={feature}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 6,
                      fontSize: 12,
                      color: 'var(--color-foreground)',
                    }}
                  >
                    <CheckCircle size={12} color="var(--color-success)" />
                    {feature}
                  </div>
                ))}
              </div>

              {/* Expiry */}
              <div
                style={{
                  padding: '8px 12px',
                  background: 'color-mix(in srgb, var(--color-separator) 40%, transparent)',
                  borderRadius: 8,
                  fontSize: 12,
                  color: 'var(--color-muted)',
                }}
              >
                Expires{' '}
                <span style={{ color: 'var(--color-foreground)', fontWeight: 600 }}>
                  Dec 31, 2026
                </span>{' '}
                — 296 days remaining
              </div>

              <div>
                <OutlineButton>Manage License</OutlineButton>
              </div>
            </div>
          </GlassCard>

          {/* IAM / Identity Provider */}
          <GlassCard className="p-5" hover={false}>
            <SectionHeader
              title="IAM / Identity Provider"
              action={<Key size={14} color="var(--color-muted)" />}
            />
            <div style={{ marginTop: 16, display: 'flex', flexDirection: 'column', gap: 12 }}>
              {/* Connected provider */}
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  padding: '8px 12px',
                  background: 'color-mix(in srgb, var(--color-success) 8%, transparent)',
                  border: '1px solid color-mix(in srgb, var(--color-success) 30%, transparent)',
                  borderRadius: 8,
                }}
              >
                <span style={{ fontSize: 13, fontWeight: 600, color: 'var(--color-foreground)' }}>
                  Zitadel (OIDC)
                </span>
                <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                  <div
                    style={{
                      width: 8,
                      height: 8,
                      borderRadius: '50%',
                      background: 'var(--color-success)',
                    }}
                  />
                  <span style={{ fontSize: 11, color: 'var(--color-success)', fontWeight: 600 }}>
                    Connected
                  </span>
                </div>
              </div>

              <FormField label="SSO URL">
                <input
                  style={inputStyle}
                  type="text"
                  defaultValue="https://auth.acme.zitadel.cloud"
                />
              </FormField>
              <FormField label="Client ID">
                <div style={{ display: 'flex', gap: 8 }}>
                  <input
                    style={{ ...inputStyle, flex: 1, fontFamily: 'monospace', letterSpacing: 2 }}
                    type="password"
                    defaultValue="patchiq-prod-01"
                  />
                  <button
                    style={{
                      background: 'transparent',
                      border: '1px solid var(--color-separator)',
                      borderRadius: 8,
                      padding: '8px 12px',
                      fontSize: 12,
                      fontWeight: 600,
                      color: 'var(--color-muted)',
                      cursor: 'pointer',
                      whiteSpace: 'nowrap',
                    }}
                  >
                    Reveal
                  </button>
                </div>
              </FormField>

              {/* Role mapping table */}
              <div>
                <div style={{ ...labelStyle, marginBottom: 8 }}>Role Mapping</div>
                <div
                  style={{
                    border: '1px solid var(--color-separator)',
                    borderRadius: 8,
                    overflow: 'hidden',
                  }}
                >
                  <div
                    style={{
                      display: 'grid',
                      gridTemplateColumns: '1fr 1fr',
                      background: 'color-mix(in srgb, var(--color-separator) 40%, transparent)',
                      padding: '6px 12px',
                    }}
                  >
                    <span
                      style={{
                        fontSize: 11,
                        fontWeight: 700,
                        color: 'var(--color-muted)',
                        textTransform: 'uppercase',
                        letterSpacing: '0.05em',
                      }}
                    >
                      Zitadel Role
                    </span>
                    <span
                      style={{
                        fontSize: 11,
                        fontWeight: 700,
                        color: 'var(--color-muted)',
                        textTransform: 'uppercase',
                        letterSpacing: '0.05em',
                      }}
                    >
                      PatchIQ Role
                    </span>
                  </div>
                  {ROLE_MAPPINGS.map((mapping, i) => (
                    <div
                      key={mapping.from}
                      style={{
                        display: 'grid',
                        gridTemplateColumns: '1fr 1fr',
                        padding: '8px 12px',
                        borderTop: i > 0 ? '1px solid var(--color-separator)' : undefined,
                      }}
                    >
                      <span
                        style={{
                          fontSize: 12,
                          color: 'var(--color-cyan)',
                          fontFamily: 'monospace',
                        }}
                      >
                        {mapping.from}
                      </span>
                      <span style={{ fontSize: 12, color: 'var(--color-foreground)' }}>
                        {mapping.to}
                      </span>
                    </div>
                  ))}
                </div>
              </div>

              <div style={{ display: 'flex', gap: 8, marginTop: 4 }}>
                <OutlineButton>
                  <RefreshCw size={12} />
                  Test Connection
                </OutlineButton>
                <PrimaryButton>
                  <Save size={13} />
                  Save
                </PrimaryButton>
              </div>
            </div>
          </GlassCard>
        </div>
      </motion.div>
    </motion.div>
  );
}
