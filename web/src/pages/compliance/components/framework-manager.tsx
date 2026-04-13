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
  Shield,
  ShieldCheck,
  ChevronLeft,
  Plus,
  Trash2,
  ChevronDown,
  ChevronUp,
  Pencil,
  ToggleLeft,
  ToggleRight,
  BookOpen,
  MonitorCheck,
  Package,
  ScanSearch,
  AlertTriangle,
  Rocket,
  Activity,
  Bug,
} from 'lucide-react';
import { useCan } from '../../../app/auth/AuthContext';
import {
  useComplianceFrameworks,
  useComplianceSummary,
  useCustomFrameworks,
  useEnableFramework,
  useDisableFramework,
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
// Shared types & constants (from custom-framework-dialog)
// ---------------------------------------------------------------

interface EditableControl extends CustomControlInput {
  _key: string;
  check_config?: CheckConfig;
}

interface ConditionDef {
  key: string;
  name: string;
  description: string;
  type: 'boolean' | 'threshold' | 'duration';
  default: number;
  unit: string;
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
    conditions: [],
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
        description: 'Endpoint is registered and active',
        type: 'boolean',
        default: 1,
        unit: '',
        min: 0,
        max: 1,
      },
      {
        key: 'has_hardware_data',
        name: 'Has Hardware Data',
        description: 'Endpoint has reported CPU, memory, and disk',
        type: 'boolean',
        default: 1,
        unit: '',
        min: 0,
        max: 1,
      },
      {
        key: 'heartbeat_freshness',
        name: 'Heartbeat Freshness',
        description: 'Agent has reported within this window',
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
        description: 'No CISA Known Exploited Vulnerabilities present',
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
        description: 'Agent must have reported within this window',
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
        description: 'Also check high-severity CVEs',
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

const CONTROL_TEMPLATES: Record<
  string,
  { id: string; name: string; category: string; description: string; hint: string }
> = {
  sla: {
    id: 'SLA-001',
    name: 'Patch SLA Compliance',
    category: 'Vulnerability Management',
    description: 'Evaluate patch remediation against CVSS-based SLA deadlines',
    hint: 'Deploy patches within defined SLA timelines',
  },
  asset_inventory: {
    id: 'ASSET-001',
    name: 'Asset Inventory Coverage',
    category: 'Asset Management',
    description: 'Verify all endpoints are enrolled with hardware data',
    hint: 'Ensure all endpoints have the agent installed and reporting',
  },
  software_inventory: {
    id: 'SW-001',
    name: 'Software Inventory Currency',
    category: 'Asset Management',
    description: 'Verify all endpoints have completed a package scan within 7 days',
    hint: 'Configure agent scan intervals to run at least weekly',
  },
  vuln_scanning: {
    id: 'VULN-001',
    name: 'Vulnerability Scan Coverage',
    category: 'Vulnerability Management',
    description: 'Verify all endpoints are covered by CVE scanning',
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
    hint: 'Remediate critical and high-severity CVEs within 30 days',
  },
};

const DEFAULT_SLA_TIERS: SLATierInput[] = [
  { label: 'critical', days: 15, cvss_min: 9.0, cvss_max: 10.0 },
  { label: 'high', days: 30, cvss_min: 7.0, cvss_max: 8.9 },
  { label: 'medium', days: 90, cvss_min: 4.0, cvss_max: 6.9 },
  { label: 'low', days: null, cvss_min: 0.0, cvss_max: 3.9 },
];

const FRAMEWORK_NAMES: Record<string, string> = {
  cis: 'CIS Controls v8',
  hipaa: 'HIPAA Security Rule',
  nist_800_53: 'NIST 800-53',
  pci_dss: 'PCI DSS v4.0',
  iso_27001: 'ISO 27001',
  soc_2: 'SOC 2 Type II',
};

const FRAMEWORK_SUBTITLES: Record<string, string> = {
  'CIS Controls v8': 'Center for Internet Security',
  'HIPAA Security Rule': 'Health Information Portability',
  'NIST 800-53': 'National Institute of Standards',
  'PCI DSS v4.0': 'Payment Card Industry Security',
  'ISO 27001': 'Information Security Management',
  'SOC 2 Type II': 'Service Organization Controls',
};

function normalizeFrameworkName(name: string): string {
  const lower = name.toLowerCase().replace(/[\s-]/g, '_');
  return FRAMEWORK_NAMES[name.toLowerCase()] ?? FRAMEWORK_NAMES[lower] ?? name;
}

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

// ---------------------------------------------------------------
// Styles
// ---------------------------------------------------------------

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
      conditions: { ...conditions, [key]: { ...current, enabled: !current.enabled } },
    });
  }

  function setConditionValue(key: string, value: number) {
    const current = conditions[key] ?? { enabled: true };
    onChange({
      ...config,
      conditions: { ...conditions, [key]: { ...current, value } },
    });
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
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
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ fontSize: 12, fontWeight: 500, color: 'var(--text-primary)' }}>
                    {cond.name}
                  </div>
                  <div style={{ fontSize: 10, color: 'var(--text-muted)', lineHeight: 1.3 }}>
                    {cond.description}
                  </div>
                </div>
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
// Control Card (for create/edit form)
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
        <span
          style={{
            fontSize: 11,
            fontFamily: 'var(--font-mono)',
            color: 'var(--text-muted)',
            flexShrink: 0,
          }}
        >
          {control.control_id || '\u2014'}
        </span>
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
                      placeholder="\u2014"
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
// Template Picker
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
// Framework List View
// ---------------------------------------------------------------

function FrameworkListView({
  onCreateCustom,
  onEditCustom,
}: {
  onCreateCustom: () => void;
  onEditCustom: (fw: CustomFrameworkResponse) => void;
}) {
  const can = useCan();
  const frameworksQuery = useComplianceFrameworks();
  const customQuery = useCustomFrameworks();
  const summaryQuery = useComplianceSummary();
  const enableFramework = useEnableFramework();
  const disableFramework = useDisableFramework();
  const deleteCustom = useDeleteCustomFramework();

  const [enabling, setEnabling] = useState<string | null>(null);
  const [disabling, setDisabling] = useState<string | null>(null);
  const [deleting, setDeleting] = useState<string | null>(null);
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);

  const enabledFrameworkIds = new Set([
    ...(frameworksQuery.data ?? []).filter((fw) => fw.enabled).map((fw) => fw.id),
    ...((summaryQuery.data as { frameworks?: { framework_id: string }[] })?.frameworks ?? []).map(
      (fw: { framework_id: string }) => fw.framework_id,
    ),
  ]);

  // The frameworks API returns both built-in and enabled custom frameworks.
  // Filter: built-in IDs are short strings (CIS, HIPAA, etc), custom are UUIDs.
  const allApiFrameworks = frameworksQuery.data ?? [];
  const customIds = new Set((customQuery.data ?? []).map((fw) => fw.id));
  const builtInFrameworks = allApiFrameworks.filter((fw) => !customIds.has(fw.id));
  const customFrameworks = customQuery.data ?? [];
  const isLoading = frameworksQuery.isLoading || customQuery.isLoading;

  // Build a config_id lookup that covers both built-in and custom frameworks
  const configIdMap = new Map<string, string>();
  for (const fw of allApiFrameworks) {
    if (fw.config_id) configIdMap.set(fw.id, fw.config_id);
  }

  async function handleEnable(id: string) {
    setEnabling(id);
    try {
      await enableFramework.mutateAsync({ framework_id: id, scoring_method: 'average' });
      toast.success('Framework enabled');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to enable');
    } finally {
      setEnabling(null);
    }
  }

  async function handleDisable(id: string) {
    // Look up config_id from the combined map (built-in + custom)
    const configId = configIdMap.get(id);
    if (!configId) {
      toast.error('Cannot disable: framework config not found');
      return;
    }
    setDisabling(id);
    try {
      await disableFramework.mutateAsync(configId);
      toast.success('Framework disabled');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to disable');
    } finally {
      setDisabling(null);
    }
  }

  async function handleDelete(id: string, name: string) {
    if (deleteConfirm !== id) {
      setDeleteConfirm(id);
      return;
    }
    setDeleting(id);
    try {
      await deleteCustom.mutateAsync(id);
      toast.success(`"${name}" deleted`);
      setDeleteConfirm(null);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to delete');
    } finally {
      setDeleting(null);
    }
  }

  if (isLoading) {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: 8, padding: '0 20px' }}>
        {Array.from({ length: 6 }).map((_, i) => (
          <Skeleton key={i} className="h-14 rounded-lg" />
        ))}
      </div>
    );
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 20, padding: '0 20px 20px' }}>
      {/* Built-in frameworks */}
      <div>
        <div
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 10,
            fontWeight: 600,
            textTransform: 'uppercase',
            letterSpacing: '0.08em',
            color: 'var(--text-muted)',
            marginBottom: 10,
            display: 'flex',
            alignItems: 'center',
            gap: 8,
          }}
        >
          <ShieldCheck size={13} style={{ opacity: 0.7 }} />
          Built-in Frameworks
          <span
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 10,
              background: 'var(--bg-inset)',
              border: '1px solid var(--border)',
              borderRadius: 4,
              padding: '1px 6px',
              color: 'var(--text-muted)',
            }}
          >
            {builtInFrameworks.length}
          </span>
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
          {builtInFrameworks.map((fw) => {
            const isEnabled = enabledFrameworkIds.has(fw.id);
            const isEnabling = enabling === fw.id;
            const isDisabling = disabling === fw.id;
            const displayName = normalizeFrameworkName(fw.name);
            const subtitle = FRAMEWORK_SUBTITLES[displayName] ?? '';

            return (
              <div
                key={fw.id}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 12,
                  padding: '12px 14px',
                  borderRadius: 8,
                  border: '1px solid var(--border)',
                  background: isEnabled ? 'var(--bg-card)' : 'var(--bg-inset)',
                  transition: 'all 0.15s',
                }}
              >
                {/* Icon */}
                <div
                  style={{
                    width: 36,
                    height: 36,
                    borderRadius: 8,
                    flexShrink: 0,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    background: isEnabled
                      ? 'color-mix(in srgb, var(--accent) 10%, transparent)'
                      : 'var(--bg-card)',
                    border: `1px solid ${isEnabled ? 'color-mix(in srgb, var(--accent) 25%, transparent)' : 'var(--border)'}`,
                  }}
                >
                  <Shield
                    size={16}
                    style={{ color: isEnabled ? 'var(--accent)' : 'var(--text-muted)' }}
                  />
                </div>

                {/* Info */}
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div
                    style={{
                      fontFamily: 'var(--font-sans)',
                      fontSize: 13,
                      fontWeight: 600,
                      color: isEnabled ? 'var(--text-primary)' : 'var(--text-secondary)',
                    }}
                  >
                    {displayName}
                  </div>
                  <div style={{ display: 'flex', gap: 8, marginTop: 2 }}>
                    {subtitle && (
                      <span
                        style={{
                          fontFamily: 'var(--font-mono)',
                          fontSize: 10,
                          color: 'var(--text-muted)',
                        }}
                      >
                        {subtitle}
                      </span>
                    )}
                    {fw.version && (
                      <span
                        style={{
                          fontFamily: 'var(--font-mono)',
                          fontSize: 10,
                          color: 'var(--text-muted)',
                        }}
                      >
                        v{fw.version}
                      </span>
                    )}
                  </div>
                </div>

                {/* Industries */}
                {(fw.applicable_industries ?? []).length > 0 && (
                  <div style={{ display: 'flex', gap: 3, flexShrink: 0 }}>
                    {(fw.applicable_industries ?? []).slice(0, 3).map((ind) => (
                      <span
                        key={ind}
                        style={{
                          fontFamily: 'var(--font-mono)',
                          fontSize: 9,
                          color: 'var(--text-muted)',
                          background: 'var(--bg-card)',
                          border: '1px solid var(--border)',
                          borderRadius: 3,
                          padding: '1px 5px',
                          textTransform: 'uppercase',
                          letterSpacing: '0.04em',
                        }}
                      >
                        {ind}
                      </span>
                    ))}
                  </div>
                )}

                {/* Toggle */}
                <button
                  type="button"
                  disabled={
                    isEnabling ||
                    isDisabling ||
                    (isEnabled ? !can('compliance', 'update') : !can('compliance', 'create'))
                  }
                  title={
                    (!isEnabled && !can('compliance', 'create')) ||
                    (isEnabled && !can('compliance', 'update'))
                      ? "You don't have permission"
                      : undefined
                  }
                  onClick={() => (isEnabled ? handleDisable(fw.id) : handleEnable(fw.id))}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 5,
                    padding: '5px 10px',
                    borderRadius: 6,
                    flexShrink: 0,
                    border: '1px solid',
                    borderColor: isEnabled ? 'var(--accent)' : 'var(--border)',
                    background: isEnabled
                      ? 'color-mix(in srgb, var(--accent) 8%, transparent)'
                      : 'transparent',
                    color: isEnabled ? 'var(--accent)' : 'var(--text-muted)',
                    fontFamily: 'var(--font-mono)',
                    fontSize: 11,
                    fontWeight: 500,
                    cursor: isEnabling || isDisabling ? 'wait' : 'pointer',
                    transition: 'all 0.15s',
                    opacity: isEnabling || isDisabling ? 0.6 : 1,
                  }}
                >
                  {isEnabling || isDisabling ? (
                    <span>{isEnabling ? 'Enabling...' : 'Disabling...'}</span>
                  ) : isEnabled ? (
                    <>
                      <ToggleRight size={14} />
                      Enabled
                    </>
                  ) : (
                    <>
                      <ToggleLeft size={14} />
                      Disabled
                    </>
                  )}
                </button>
              </div>
            );
          })}
        </div>
      </div>

      {/* Custom frameworks */}
      <div>
        <div
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 10,
            fontWeight: 600,
            textTransform: 'uppercase',
            letterSpacing: '0.08em',
            color: 'var(--text-muted)',
            marginBottom: 10,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <BookOpen size={13} style={{ opacity: 0.7 }} />
            Custom Frameworks
            <span
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 10,
                background: 'var(--bg-inset)',
                border: '1px solid var(--border)',
                borderRadius: 4,
                padding: '1px 6px',
                color: 'var(--text-muted)',
              }}
            >
              {customFrameworks.length}
            </span>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={onCreateCustom}
            disabled={!can('compliance', 'create')}
            title={!can('compliance', 'create') ? "You don't have permission" : undefined}
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 10,
              padding: '3px 10px',
              height: 'auto',
            }}
          >
            <Plus size={11} style={{ marginRight: 4 }} />
            New
          </Button>
        </div>

        {customFrameworks.length === 0 ? (
          <div
            style={{
              padding: '24px 16px',
              borderRadius: 8,
              border: '1px dashed var(--border)',
              background: 'var(--bg-inset)',
              textAlign: 'center',
            }}
          >
            <BookOpen
              size={20}
              style={{ color: 'var(--text-muted)', margin: '0 auto 8px', opacity: 0.5 }}
            />
            <div
              style={{
                fontFamily: 'var(--font-sans)',
                fontSize: 12,
                color: 'var(--text-secondary)',
                marginBottom: 4,
              }}
            >
              No custom frameworks yet
            </div>
            <div
              style={{ fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--text-muted)' }}
            >
              Create one to define your own compliance checks
            </div>
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            {customFrameworks.map((fw) => {
              const controlCount = fw.control_count ?? fw.controls?.length ?? 0;
              const isEnabled = enabledFrameworkIds.has(fw.id);
              const isDeleting = deleting === fw.id;
              const isConfirmDelete = deleteConfirm === fw.id;

              return (
                <div
                  key={fw.id}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 12,
                    padding: '12px 14px',
                    borderRadius: 8,
                    border: '1px solid var(--border)',
                    background: isEnabled ? 'var(--bg-card)' : 'var(--bg-inset)',
                    transition: 'all 0.15s',
                  }}
                >
                  {/* Icon */}
                  <div
                    style={{
                      width: 36,
                      height: 36,
                      borderRadius: 8,
                      flexShrink: 0,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      background: isEnabled
                        ? 'color-mix(in srgb, var(--accent) 10%, transparent)'
                        : 'var(--bg-card)',
                      border: `1px solid ${isEnabled ? 'color-mix(in srgb, var(--accent) 25%, transparent)' : 'var(--border)'}`,
                    }}
                  >
                    <BookOpen
                      size={16}
                      style={{ color: isEnabled ? 'var(--accent)' : 'var(--text-muted)' }}
                    />
                  </div>

                  {/* Info */}
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                      <span
                        style={{
                          fontFamily: 'var(--font-sans)',
                          fontSize: 13,
                          fontWeight: 600,
                          color: isEnabled ? 'var(--text-primary)' : 'var(--text-secondary)',
                          overflow: 'hidden',
                          textOverflow: 'ellipsis',
                          whiteSpace: 'nowrap',
                        }}
                      >
                        {fw.name}
                      </span>
                      <span
                        style={{
                          fontFamily: 'var(--font-mono)',
                          fontSize: 9,
                          textTransform: 'uppercase',
                          letterSpacing: '0.06em',
                          background: 'var(--accent-subtle)',
                          color: 'var(--accent)',
                          border: '1px solid var(--accent)',
                          borderRadius: 3,
                          padding: '1px 5px',
                          flexShrink: 0,
                        }}
                      >
                        Custom
                      </span>
                    </div>
                    <div style={{ display: 'flex', gap: 8, marginTop: 2 }}>
                      <span
                        style={{
                          fontFamily: 'var(--font-mono)',
                          fontSize: 10,
                          color: 'var(--text-muted)',
                        }}
                      >
                        v{fw.version}
                      </span>
                      <span
                        style={{
                          fontFamily: 'var(--font-mono)',
                          fontSize: 10,
                          color: 'var(--text-muted)',
                        }}
                      >
                        {controlCount} controls
                      </span>
                    </div>
                  </div>

                  {/* Enable/Disable toggle */}
                  <button
                    type="button"
                    disabled={
                      enabling === fw.id ||
                      disabling === fw.id ||
                      (isEnabled ? !can('compliance', 'update') : !can('compliance', 'create'))
                    }
                    title={
                      (!isEnabled && !can('compliance', 'create')) ||
                      (isEnabled && !can('compliance', 'update'))
                        ? "You don't have permission"
                        : undefined
                    }
                    onClick={() => (isEnabled ? handleDisable(fw.id) : handleEnable(fw.id))}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 5,
                      padding: '5px 10px',
                      borderRadius: 6,
                      flexShrink: 0,
                      border: '1px solid',
                      borderColor: isEnabled ? 'var(--accent)' : 'var(--border)',
                      background: isEnabled
                        ? 'color-mix(in srgb, var(--accent) 8%, transparent)'
                        : 'transparent',
                      color: isEnabled ? 'var(--accent)' : 'var(--text-muted)',
                      fontFamily: 'var(--font-mono)',
                      fontSize: 11,
                      fontWeight: 500,
                      cursor: enabling === fw.id || disabling === fw.id ? 'wait' : 'pointer',
                      transition: 'all 0.15s',
                      opacity: enabling === fw.id || disabling === fw.id ? 0.6 : 1,
                    }}
                  >
                    {enabling === fw.id || disabling === fw.id ? (
                      <span>{enabling === fw.id ? 'Enabling...' : 'Disabling...'}</span>
                    ) : isEnabled ? (
                      <>
                        <ToggleRight size={14} />
                        Enabled
                      </>
                    ) : (
                      <>
                        <ToggleLeft size={14} />
                        Disabled
                      </>
                    )}
                  </button>

                  {/* Actions */}
                  <div style={{ display: 'flex', gap: 4, flexShrink: 0 }}>
                    <button
                      type="button"
                      onClick={() => onEditCustom(fw)}
                      disabled={!can('compliance', 'update')}
                      title={
                        !can('compliance', 'update')
                          ? "You don't have permission"
                          : 'Edit framework'
                      }
                      aria-label={`Edit ${fw.name}`}
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        width: 30,
                        height: 30,
                        borderRadius: 6,
                        border: '1px solid var(--border)',
                        background: 'transparent',
                        cursor: 'pointer',
                        color: 'var(--text-secondary)',
                        transition: 'all 0.15s',
                      }}
                      onMouseEnter={(e) => {
                        e.currentTarget.style.borderColor = 'var(--accent)';
                        e.currentTarget.style.color = 'var(--accent)';
                      }}
                      onMouseLeave={(e) => {
                        e.currentTarget.style.borderColor = 'var(--border)';
                        e.currentTarget.style.color = 'var(--text-secondary)';
                      }}
                    >
                      <Pencil size={13} />
                    </button>
                    {isConfirmDelete ? (
                      <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                        <button
                          type="button"
                          onClick={() => handleDelete(fw.id, fw.name)}
                          disabled={isDeleting || !can('compliance', 'delete')}
                          style={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: 4,
                            padding: '4px 8px',
                            borderRadius: 6,
                            border: '1px solid var(--signal-critical)',
                            background:
                              'color-mix(in srgb, var(--signal-critical) 8%, transparent)',
                            cursor: isDeleting ? 'wait' : 'pointer',
                            color: 'var(--signal-critical)',
                            fontFamily: 'var(--font-mono)',
                            fontSize: 10,
                            fontWeight: 600,
                          }}
                        >
                          {isDeleting ? 'Deleting...' : 'Confirm'}
                        </button>
                        <button
                          type="button"
                          onClick={() => setDeleteConfirm(null)}
                          style={{
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            width: 30,
                            height: 30,
                            borderRadius: 6,
                            border: '1px solid var(--border)',
                            background: 'transparent',
                            cursor: 'pointer',
                            color: 'var(--text-muted)',
                            fontFamily: 'var(--font-mono)',
                            fontSize: 10,
                          }}
                        >
                          Cancel
                        </button>
                      </div>
                    ) : (
                      <button
                        type="button"
                        onClick={() => handleDelete(fw.id, fw.name)}
                        disabled={!can('compliance', 'delete')}
                        title={
                          !can('compliance', 'delete')
                            ? "You don't have permission"
                            : 'Delete framework'
                        }
                        aria-label={`Delete ${fw.name}`}
                        style={{
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          width: 30,
                          height: 30,
                          borderRadius: 6,
                          border: '1px solid var(--border)',
                          background: 'transparent',
                          cursor: 'pointer',
                          color: 'var(--text-muted)',
                          transition: 'all 0.15s',
                        }}
                        onMouseEnter={(e) => {
                          e.currentTarget.style.borderColor = 'var(--signal-critical)';
                          e.currentTarget.style.color = 'var(--signal-critical)';
                        }}
                        onMouseLeave={(e) => {
                          e.currentTarget.style.borderColor = 'var(--border)';
                          e.currentTarget.style.color = 'var(--text-muted)';
                        }}
                      >
                        <Trash2 size={13} />
                      </button>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}

// ---------------------------------------------------------------
// Custom Framework Form View (create/edit)
// ---------------------------------------------------------------

function CustomFrameworkForm({
  existing,
  onBack,
}: {
  existing?: CustomFrameworkResponse;
  onBack: () => void;
}) {
  const can = useCan();
  const isEdit = !!existing;
  const createFw = useCreateCustomFramework();
  const updateFw = useUpdateCustomFramework();
  const updateControlsMutation = useUpdateCustomControls();
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
  const [controlsPopulated, setControlsPopulated] = useState(false);

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
  }, [existing]);

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
    const existingOfType = controls.filter((c) => c.check_type === checkType);
    const ctrl = makeControlFromTemplate(checkType);
    if (existingOfType.length > 0) {
      const num = existingOfType.length + 1;
      ctrl.control_id = ctrl.control_id.replace(/\d+$/, String(num).padStart(3, '0'));
    }
    setControls((prev) => [...prev, ctrl]);
  }

  const isBusy = createFw.isPending || updateFw.isPending || updateControlsMutation.isPending;
  const validControlCount = controls.filter((c) => c.control_id.trim() && c.name.trim()).length;
  const existingCheckTypes = new Set(controls.map((c) => c.check_type || 'sla'));

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
        await updateControlsMutation.mutateAsync({ id: existing.id, controls: controlsPayload });
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
      onBack();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to save framework');
    }
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* Sub-header with back button */}
      <div
        style={{
          padding: '12px 20px',
          borderBottom: '1px solid var(--border)',
          display: 'flex',
          alignItems: 'center',
          gap: 10,
          background: 'var(--bg-inset)',
        }}
      >
        <button
          type="button"
          onClick={onBack}
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            width: 28,
            height: 28,
            borderRadius: 6,
            border: '1px solid var(--border)',
            background: 'var(--bg-card)',
            cursor: 'pointer',
            color: 'var(--text-secondary)',
            transition: 'all 0.15s',
          }}
          onMouseEnter={(e) => {
            e.currentTarget.style.borderColor = 'var(--accent)';
            e.currentTarget.style.color = 'var(--accent)';
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.borderColor = 'var(--border)';
            e.currentTarget.style.color = 'var(--text-secondary)';
          }}
        >
          <ChevronLeft size={14} />
        </button>
        <div>
          <div
            style={{
              fontFamily: 'var(--font-sans)',
              fontSize: 13,
              fontWeight: 600,
              color: 'var(--text-primary)',
            }}
          >
            {isEdit ? `Edit: ${existing?.name}` : 'New Custom Framework'}
          </div>
          <div style={{ fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--text-muted)' }}>
            {isEdit
              ? 'Modify configuration and controls'
              : 'Define compliance checks and thresholds'}
          </div>
        </div>
      </div>

      {/* Scrollable form content */}
      <div
        style={{
          flex: 1,
          overflowY: 'auto',
          padding: '16px 20px',
          display: 'flex',
          flexDirection: 'column',
          gap: 16,
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
                  { key: 'worst_case', label: 'Worst case', desc: 'Fails if any endpoint fails' },
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

        {/* Step 2: Controls */}
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
          {validControlCount > 0 && (
            <span
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 10,
                color: 'var(--text-muted)',
                fontWeight: 400,
                marginLeft: 4,
              }}
            >
              {validControlCount} of {controls.length} valid
            </span>
          )}
        </div>

        {controls.length > 0 && (
          <div>
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
                  onChange={(updated) =>
                    setControls((prev) => prev.map((c) => (c._key === ctrl._key ? updated : c)))
                  }
                  onRemove={() => setControls((prev) => prev.filter((c) => c._key !== ctrl._key))}
                />
              ))
            )}
          </div>
        )}

        <TemplatePicker existingTypes={existingCheckTypes} onAdd={addControlFromTemplate} />
      </div>

      {/* Footer */}
      <div
        style={{
          padding: '12px 20px',
          borderTop: '1px solid var(--border)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          background: 'var(--bg-card)',
        }}
      >
        <Button
          variant="outline"
          size="sm"
          onClick={onBack}
          style={{ fontFamily: 'var(--font-mono)', fontSize: 11 }}
        >
          Cancel
        </Button>
        <div style={{ display: 'flex', gap: 8 }}>
          {validControlCount > 0 && (
            <span
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 10,
                color: 'var(--text-muted)',
                alignSelf: 'center',
              }}
            >
              {validControlCount} control{validControlCount !== 1 ? 's' : ''} ready
            </span>
          )}
          <Button
            size="sm"
            disabled={
              isBusy ||
              !name.trim() ||
              validControlCount === 0 ||
              (isEdit ? !can('compliance', 'update') : !can('compliance', 'create'))
            }
            title={
              (isEdit && !can('compliance', 'update')) || (!isEdit && !can('compliance', 'create'))
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
    </div>
  );
}

// ---------------------------------------------------------------
// Main Component: Framework Manager Panel
// ---------------------------------------------------------------

interface FrameworkManagerProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

type ManagerView =
  | { type: 'list' }
  | { type: 'create' }
  | { type: 'edit'; framework: CustomFrameworkResponse };

export function FrameworkManager({ open, onOpenChange }: FrameworkManagerProps) {
  const [view, setView] = useState<ManagerView>({ type: 'list' });

  // Reset to list view when panel opens
  useEffect(() => {
    if (open) setView({ type: 'list' });
  }, [open]);

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side="right"
        style={{
          width: view.type === 'list' ? 560 : 700,
          maxWidth: view.type === 'list' ? 560 : 700,
          padding: 0,
          display: 'flex',
          flexDirection: 'column',
          transition: 'width 0.2s ease',
        }}
        showCloseButton
      >
        {view.type === 'list' && (
          <>
            <SheetHeader style={{ padding: '20px 20px 16px' }}>
              <SheetTitle
                style={{
                  fontFamily: 'var(--font-display)',
                  fontSize: 16,
                  fontWeight: 700,
                  color: 'var(--text-emphasis)',
                }}
              >
                Framework Management
              </SheetTitle>
              <SheetDescription
                style={{
                  fontFamily: 'var(--font-sans)',
                  fontSize: 12,
                  color: 'var(--text-secondary)',
                }}
              >
                Enable built-in frameworks or create custom compliance checks
              </SheetDescription>
            </SheetHeader>
            <div style={{ flex: 1, overflowY: 'auto', minHeight: 0 }}>
              <FrameworkListView
                onCreateCustom={() => setView({ type: 'create' })}
                onEditCustom={(fw) => setView({ type: 'edit', framework: fw })}
              />
            </div>
          </>
        )}

        {view.type === 'create' && <CustomFrameworkForm onBack={() => setView({ type: 'list' })} />}

        {view.type === 'edit' && (
          <CustomFrameworkForm existing={view.framework} onBack={() => setView({ type: 'list' })} />
        )}
      </SheetContent>
    </Sheet>
  );
}
