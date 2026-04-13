import { useState, useEffect } from 'react';
import { toast } from 'sonner';
import {
  Button,
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
  Input,
  Skeleton,
} from '@patchiq/ui';
import {
  Trash2,
  ChevronDown,
  ChevronUp,
  Shield,
  MonitorCheck,
  Package,
  ScanSearch,
  AlertTriangle,
  Rocket,
  Activity,
  Bug,
  Plus,
} from 'lucide-react';
import { useCan } from '../../../app/auth/AuthContext';
import {
  useCreateCustomFramework,
  useUpdateCustomFramework,
  useDeleteCustomFramework,
  useUpdateCustomControls,
  useCustomFrameworkControls,
  type CustomFrameworkResponse,
  type CustomControlInput,
  type SLATierInput,
  type CheckConfig,
  type ConditionValue,
} from '../../../api/hooks/useCompliance';

// ---------------------------------------------------------------
// Types & Constants
// ---------------------------------------------------------------

interface EditableControl extends CustomControlInput {
  _key: string;
  check_config?: CheckConfig;
}

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  existing?: CustomFrameworkResponse;
}

interface ConditionDef {
  key: string;
  name: string;
  description: string;
  type: 'boolean' | 'threshold' | 'duration';
  default: number; // 1 = true for boolean
  unit: string; // 'hours', 'days', '%', ''
  min: number;
  max: number;
}

interface CheckTypeDef {
  type: string;
  name: string;
  description: string;
  icon: typeof Shield;
  color: string;
  conditions: ConditionDef[];
  defaultPassThreshold: number;
  defaultPartialThreshold: number;
}

const CHECK_TYPES: CheckTypeDef[] = [
  {
    type: 'sla',
    name: 'SLA Compliance',
    description: 'Patch remediation within CVSS-based SLA deadlines',
    icon: Shield,
    color: 'var(--accent)',
    conditions: [], // SLA uses SLA tiers, not conditions
    defaultPassThreshold: 95,
    defaultPartialThreshold: 70,
  },
  {
    type: 'asset_inventory',
    name: 'Asset Inventory',
    description: 'Endpoints enrolled with hardware data and recent heartbeat',
    icon: MonitorCheck,
    color: 'var(--signal-healthy)',
    conditions: [
      {
        key: 'endpoint_enrolled',
        name: 'Endpoint Enrolled',
        description: 'Endpoint is registered and active in the system',
        type: 'boolean',
        default: 1,
        unit: '',
        min: 0,
        max: 1,
      },
      {
        key: 'has_hardware_data',
        name: 'Has Hardware Data',
        description: 'Endpoint has reported CPU, memory, and disk information',
        type: 'boolean',
        default: 1,
        unit: '',
        min: 0,
        max: 1,
      },
      {
        key: 'heartbeat_freshness',
        name: 'Heartbeat Freshness',
        description: 'Agent has reported within this time window',
        type: 'duration',
        default: 24,
        unit: 'hours',
        min: 1,
        max: 168,
      },
    ],
    defaultPassThreshold: 95,
    defaultPartialThreshold: 70,
  },
  {
    type: 'software_inventory',
    name: 'Software Inventory',
    description: 'Package scans completed within configurable window',
    icon: Package,
    color: '#6366f1',
    conditions: [
      {
        key: 'has_recent_scan',
        name: 'Recent Package Scan',
        description: 'Endpoint has completed a package inventory scan',
        type: 'boolean',
        default: 1,
        unit: '',
        min: 0,
        max: 1,
      },
      {
        key: 'scan_max_age',
        name: 'Max Scan Age',
        description: 'Maximum age of the most recent package scan',
        type: 'duration',
        default: 7,
        unit: 'days',
        min: 1,
        max: 90,
      },
    ],
    defaultPassThreshold: 95,
    defaultPartialThreshold: 70,
  },
  {
    type: 'vuln_scanning',
    name: 'Vulnerability Scanning',
    description: 'CVE vulnerability scan coverage across endpoints',
    icon: ScanSearch,
    color: '#0ea5e9',
    conditions: [
      {
        key: 'has_cve_data',
        name: 'CVE Scan Coverage',
        description: 'Endpoint has been evaluated for known CVE vulnerabilities',
        type: 'boolean',
        default: 1,
        unit: '',
        min: 0,
        max: 1,
      },
    ],
    defaultPassThreshold: 95,
    defaultPartialThreshold: 70,
  },
  {
    type: 'kev_compliance',
    name: 'CISA KEV Compliance',
    description: 'Zero CISA Known Exploited Vulnerabilities',
    icon: AlertTriangle,
    color: 'var(--signal-critical)',
    conditions: [
      {
        key: 'zero_kev',
        name: 'Zero KEV Exposure',
        description: 'No CISA Known Exploited Vulnerabilities present on any endpoint',
        type: 'boolean',
        default: 1,
        unit: '',
        min: 0,
        max: 1,
      },
    ],
    defaultPassThreshold: 100,
    defaultPartialThreshold: 95,
  },
  {
    type: 'deployment_governance',
    name: 'Deployment Governance',
    description: 'Patch deployment success rates above configurable threshold',
    icon: Rocket,
    color: '#f59e0b',
    conditions: [
      {
        key: 'min_success_rate',
        name: 'Min Success Rate',
        description: 'Required deployment success percentage',
        type: 'threshold',
        default: 80,
        unit: '%',
        min: 0,
        max: 100,
      },
      {
        key: 'lookback_days',
        name: 'Lookback Period',
        description: 'Days of deployment history to evaluate',
        type: 'duration',
        default: 30,
        unit: 'days',
        min: 1,
        max: 365,
      },
    ],
    defaultPassThreshold: 90,
    defaultPartialThreshold: 70,
  },
  {
    type: 'agent_monitoring',
    name: 'Agent Monitoring',
    description: '95%+ endpoints with recent agent heartbeats',
    icon: Activity,
    color: '#10b981',
    conditions: [
      {
        key: 'heartbeat_freshness',
        name: 'Heartbeat Freshness',
        description: 'Agent must have reported within this time window',
        type: 'duration',
        default: 24,
        unit: 'hours',
        min: 1,
        max: 168,
      },
    ],
    defaultPassThreshold: 95,
    defaultPartialThreshold: 80,
  },
  {
    type: 'critical_vuln_remediation',
    name: 'Critical Vuln Remediation',
    description: 'No critical/high CVEs unpatched beyond allowed window',
    icon: Bug,
    color: '#ef4444',
    conditions: [
      {
        key: 'max_age_days',
        name: 'Max Unpatched Age',
        description: 'Maximum days a critical/high CVE can remain unpatched',
        type: 'duration',
        default: 30,
        unit: 'days',
        min: 1,
        max: 365,
      },
      {
        key: 'include_high',
        name: 'Include High Severity',
        description: 'Also check high-severity CVEs, not just critical',
        type: 'boolean',
        default: 1,
        unit: '',
        min: 0,
        max: 1,
      },
    ],
    defaultPassThreshold: 100,
    defaultPartialThreshold: 90,
  },
];

// Pre-filled templates for each check type
const CONTROL_TEMPLATES: Record<
  string,
  { id: string; name: string; category: string; description: string; hint: string }
> = {
  sla: {
    id: 'SLA-001',
    name: 'Patch SLA Compliance',
    category: 'Vulnerability Management',
    description: 'Evaluate patch remediation against CVSS-based SLA deadlines',
    hint: 'Deploy patches within defined SLA timelines using the deployment workflow',
  },
  asset_inventory: {
    id: 'ASSET-001',
    name: 'Asset Inventory Coverage',
    category: 'Asset Management',
    description: 'Verify all endpoints are enrolled with hardware data and actively reporting',
    hint: 'Ensure all endpoints have the agent installed and reporting hardware inventory',
  },
  software_inventory: {
    id: 'SW-001',
    name: 'Software Inventory Currency',
    category: 'Asset Management',
    description: 'Verify all endpoints have completed a package scan within the last 7 days',
    hint: 'Configure agent scan intervals to run at least weekly',
  },
  vuln_scanning: {
    id: 'VULN-001',
    name: 'Vulnerability Scan Coverage',
    category: 'Vulnerability Management',
    description: 'Verify all endpoints are covered by CVE vulnerability scanning',
    hint: 'Ensure agents are running inventory scans for CVE correlation',
  },
  kev_compliance: {
    id: 'KEV-001',
    name: 'CISA KEV Compliance',
    category: 'Vulnerability Management',
    description: 'Verify no endpoints have CISA Known Exploited Vulnerabilities',
    hint: 'Prioritize remediation of CISA KEV entries immediately',
  },
  deployment_governance: {
    id: 'DEPLOY-001',
    name: 'Deployment Governance',
    category: 'Change Management',
    description: 'Verify patch deployments have acceptable success rates',
    hint: 'Route all patch deployments through approved workflows',
  },
  agent_monitoring: {
    id: 'MON-001',
    name: 'Agent Monitoring',
    category: 'Operations',
    description: 'Verify 95%+ of endpoints have recent agent heartbeats',
    hint: 'Investigate and remediate offline agents promptly',
  },
  critical_vuln_remediation: {
    id: 'CRIT-001',
    name: 'Critical Vuln Remediation',
    category: 'Vulnerability Management',
    description: 'Verify no critical/high CVEs are unpatched for over 30 days',
    hint: 'Remediate critical and high-severity CVEs within 30 days of detection',
  },
};

const DEFAULT_SLA_TIERS: SLATierInput[] = [
  { label: 'critical', days: 15, cvss_min: 9.0, cvss_max: 10.0 },
  { label: 'high', days: 30, cvss_min: 7.0, cvss_max: 8.9 },
  { label: 'medium', days: 90, cvss_min: 4.0, cvss_max: 6.9 },
  { label: 'low', days: null, cvss_min: 0.0, cvss_max: 3.9 },
];

function makeKey() {
  return Math.random().toString(36).slice(2);
}

function buildDefaultCheckConfig(checkType: string): CheckConfig {
  const typeDef = CHECK_TYPES.find((ct) => ct.type === checkType);
  const conditions: Record<string, ConditionValue> = {};
  if (typeDef) {
    for (const cond of typeDef.conditions) {
      conditions[cond.key] = {
        enabled: true,
        value: cond.type !== 'boolean' ? cond.default : undefined,
      };
    }
  }
  return {
    conditions,
    pass_threshold: typeDef?.defaultPassThreshold ?? 95,
    partial_threshold: typeDef?.defaultPartialThreshold ?? 70,
  };
}

function makeControlFromTemplate(checkType: string): EditableControl {
  const tmpl = CONTROL_TEMPLATES[checkType];
  const checkConfig = buildDefaultCheckConfig(checkType);
  if (!tmpl) {
    return {
      _key: makeKey(),
      control_id: '',
      name: '',
      description: '',
      category: 'General',
      check_type: checkType,
      remediation_hint: '',
      sla_tiers: checkType === 'sla' ? DEFAULT_SLA_TIERS.map((t) => ({ ...t })) : [],
      check_config: checkConfig,
    };
  }
  return {
    _key: makeKey(),
    control_id: tmpl.id,
    name: tmpl.name,
    description: tmpl.description,
    category: tmpl.category,
    check_type: checkType,
    remediation_hint: tmpl.hint,
    sla_tiers: checkType === 'sla' ? DEFAULT_SLA_TIERS.map((t) => ({ ...t })) : [],
    check_config: checkConfig,
  };
}

const labelStyle: React.CSSProperties = {
  display: 'block',
  fontSize: 10,
  fontFamily: 'var(--font-mono)',
  color: 'var(--text-muted)',
  textTransform: 'uppercase',
  letterSpacing: '0.06em',
  marginBottom: 4,
};

// ---------------------------------------------------------------
// Condition Configurator
// ---------------------------------------------------------------

function ConditionConfigurator({
  checkType,
  config,
  onChange,
}: {
  checkType: string;
  config: CheckConfig;
  onChange: (config: CheckConfig) => void;
}) {
  const typeDef = CHECK_TYPES.find((ct) => ct.type === checkType);
  if (!typeDef || typeDef.conditions.length === 0) return null;

  const conditions = config.conditions ?? {};

  function toggleCondition(key: string) {
    const current = conditions[key] ?? { enabled: true };
    onChange({
      ...config,
      conditions: {
        ...conditions,
        [key]: { ...current, enabled: !current.enabled },
      },
    });
  }

  function setConditionValue(key: string, value: number) {
    const current = conditions[key] ?? { enabled: true };
    onChange({
      ...config,
      conditions: {
        ...conditions,
        [key]: { ...current, value },
      },
    });
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      {/* Conditions */}
      <div>
        <label style={labelStyle}>Conditions</label>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
          {typeDef.conditions.map((cond) => {
            const cv = conditions[cond.key] ?? { enabled: true, value: cond.default };
            const isEnabled = cv.enabled !== false;

            return (
              <div
                key={cond.key}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 10,
                  padding: '8px 10px',
                  borderRadius: 6,
                  border: '1px solid var(--border)',
                  background: isEnabled ? 'var(--bg-card)' : 'var(--bg-inset)',
                  opacity: isEnabled ? 1 : 0.6,
                  transition: 'all 0.15s',
                }}
              >
                {/* Toggle */}
                <button
                  type="button"
                  onClick={() => toggleCondition(cond.key)}
                  style={{
                    width: 32,
                    height: 18,
                    borderRadius: 9,
                    flexShrink: 0,
                    background: isEnabled ? 'var(--accent)' : 'var(--border)',
                    border: 'none',
                    cursor: 'pointer',
                    position: 'relative',
                    transition: 'background 0.15s',
                  }}
                >
                  <span
                    style={{
                      position: 'absolute',
                      top: 2,
                      left: isEnabled ? 16 : 2,
                      width: 14,
                      height: 14,
                      borderRadius: '50%',
                      background: 'white',
                      transition: 'left 0.15s',
                    }}
                  />
                </button>

                {/* Label + description */}
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ fontSize: 12, fontWeight: 500, color: 'var(--text-primary)' }}>
                    {cond.name}
                  </div>
                  <div style={{ fontSize: 10, color: 'var(--text-muted)', lineHeight: 1.3 }}>
                    {cond.description}
                  </div>
                </div>

                {/* Value input for duration/threshold types */}
                {cond.type !== 'boolean' && isEnabled && (
                  <div style={{ display: 'flex', alignItems: 'center', gap: 4, flexShrink: 0 }}>
                    <Input
                      type="number"
                      min={cond.min}
                      max={cond.max}
                      value={cv.value ?? cond.default}
                      onChange={(e) =>
                        setConditionValue(cond.key, parseFloat(e.target.value) || cond.default)
                      }
                      style={{
                        width: 64,
                        fontSize: 12,
                        fontFamily: 'var(--font-mono)',
                        textAlign: 'center',
                      }}
                    />
                    <span
                      style={{
                        fontSize: 10,
                        fontFamily: 'var(--font-mono)',
                        color: 'var(--text-muted)',
                      }}
                    >
                      {cond.unit}
                    </span>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </div>

      {/* Pass/Partial thresholds */}
      <div>
        <label style={labelStyle}>Thresholds</label>
        <div style={{ display: 'flex', gap: 12 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
            <span
              style={{
                fontSize: 11,
                color: 'var(--signal-healthy)',
                fontFamily: 'var(--font-mono)',
                fontWeight: 600,
              }}
            >
              Pass
            </span>
            <Input
              type="number"
              min={0}
              max={100}
              value={config.pass_threshold ?? typeDef.defaultPassThreshold}
              onChange={(e) =>
                onChange({ ...config, pass_threshold: parseFloat(e.target.value) || 95 })
              }
              style={{
                width: 56,
                fontSize: 12,
                fontFamily: 'var(--font-mono)',
                textAlign: 'center',
              }}
            />
            <span style={{ fontSize: 10, color: 'var(--text-muted)' }}>%</span>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
            <span
              style={{
                fontSize: 11,
                color: 'var(--signal-warning)',
                fontFamily: 'var(--font-mono)',
                fontWeight: 600,
              }}
            >
              Partial
            </span>
            <Input
              type="number"
              min={0}
              max={100}
              value={config.partial_threshold ?? typeDef.defaultPartialThreshold}
              onChange={(e) =>
                onChange({ ...config, partial_threshold: parseFloat(e.target.value) || 70 })
              }
              style={{
                width: 56,
                fontSize: 12,
                fontFamily: 'var(--font-mono)',
                textAlign: 'center',
              }}
            />
            <span style={{ fontSize: 10, color: 'var(--text-muted)' }}>%</span>
          </div>
        </div>
        <div
          style={{
            fontSize: 10,
            color: 'var(--text-muted)',
            fontFamily: 'var(--font-mono)',
            marginTop: 4,
          }}
        >
          Below partial threshold = fail
        </div>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------
// Control Card (replaces the old spreadsheet-style ControlRow)
// ---------------------------------------------------------------

function ControlCard({
  control,
  onChange,
  onRemove,
}: {
  control: EditableControl;
  onChange: (updated: EditableControl) => void;
  onRemove: () => void;
}) {
  const [expanded, setExpanded] = useState(false);
  const checkInfo = CHECK_TYPES.find((ct) => ct.type === (control.check_type || 'sla'));
  const Icon = checkInfo?.icon ?? Shield;

  function updateField(field: keyof CustomControlInput, value: string) {
    onChange({ ...control, [field]: value });
  }

  function updateTierDays(tierLabel: string, days: string) {
    const parsed = days === '' ? null : parseInt(days, 10);
    onChange({
      ...control,
      sla_tiers: control.sla_tiers.map((t) =>
        t.label === tierLabel ? { ...t, days: Number.isNaN(parsed as number) ? null : parsed } : t,
      ),
    });
  }

  return (
    <div
      style={{
        border: '1px solid var(--border)',
        borderRadius: 8,
        overflow: 'hidden',
        marginBottom: 8,
        transition: 'box-shadow 0.15s',
      }}
    >
      {/* Card header: check type badge + ID + name + actions */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 10,
          padding: '10px 12px',
          background: 'var(--bg-inset)',
          cursor: 'pointer',
        }}
        onClick={() => setExpanded((v) => !v)}
      >
        {/* Check type icon + badge */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 6,
            padding: '3px 8px 3px 6px',
            borderRadius: 5,
            background: `color-mix(in srgb, ${checkInfo?.color ?? 'var(--accent)'} 10%, transparent)`,
            border: `1px solid color-mix(in srgb, ${checkInfo?.color ?? 'var(--accent)'} 20%, transparent)`,
            flexShrink: 0,
          }}
        >
          <Icon size={12} style={{ color: checkInfo?.color ?? 'var(--accent)' }} />
          <span
            style={{
              fontSize: 10,
              fontFamily: 'var(--font-mono)',
              fontWeight: 600,
              color: checkInfo?.color ?? 'var(--accent)',
            }}
          >
            {checkInfo?.name ?? 'Custom'}
          </span>
        </div>

        {/* Control ID */}
        <span
          style={{
            fontSize: 11,
            fontFamily: 'var(--font-mono)',
            color: 'var(--text-muted)',
            flexShrink: 0,
          }}
        >
          {control.control_id || '—'}
        </span>

        {/* Control name */}
        <span
          style={{
            fontSize: 12,
            color: 'var(--text-primary)',
            flex: 1,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          }}
        >
          {control.name || 'Unnamed control'}
        </span>

        {/* Actions */}
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation();
            setExpanded((v) => !v);
          }}
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            width: 26,
            height: 26,
            border: '1px solid var(--border)',
            borderRadius: 5,
            background: 'transparent',
            cursor: 'pointer',
            color: 'var(--text-secondary)',
            flexShrink: 0,
          }}
        >
          {expanded ? <ChevronUp size={12} /> : <ChevronDown size={12} />}
        </button>
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation();
            onRemove();
          }}
          title="Remove control"
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            width: 26,
            height: 26,
            border: '1px solid var(--border)',
            borderRadius: 5,
            background: 'transparent',
            cursor: 'pointer',
            color: 'var(--signal-critical)',
            flexShrink: 0,
          }}
        >
          <Trash2 size={12} />
        </button>
      </div>

      {/* Expanded edit area */}
      {expanded && (
        <div
          style={{
            padding: '14px',
            display: 'flex',
            flexDirection: 'column',
            gap: 12,
            borderTop: '1px solid var(--border)',
            background: 'var(--bg-card)',
          }}
        >
          {/* ID + Name row */}
          <div style={{ display: 'flex', gap: 10 }}>
            <div style={{ width: 120, flexShrink: 0 }}>
              <label style={labelStyle}>Control ID</label>
              <Input
                value={control.control_id}
                onChange={(e) => updateField('control_id', e.target.value)}
                style={{ width: '100%', fontFamily: 'var(--font-mono)', fontSize: 11 }}
              />
            </div>
            <div style={{ flex: 1 }}>
              <label style={labelStyle}>Name</label>
              <Input
                value={control.name}
                onChange={(e) => updateField('name', e.target.value)}
                style={{ width: '100%', fontSize: 12 }}
              />
            </div>
            <div style={{ width: 130, flexShrink: 0 }}>
              <label style={labelStyle}>Category</label>
              <Input
                value={control.category}
                onChange={(e) => updateField('category', e.target.value)}
                style={{ width: '100%', fontSize: 12 }}
              />
            </div>
          </div>

          {/* Description + Remediation */}
          <div style={{ display: 'flex', gap: 10 }}>
            <div style={{ flex: 1 }}>
              <label style={labelStyle}>Description</label>
              <Input
                placeholder="What this control checks"
                value={control.description}
                onChange={(e) => updateField('description', e.target.value)}
                style={{ fontSize: 12, width: '100%' }}
              />
            </div>
            <div style={{ flex: 1 }}>
              <label style={labelStyle}>Remediation Hint</label>
              <Input
                placeholder="How to fix when failing"
                value={control.remediation_hint}
                onChange={(e) => updateField('remediation_hint', e.target.value)}
                style={{ fontSize: 12, width: '100%' }}
              />
            </div>
          </div>

          {/* SLA Tiers — only for SLA check type */}
          {(!control.check_type || control.check_type === 'sla') && (
            <div>
              <label style={labelStyle}>SLA Tiers (days to remediate, blank = no SLA)</label>
              <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
                {control.sla_tiers.map((tier) => (
                  <div
                    key={tier.label}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 6,
                      background: 'var(--bg-inset)',
                      border: '1px solid var(--border)',
                      borderRadius: 6,
                      padding: '4px 8px',
                    }}
                  >
                    <span
                      style={{
                        fontSize: 10,
                        fontFamily: 'var(--font-mono)',
                        color: 'var(--text-secondary)',
                        textTransform: 'uppercase',
                        width: 52,
                      }}
                    >
                      {tier.label}
                    </span>
                    <Input
                      type="number"
                      placeholder="—"
                      value={tier.days === null ? '' : String(tier.days)}
                      onChange={(e) => updateTierDays(tier.label, e.target.value)}
                      style={{
                        width: 60,
                        fontSize: 12,
                        fontFamily: 'var(--font-mono)',
                        textAlign: 'center',
                      }}
                    />
                    <span style={{ fontSize: 10, color: 'var(--text-muted)' }}>d</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Condition configurator — for all non-SLA check types */}
          {control.check_type && control.check_type !== 'sla' && (
            <ConditionConfigurator
              checkType={control.check_type}
              config={
                control.check_config ?? {
                  conditions: {},
                  pass_threshold:
                    CHECK_TYPES.find((ct) => ct.type === control.check_type)
                      ?.defaultPassThreshold ?? 95,
                  partial_threshold:
                    CHECK_TYPES.find((ct) => ct.type === control.check_type)
                      ?.defaultPartialThreshold ?? 70,
                }
              }
              onChange={(newConfig) => onChange({ ...control, check_config: newConfig })}
            />
          )}
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------
// Template Picker (the key UX improvement)
// ---------------------------------------------------------------

function TemplatePicker({
  existingTypes,
  onAdd,
}: {
  existingTypes: Set<string>;
  onAdd: (checkType: string) => void;
}) {
  return (
    <div>
      <div
        style={{
          fontSize: 10,
          fontFamily: 'var(--font-mono)',
          color: 'var(--text-muted)',
          textTransform: 'uppercase',
          letterSpacing: '0.06em',
          marginBottom: 10,
        }}
      >
        Add a compliance check
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 6 }}>
        {CHECK_TYPES.map((ct) => {
          const Icon = ct.icon;
          const alreadyAdded = existingTypes.has(ct.type);
          return (
            <button
              key={ct.type}
              type="button"
              onClick={() => onAdd(ct.type)}
              style={{
                display: 'flex',
                alignItems: 'flex-start',
                gap: 8,
                padding: '10px',
                borderRadius: 7,
                border: '1px solid var(--border)',
                background: alreadyAdded
                  ? `color-mix(in srgb, ${ct.color} 5%, var(--bg-card))`
                  : 'var(--bg-card)',
                cursor: 'pointer',
                textAlign: 'left',
                transition: 'all 0.15s',
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.borderColor = ct.color;
                e.currentTarget.style.background = `color-mix(in srgb, ${ct.color} 8%, var(--bg-card))`;
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.borderColor = 'var(--border)';
                e.currentTarget.style.background = alreadyAdded
                  ? `color-mix(in srgb, ${ct.color} 5%, var(--bg-card))`
                  : 'var(--bg-card)';
              }}
            >
              <div
                style={{
                  width: 28,
                  height: 28,
                  borderRadius: 6,
                  flexShrink: 0,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  background: `color-mix(in srgb, ${ct.color} 12%, transparent)`,
                }}
              >
                <Icon size={14} style={{ color: ct.color }} />
              </div>
              <div style={{ minWidth: 0 }}>
                <div
                  style={{
                    fontSize: 11,
                    fontWeight: 600,
                    color: 'var(--text-primary)',
                    marginBottom: 2,
                    display: 'flex',
                    alignItems: 'center',
                    gap: 6,
                  }}
                >
                  {ct.name}
                  {alreadyAdded && (
                    <span
                      style={{
                        fontSize: 9,
                        fontFamily: 'var(--font-mono)',
                        fontWeight: 600,
                        color: ct.color,
                        background: `color-mix(in srgb, ${ct.color} 12%, transparent)`,
                        padding: '1px 6px',
                        borderRadius: 3,
                        textTransform: 'uppercase',
                        letterSpacing: '0.04em',
                      }}
                    >
                      Added
                    </span>
                  )}
                </div>
                <div style={{ fontSize: 10, color: 'var(--text-muted)', lineHeight: 1.3 }}>
                  {ct.description}
                </div>
              </div>
            </button>
          );
        })}
      </div>

      {/* Custom blank control option */}
      <button
        type="button"
        onClick={() => onAdd('sla')}
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          gap: 6,
          width: '100%',
          marginTop: 8,
          padding: '8px',
          borderRadius: 7,
          border: '1px dashed var(--border)',
          background: 'transparent',
          cursor: 'pointer',
          fontSize: 11,
          fontFamily: 'var(--font-mono)',
          color: 'var(--text-muted)',
          transition: 'all 0.15s',
        }}
        onMouseEnter={(e) => {
          e.currentTarget.style.borderColor = 'var(--text-secondary)';
          e.currentTarget.style.color = 'var(--text-secondary)';
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.borderColor = 'var(--border)';
          e.currentTarget.style.color = 'var(--text-muted)';
        }}
      >
        <Plus size={12} />
        Add blank control
      </button>
    </div>
  );
}

// ---------------------------------------------------------------
// Preview Panel
// ---------------------------------------------------------------

function FrameworkPreview({
  name,
  version,
  scoringMethod,
  controls,
}: {
  name: string;
  version: string;
  scoringMethod: string;
  controls: EditableControl[];
}) {
  const validControls = controls.filter((c) => c.control_id.trim() && c.name.trim());
  const checkTypeCounts = new Map<string, number>();
  for (const c of validControls) {
    const ct = c.check_type || 'sla';
    checkTypeCounts.set(ct, (checkTypeCounts.get(ct) ?? 0) + 1);
  }

  const totalCheckTypes = CHECK_TYPES.length;
  const coveredTypes = checkTypeCounts.size;
  const coveragePct = totalCheckTypes > 0 ? Math.round((coveredTypes / totalCheckTypes) * 100) : 0;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
      {/* Mini framework card */}
      <div>
        <div style={sectionLabel}>Card Preview</div>
        <div
          style={{
            border: '1px solid var(--border)',
            borderRadius: 8,
            padding: 12,
            background: 'var(--bg-card)',
          }}
        >
          <div
            style={{
              fontSize: 13,
              fontWeight: 600,
              fontFamily: 'var(--font-display)',
              color: 'var(--text-primary)',
              marginBottom: 4,
            }}
          >
            {name || 'Untitled Framework'}
          </div>
          <div
            style={{
              display: 'flex',
              gap: 8,
              alignItems: 'center',
              fontSize: 10,
              fontFamily: 'var(--font-mono)',
              color: 'var(--text-muted)',
            }}
          >
            {version && <span>v{version}</span>}
            <span>{validControls.length} controls</span>
            <span>{scoringMethod.replace('_', ' ')}</span>
          </div>
        </div>
      </div>

      {/* Coverage gauge */}
      <div>
        <div style={sectionLabel}>Check Coverage</div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 8 }}>
          <div style={{ position: 'relative', width: 44, height: 44 }}>
            <svg width={44} height={44} viewBox="0 0 44 44">
              <circle cx={22} cy={22} r={18} fill="none" stroke="var(--border)" strokeWidth={4} />
              <circle
                cx={22}
                cy={22}
                r={18}
                fill="none"
                stroke={
                  coveragePct >= 75
                    ? 'var(--signal-healthy)'
                    : coveragePct >= 40
                      ? 'var(--signal-warning)'
                      : 'var(--text-muted)'
                }
                strokeWidth={4}
                strokeDasharray={`${(coveragePct / 100) * 113.1} 113.1`}
                strokeLinecap="round"
                transform="rotate(-90 22 22)"
              />
            </svg>
            <span
              style={{
                position: 'absolute',
                inset: 0,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: 10,
                fontFamily: 'var(--font-mono)',
                fontWeight: 700,
                color: 'var(--text-primary)',
              }}
            >
              {coveredTypes}/{totalCheckTypes}
            </span>
          </div>
          <div
            style={{
              fontSize: 10,
              fontFamily: 'var(--font-mono)',
              color: 'var(--text-muted)',
              lineHeight: 1.4,
            }}
          >
            {coveredTypes} of {totalCheckTypes} available
            <br />
            check types configured
          </div>
        </div>

        {/* Check type dots */}
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
          {CHECK_TYPES.map((ct) => {
            const active = checkTypeCounts.has(ct.type);
            return (
              <div
                key={ct.type}
                title={`${ct.name}${active ? ` (${checkTypeCounts.get(ct.type)})` : ''}`}
                style={{
                  width: 22,
                  height: 22,
                  borderRadius: 4,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  background: active
                    ? `color-mix(in srgb, ${ct.color} 15%, transparent)`
                    : 'var(--bg-card)',
                  border: `1px solid ${active ? ct.color : 'var(--border)'}`,
                }}
              >
                <ct.icon
                  size={11}
                  style={{
                    color: active ? ct.color : 'var(--text-muted)',
                    opacity: active ? 1 : 0.3,
                  }}
                />
              </div>
            );
          })}
        </div>
      </div>

      {/* Controls list */}
      {validControls.length > 0 && (
        <div>
          <div style={sectionLabel}>Controls ({validControls.length})</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
            {validControls.map((c) => {
              const ci = CHECK_TYPES.find((ct) => ct.type === (c.check_type || 'sla'));
              return (
                <div
                  key={c._key}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 5,
                    fontSize: 10,
                    fontFamily: 'var(--font-mono)',
                    padding: '2px 0',
                  }}
                >
                  <span
                    style={{
                      width: 6,
                      height: 6,
                      borderRadius: '50%',
                      flexShrink: 0,
                      background: ci?.color ?? 'var(--accent)',
                    }}
                  />
                  <span
                    style={{
                      color: 'var(--text-muted)',
                      width: 56,
                      flexShrink: 0,
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                      whiteSpace: 'nowrap',
                    }}
                  >
                    {c.control_id}
                  </span>
                  <span
                    style={{
                      color: 'var(--text-secondary)',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                      whiteSpace: 'nowrap',
                    }}
                  >
                    {c.name}
                  </span>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Readiness */}
      <div>
        <div style={sectionLabel}>Readiness</div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
          {[
            { label: 'Framework name set', ok: !!name.trim() },
            { label: 'Has controls', ok: validControls.length > 0 },
            {
              label: 'All controls complete',
              ok: controls.length > 0 && validControls.length === controls.length,
            },
          ].map(({ label, ok }) => (
            <div
              key={label}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 6,
                fontSize: 10,
                fontFamily: 'var(--font-mono)',
              }}
            >
              <span style={{ color: ok ? 'var(--signal-healthy)' : 'var(--text-muted)' }}>
                {ok ? '\u2713' : '\u25CB'}
              </span>
              <span style={{ color: ok ? 'var(--text-secondary)' : 'var(--text-muted)' }}>
                {label}
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

const sectionLabel: React.CSSProperties = {
  fontSize: 9,
  fontFamily: 'var(--font-mono)',
  color: 'var(--text-muted)',
  textTransform: 'uppercase',
  letterSpacing: '0.08em',
  marginBottom: 8,
};

// ---------------------------------------------------------------
// Main Component
// ---------------------------------------------------------------

export function CustomFrameworkDialog({ open, onOpenChange, existing }: Props) {
  const can = useCan();
  const isEdit = !!existing;
  const createFw = useCreateCustomFramework();
  const updateFw = useUpdateCustomFramework();
  const deleteFw = useDeleteCustomFramework();
  const updateControls = useUpdateCustomControls();

  const { data: fetchedControls, isLoading: controlsLoading } = useCustomFrameworkControls(
    existing?.id ?? '',
  );

  const [name, setName] = useState('');
  const [version, setVersion] = useState('1.0');
  const [description, setDescription] = useState('');
  const [scoringMethod, setScoringMethod] = useState<
    'average' | 'strictest' | 'worst_case' | 'weighted'
  >('average');
  const [controls, setControls] = useState<EditableControl[]>([]);
  const [deleteConfirm, setDeleteConfirm] = useState(false);
  const [controlsPopulated, setControlsPopulated] = useState(false);

  // Populate metadata when editing
  useEffect(() => {
    if (existing) {
      setName(existing.name);
      setVersion(existing.version ?? '1.0');
      setDescription(existing.description ?? '');
      setScoringMethod(
        (existing.scoring_method as 'average' | 'strictest' | 'worst_case' | 'weighted') ??
          'average',
      );
      setControlsPopulated(false);
    } else {
      setName('');
      setVersion('1.0');
      setDescription('');
      setScoringMethod('average');
      setControls([]);
      setControlsPopulated(false);
    }
    setDeleteConfirm(false);
  }, [existing, open]);

  // Populate controls from fetched data when editing
  useEffect(() => {
    if (!existing || controlsPopulated) return;
    if (controlsLoading) return;

    const sourceControls = fetchedControls ?? existing.controls ?? [];
    const editableControls: EditableControl[] = sourceControls.map((c) => ({
      _key: c.id ?? makeKey(),
      control_id: c.control_id,
      name: c.name,
      description: c.description ?? '',
      category: c.category ?? 'General',
      check_type: c.check_type || 'sla',
      remediation_hint: c.remediation_hint ?? '',
      sla_tiers:
        c.sla_tiers && c.sla_tiers.length > 0
          ? c.sla_tiers
          : DEFAULT_SLA_TIERS.map((t) => ({ ...t })),
      check_config: c.check_config ?? buildDefaultCheckConfig(c.check_type || 'sla'),
    }));
    setControls(editableControls);
    setControlsPopulated(true);
  }, [existing, fetchedControls, controlsLoading, controlsPopulated]);

  function addControlFromTemplate(checkType: string) {
    // Auto-increment ID if same check type already exists
    const existing = controls.filter((c) => c.check_type === checkType);
    const ctrl = makeControlFromTemplate(checkType);
    if (existing.length > 0) {
      const num = existing.length + 1;
      ctrl.control_id = ctrl.control_id.replace(/\d+$/, String(num).padStart(3, '0'));
    }
    setControls((prev) => [...prev, ctrl]);
  }

  function removeControl(key: string) {
    setControls((prev) => prev.filter((c) => c._key !== key));
  }

  function updateControl(key: string, updated: EditableControl) {
    setControls((prev) => prev.map((c) => (c._key === key ? updated : c)));
  }

  const isBusy =
    createFw.isPending || updateFw.isPending || deleteFw.isPending || updateControls.isPending;
  const validControlCount = controls.filter((c) => c.control_id.trim() && c.name.trim()).length;

  async function handleSubmit() {
    if (!name.trim()) {
      toast.error('Framework name is required');
      return;
    }

    const validControls = controls.filter((c) => c.control_id.trim() && c.name.trim());
    const controlsPayload: CustomControlInput[] = validControls.map((ctrl) => {
      const { _key: _, ...rest } = ctrl;
      void _;
      return rest;
    });

    try {
      if (isEdit && existing) {
        await updateFw.mutateAsync({
          id: existing.id,
          name: name.trim(),
          version: version.trim() || '1.0',
          description: description.trim(),
          scoring_method: scoringMethod,
        });
        await updateControls.mutateAsync({ id: existing.id, controls: controlsPayload });
        toast.success(`Framework "${name}" updated`);
      } else {
        await createFw.mutateAsync({
          name: name.trim(),
          version: version.trim() || '1.0',
          description: description.trim(),
          scoring_method: scoringMethod,
          controls: controlsPayload,
        });
        toast.success(`Framework "${name}" created`);
      }
      onOpenChange(false);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to save framework');
    }
  }

  async function handleDelete() {
    if (!existing) return;
    if (!deleteConfirm) {
      setDeleteConfirm(true);
      return;
    }
    try {
      await deleteFw.mutateAsync(existing.id);
      toast.success(`Framework "${existing.name}" deleted`);
      onOpenChange(false);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to delete framework');
    }
  }

  const existingCheckTypes = new Set(controls.map((c) => c.check_type || 'sla'));

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side="right"
        style={{ width: 820, maxWidth: 820, padding: 0, display: 'flex', flexDirection: 'column' }}
        showCloseButton
      >
        <SheetHeader style={{ padding: '16px 16px 0' }}>
          <SheetTitle style={{ fontFamily: 'var(--font-display)', fontSize: 15 }}>
            {isEdit ? `Edit: ${existing?.name}` : 'New Compliance Framework'}
          </SheetTitle>
          <SheetDescription style={{ fontSize: 11 }}>
            {isEdit
              ? 'Modify framework configuration and controls'
              : 'Pick the compliance checks you need, then customize the details'}
          </SheetDescription>
        </SheetHeader>

        {/* Main content: form + preview */}
        <div style={{ flex: 1, overflowY: 'auto', display: 'flex', minHeight: 0 }}>
          {/* LEFT: Form */}
          <div
            style={{
              flex: 1,
              minWidth: 0,
              padding: '16px',
              display: 'flex',
              flexDirection: 'column',
              gap: 16,
              overflowY: 'auto',
            }}
          >
            {/* Step 1: Framework identity */}
            <div
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 10,
                fontWeight: 600,
                textTransform: 'uppercase',
                letterSpacing: '0.06em',
                color: 'var(--text-muted)',
                display: 'flex',
                alignItems: 'center',
                gap: 6,
              }}
            >
              <span
                style={{
                  width: 18,
                  height: 18,
                  borderRadius: '50%',
                  background: 'var(--accent)',
                  color: 'var(--btn-accent-text, #000)',
                  display: 'inline-flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: 10,
                  fontWeight: 700,
                }}
              >
                1
              </span>
              Framework Details
            </div>
            <div
              style={{
                background: 'var(--bg-inset)',
                border: '1px solid var(--border)',
                borderRadius: 8,
                padding: 14,
                display: 'flex',
                flexDirection: 'column',
                gap: 12,
              }}
            >
              <div style={{ display: 'flex', gap: 12 }}>
                <div style={{ flex: 1 }}>
                  <label style={labelStyle}>Framework Name *</label>
                  <Input
                    placeholder="e.g. Internal Security Standard"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    style={{ fontSize: 13, width: '100%' }}
                    autoFocus
                  />
                </div>
                <div style={{ width: 80 }}>
                  <label style={labelStyle}>Version</label>
                  <Input
                    placeholder="1.0"
                    value={version}
                    onChange={(e) => setVersion(e.target.value)}
                    style={{ fontSize: 13, width: '100%' }}
                  />
                </div>
              </div>
              <div>
                <label style={labelStyle}>Description</label>
                <Input
                  placeholder="What this framework covers"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  style={{ fontSize: 13, width: '100%' }}
                />
              </div>
              <div>
                <label style={labelStyle}>Scoring Method</label>
                <div style={{ display: 'flex', gap: 6 }}>
                  {(
                    [
                      { key: 'average', label: 'Average', desc: 'Mean of all endpoint scores' },
                      {
                        key: 'strictest',
                        label: 'Strictest',
                        desc: 'Lowest individual endpoint score',
                      },
                      {
                        key: 'worst_case',
                        label: 'Worst case',
                        desc: 'Fails if any endpoint fails',
                      },
                      { key: 'weighted', label: 'Weighted', desc: 'Score weighted by severity' },
                    ] as const
                  ).map((m) => (
                    <button
                      key={m.key}
                      type="button"
                      title={m.desc}
                      onClick={() => setScoringMethod(m.key)}
                      style={{
                        fontFamily: 'var(--font-mono)',
                        fontSize: 11,
                        padding: '5px 12px',
                        borderRadius: 6,
                        border: '1px solid',
                        borderColor: scoringMethod === m.key ? 'var(--accent)' : 'var(--border)',
                        background:
                          scoringMethod === m.key
                            ? 'color-mix(in srgb, var(--accent) 10%, transparent)'
                            : 'transparent',
                        color: scoringMethod === m.key ? 'var(--accent)' : 'var(--text-secondary)',
                        cursor: 'pointer',
                        transition: 'all 0.15s',
                      }}
                    >
                      {m.label}
                    </button>
                  ))}
                </div>
              </div>
            </div>

            {/* Step 2: Add controls */}
            <div
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 10,
                fontWeight: 600,
                textTransform: 'uppercase',
                letterSpacing: '0.06em',
                color: 'var(--text-muted)',
                display: 'flex',
                alignItems: 'center',
                gap: 6,
              }}
            >
              <span
                style={{
                  width: 18,
                  height: 18,
                  borderRadius: '50%',
                  background: controls.length > 0 ? 'var(--accent)' : 'var(--bg-inset)',
                  color: controls.length > 0 ? 'var(--btn-accent-text, #000)' : 'var(--text-muted)',
                  border: controls.length > 0 ? 'none' : '1px solid var(--border)',
                  display: 'inline-flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: 10,
                  fontWeight: 700,
                }}
              >
                2
              </span>
              Add Controls
            </div>

            {controls.length > 0 && (
              <div>
                <div
                  style={{
                    fontSize: 12,
                    fontWeight: 600,
                    color: 'var(--text-primary)',
                    marginBottom: 8,
                    display: 'flex',
                    alignItems: 'center',
                    gap: 8,
                  }}
                >
                  Active Controls
                  <span
                    style={{
                      fontSize: 10,
                      fontFamily: 'var(--font-mono)',
                      color: 'var(--text-muted)',
                      fontWeight: 400,
                    }}
                  >
                    {validControlCount} of {controls.length} valid
                  </span>
                </div>

                {isEdit && controlsLoading ? (
                  <>
                    {[1, 2, 3].map((i) => (
                      <div
                        key={i}
                        style={{
                          border: '1px solid var(--border)',
                          borderRadius: 8,
                          padding: 12,
                          marginBottom: 8,
                        }}
                      >
                        <div style={{ display: 'flex', gap: 8 }}>
                          <Skeleton className="h-8" style={{ width: 130 }} />
                          <Skeleton className="h-8" style={{ flex: 1 }} />
                        </div>
                      </div>
                    ))}
                  </>
                ) : (
                  controls.map((ctrl) => (
                    <ControlCard
                      key={ctrl._key}
                      control={ctrl}
                      onChange={(updated) => updateControl(ctrl._key, updated)}
                      onRemove={() => removeControl(ctrl._key)}
                    />
                  ))
                )}
              </div>
            )}

            {/* Template picker */}
            <TemplatePicker existingTypes={existingCheckTypes} onAdd={addControlFromTemplate} />
          </div>

          {/* RIGHT: Preview */}
          <div
            style={{
              width: 250,
              flexShrink: 0,
              padding: '16px 14px',
              borderLeft: '1px solid var(--border)',
              background: 'var(--bg-inset)',
              overflowY: 'auto',
            }}
          >
            <FrameworkPreview
              name={name}
              version={version}
              scoringMethod={scoringMethod}
              controls={controls}
            />
          </div>
        </div>

        {/* Footer */}
        <div
          style={{
            padding: '12px 16px',
            borderTop: '1px solid var(--border)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
          }}
        >
          {isEdit ? (
            deleteConfirm ? (
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <span
                  style={{
                    fontFamily: 'var(--font-sans)',
                    fontSize: 12,
                    color: 'var(--signal-critical)',
                  }}
                >
                  Delete this framework?
                </span>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={isBusy || !can('compliance', 'delete')}
                  title={!can('compliance', 'delete') ? "You don't have permission" : undefined}
                  onClick={handleDelete}
                  style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: 11,
                    borderColor: 'var(--signal-critical)',
                    color: 'var(--signal-critical)',
                  }}
                >
                  Yes, Delete
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setDeleteConfirm(false)}
                  style={{ fontFamily: 'var(--font-mono)', fontSize: 11 }}
                >
                  Cancel
                </Button>
              </div>
            ) : (
              <Button
                variant="outline"
                size="sm"
                disabled={isBusy || !can('compliance', 'delete')}
                title={!can('compliance', 'delete') ? "You don't have permission" : undefined}
                onClick={() => setDeleteConfirm(true)}
                style={{ fontFamily: 'var(--font-mono)', fontSize: 11 }}
              >
                <Trash2 style={{ width: 12, height: 12, marginRight: 5 }} />
                Delete
              </Button>
            )
          ) : (
            <div />
          )}
          <div style={{ display: 'flex', gap: 8 }}>
            <Button
              variant="outline"
              size="sm"
              onClick={() => onOpenChange(false)}
              style={{ fontFamily: 'var(--font-mono)', fontSize: 11 }}
            >
              Cancel
            </Button>
            <Button
              size="sm"
              disabled={
                isBusy ||
                !name.trim() ||
                validControlCount === 0 ||
                (isEdit ? !can('compliance', 'update') : !can('compliance', 'create'))
              }
              title={
                (isEdit && !can('compliance', 'update')) ||
                (!isEdit && !can('compliance', 'create'))
                  ? "You don't have permission"
                  : undefined
              }
              onClick={handleSubmit}
              style={{ fontFamily: 'var(--font-mono)', fontSize: 11 }}
            >
              {isBusy ? 'Saving...' : isEdit ? 'Save Changes' : 'Create Framework'}
            </Button>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
}
