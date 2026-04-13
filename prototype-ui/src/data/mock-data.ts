// ============================================================
// PatchIQ PM Dashboard — Comprehensive Mock Data
// All 12 widget datasets. No backend — everything hardcoded.
// ============================================================

// ============================================================
// Micro-viz types for StatCard
// ============================================================
export type MicroVizType =
  | 'ring-gauge'
  | 'severity-bars'
  | 'gradient-ring'
  | 'pulsing-dots'
  | 'countdown-text'
  | 'sparkline'
  | 'pipeline-dots'
  | 'pulsing-green-dot';

export interface StatCardData {
  id: string;
  icon: string;
  iconColor: string;
  value: string | number;
  valueColor?: string;
  label: string;
  trend: { value: string; positive: boolean };
  trendText: string;
  microViz: MicroVizType;
  microVizData?: Record<string, unknown>;
}

export const DASHBOARD_STATS: StatCardData[] = [
  {
    id: 'endpoints',
    icon: 'Monitor',
    iconColor: 'var(--color-primary)',
    value: 247,
    label: 'Endpoints Online',
    trend: { value: '2.3%', positive: true },
    trendText: 'vs last week',
    microViz: 'ring-gauge',
    microVizData: { current: 247, total: 256, offline: 34, pending: 18 },
  },
  {
    id: 'critical-patches',
    icon: 'AlertTriangle',
    iconColor: 'var(--color-danger)',
    value: 12,
    label: 'Critical Patches',
    trend: { value: '4', positive: false },
    trendText: 'new this week',
    microViz: 'severity-bars',
    microVizData: { critical: 12, high: 23, medium: 45, low: 18 },
  },
  {
    id: 'compliance',
    icon: 'CheckCircle',
    iconColor: 'var(--color-success)',
    value: '94.2%',
    label: 'Compliance Rate',
    trend: { value: '1.8%', positive: true },
    trendText: 'vs last scan',
    microViz: 'gradient-ring',
    microVizData: { percentage: 94, nist: 92, pci: 88, hipaa: 95 },
  },
  {
    id: 'deployments',
    icon: 'Rocket',
    iconColor: 'var(--color-cyan)',
    value: 3,
    label: 'Active Deployments',
    trend: { value: '1', positive: true },
    trendText: 'started today',
    microViz: 'pulsing-dots',
    microVizData: { statuses: ['running', 'running', 'pending'] },
  },
  {
    id: 'overdue-sla',
    icon: 'Clock',
    iconColor: 'var(--color-warning)',
    value: 5,
    valueColor: 'var(--color-warning)',
    label: 'Overdue SLA',
    trend: { value: '2', positive: false },
    trendText: 'since yesterday',
    microViz: 'countdown-text',
    microVizData: { oldestOverdue: '3d 14h overdue', patchRef: 'KB5034441', nextDue: '4h 12m' },
  },
  {
    id: 'failed-deploys',
    icon: 'XCircle',
    iconColor: 'var(--color-danger)',
    value: 2,
    label: 'Failed Deployments',
    trend: { value: '1', positive: true },
    trendText: 'fewer than last week',
    microViz: 'sparkline',
    microVizData: { points: [4, 3, 5, 2, 1, 3, 2], recentIds: ['DEP-0041', 'DEP-0040'] },
  },
  {
    id: 'workflows',
    icon: 'GitBranch',
    iconColor: 'var(--color-purple)',
    value: 4,
    label: 'Workflows Running',
    trend: { value: '1', positive: true },
    trendText: 'queued',
    microViz: 'pipeline-dots',
    microVizData: {
      stages: ['complete', 'complete', 'running', 'pending', 'pending'],
      queued: 1,
      failed: 1,
    },
  },
  {
    id: 'hub-sync',
    icon: 'RefreshCw',
    iconColor: 'var(--color-success)',
    value: 'Synced',
    label: 'Hub Sync',
    trend: { value: '100%', positive: true },
    trendText: 'healthy',
    microViz: 'pulsing-green-dot',
    microVizData: { lastSync: '2 min ago', patchesSynced: 156, endpointsSynced: 247 },
  },
];

export const ALERT_MESSAGES: string[] = [
  '5 patches overdue SLA',
  '2 deployments failed in the last 24h',
  '3 endpoints at critical risk',
];

// ============================================================
// Blast Radius Graph
// ============================================================
export interface BlastRadiusNode {
  id: string;
  label: string;
  sublabel: string;
  count: number;
  compliance: 'compliant' | 'at-risk' | 'non-compliant';
  severity: 'critical' | 'high' | 'medium' | 'low';
}

export interface BlastRadiusEdge {
  source: string;
  target: string;
  severity: 'critical' | 'high' | 'medium' | 'low';
}

export interface BlastRadiusCVE {
  id: string;
  label: string;
  severity: 'critical' | 'high' | 'medium';
}

export interface BlastRadiusData {
  cve: BlastRadiusCVE;
  nodes: BlastRadiusNode[];
  edges: BlastRadiusEdge[];
}

export const BLAST_RADIUS_CVES: BlastRadiusCVE[] = [
  { id: 'CVE-2024-21762', label: 'CVE-2024-21762', severity: 'critical' },
  { id: 'CVE-2024-38094', label: 'CVE-2024-38094', severity: 'critical' },
  { id: 'CVE-2024-49113', label: 'CVE-2024-49113', severity: 'high' },
];

export const BLAST_RADIUS_DATA: Record<string, BlastRadiusData> = {
  'CVE-2024-21762': {
    cve: { id: 'CVE-2024-21762', label: 'CVE-2024-21762', severity: 'critical' },
    nodes: [
      {
        id: 'cve-center',
        label: 'CVE-2024-21762',
        sublabel: 'FortiOS SSL-VPN RCE',
        count: 0,
        compliance: 'non-compliant',
        severity: 'critical',
      },
      {
        id: 'win-servers',
        label: 'Windows Servers',
        sublabel: '2019 / 2022',
        count: 34,
        compliance: 'non-compliant',
        severity: 'critical',
      },
      {
        id: 'linux-web',
        label: 'Linux Web Tier',
        sublabel: 'Ubuntu 22.04 LTS',
        count: 18,
        compliance: 'at-risk',
        severity: 'high',
      },
      {
        id: 'macos-dev',
        label: 'macOS Dev Fleet',
        sublabel: 'Sonoma 14.x',
        count: 12,
        compliance: 'at-risk',
        severity: 'high',
      },
      {
        id: 'db-cluster',
        label: 'Database Cluster',
        sublabel: 'PostgreSQL on RHEL',
        count: 8,
        compliance: 'non-compliant',
        severity: 'critical',
      },
      {
        id: 'infra-network',
        label: 'Network Infra',
        sublabel: 'FortiGate appliances',
        count: 6,
        compliance: 'non-compliant',
        severity: 'critical',
      },
      {
        id: 'k8s-nodes',
        label: 'Kubernetes Nodes',
        sublabel: 'EKS worker nodes',
        count: 24,
        compliance: 'at-risk',
        severity: 'high',
      },
      {
        id: 'ci-runners',
        label: 'CI/CD Runners',
        sublabel: 'GitHub Actions self-hosted',
        count: 10,
        compliance: 'compliant',
        severity: 'medium',
      },
      {
        id: 'vpn-gateways',
        label: 'VPN Gateways',
        sublabel: 'FortiClient endpoints',
        count: 15,
        compliance: 'non-compliant',
        severity: 'critical',
      },
    ],
    edges: [
      { source: 'cve-center', target: 'win-servers', severity: 'critical' },
      { source: 'cve-center', target: 'linux-web', severity: 'high' },
      { source: 'cve-center', target: 'macos-dev', severity: 'high' },
      { source: 'cve-center', target: 'db-cluster', severity: 'critical' },
      { source: 'cve-center', target: 'infra-network', severity: 'critical' },
      { source: 'cve-center', target: 'k8s-nodes', severity: 'high' },
      { source: 'cve-center', target: 'ci-runners', severity: 'medium' },
      { source: 'cve-center', target: 'vpn-gateways', severity: 'critical' },
    ],
  },

  'CVE-2024-38094': {
    cve: { id: 'CVE-2024-38094', label: 'CVE-2024-38094', severity: 'critical' },
    nodes: [
      {
        id: 'cve-center',
        label: 'CVE-2024-38094',
        sublabel: 'SharePoint RCE',
        count: 0,
        compliance: 'non-compliant',
        severity: 'critical',
      },
      {
        id: 'sharepoint-farm',
        label: 'SharePoint Farm',
        sublabel: 'SP 2019 on-prem',
        count: 4,
        compliance: 'non-compliant',
        severity: 'critical',
      },
      {
        id: 'win-workstations',
        label: 'Win Workstations',
        sublabel: 'Windows 11 22H2',
        count: 89,
        compliance: 'at-risk',
        severity: 'high',
      },
      {
        id: 'exchange-servers',
        label: 'Exchange Servers',
        sublabel: 'Exchange 2019 CU14',
        count: 3,
        compliance: 'non-compliant',
        severity: 'critical',
      },
      {
        id: 'ad-controllers',
        label: 'Active Directory',
        sublabel: 'Domain Controllers',
        count: 5,
        compliance: 'non-compliant',
        severity: 'critical',
      },
      {
        id: 'file-servers',
        label: 'File Servers',
        sublabel: 'Windows Server 2022',
        count: 12,
        compliance: 'at-risk',
        severity: 'high',
      },
      {
        id: 'remote-workers',
        label: 'Remote Workers',
        sublabel: 'VPN-connected laptops',
        count: 47,
        compliance: 'at-risk',
        severity: 'high',
      },
      {
        id: 'print-servers',
        label: 'Print Servers',
        sublabel: 'Windows Server 2019',
        count: 6,
        compliance: 'compliant',
        severity: 'medium',
      },
      {
        id: 'sql-servers',
        label: 'SQL Servers',
        sublabel: 'SQL Server 2022',
        count: 9,
        compliance: 'at-risk',
        severity: 'high',
      },
    ],
    edges: [
      { source: 'cve-center', target: 'sharepoint-farm', severity: 'critical' },
      { source: 'cve-center', target: 'win-workstations', severity: 'high' },
      { source: 'cve-center', target: 'exchange-servers', severity: 'critical' },
      { source: 'cve-center', target: 'ad-controllers', severity: 'critical' },
      { source: 'cve-center', target: 'file-servers', severity: 'high' },
      { source: 'cve-center', target: 'remote-workers', severity: 'high' },
      { source: 'cve-center', target: 'print-servers', severity: 'medium' },
      { source: 'cve-center', target: 'sql-servers', severity: 'high' },
    ],
  },

  'CVE-2024-49113': {
    cve: { id: 'CVE-2024-49113', label: 'CVE-2024-49113', severity: 'high' },
    nodes: [
      {
        id: 'cve-center',
        label: 'CVE-2024-49113',
        sublabel: 'Windows LDAP DoS',
        count: 0,
        compliance: 'at-risk',
        severity: 'high',
      },
      {
        id: 'ldap-servers',
        label: 'LDAP Servers',
        sublabel: 'OpenLDAP 2.6.x',
        count: 7,
        compliance: 'non-compliant',
        severity: 'high',
      },
      {
        id: 'win-dc',
        label: 'Windows DCs',
        sublabel: 'Server 2022 DCs',
        count: 4,
        compliance: 'at-risk',
        severity: 'high',
      },
      {
        id: 'linux-auth',
        label: 'Linux Auth Servers',
        sublabel: 'SSSD + Kerberos',
        count: 11,
        compliance: 'at-risk',
        severity: 'high',
      },
      {
        id: 'radius-servers',
        label: 'RADIUS Servers',
        sublabel: 'FreeRADIUS 3.2.x',
        count: 3,
        compliance: 'compliant',
        severity: 'medium',
      },
      {
        id: 'vpn-concentrators',
        label: 'VPN Concentrators',
        sublabel: 'Cisco ASA 9.x',
        count: 5,
        compliance: 'at-risk',
        severity: 'high',
      },
      {
        id: 'jump-boxes',
        label: 'Jump Boxes',
        sublabel: 'Bastion hosts',
        count: 8,
        compliance: 'compliant',
        severity: 'medium',
      },
      {
        id: 'sso-providers',
        label: 'SSO Providers',
        sublabel: 'Okta LDAP agents',
        count: 2,
        compliance: 'at-risk',
        severity: 'high',
      },
    ],
    edges: [
      { source: 'cve-center', target: 'ldap-servers', severity: 'high' },
      { source: 'cve-center', target: 'win-dc', severity: 'high' },
      { source: 'cve-center', target: 'linux-auth', severity: 'high' },
      { source: 'cve-center', target: 'radius-servers', severity: 'medium' },
      { source: 'cve-center', target: 'vpn-concentrators', severity: 'high' },
      { source: 'cve-center', target: 'jump-boxes', severity: 'medium' },
      { source: 'cve-center', target: 'sso-providers', severity: 'high' },
    ],
  },
};

// ============================================================
// Risk Delta Projection
// ============================================================
export const RISK_DELTA_DATA: {
  day: number;
  current: number;
  deployAll: number;
  doNothing: number;
}[] = [
  { day: 1, current: 45, deployAll: 45, doNothing: 45 },
  { day: 2, current: 46, deployAll: 42, doNothing: 47 },
  { day: 3, current: 47, deployAll: 39, doNothing: 49 },
  { day: 4, current: 46, deployAll: 36, doNothing: 52 },
  { day: 5, current: 48, deployAll: 33, doNothing: 55 },
  { day: 6, current: 49, deployAll: 30, doNothing: 57 },
  { day: 7, current: 47, deployAll: 27, doNothing: 59 },
  { day: 8, current: 50, deployAll: 25, doNothing: 61 },
  { day: 9, current: 51, deployAll: 23, doNothing: 63 },
  { day: 10, current: 49, deployAll: 21, doNothing: 65 },
  { day: 11, current: 52, deployAll: 20, doNothing: 67 },
  { day: 12, current: 51, deployAll: 19, doNothing: 68 },
  { day: 13, current: 53, deployAll: 18, doNothing: 70 },
  { day: 14, current: 52, deployAll: 17, doNothing: 72 },
  { day: 15, current: 54, deployAll: 16, doNothing: 73 },
  { day: 16, current: 53, deployAll: 16, doNothing: 75 },
  { day: 17, current: 55, deployAll: 15, doNothing: 76 },
  { day: 18, current: 54, deployAll: 15, doNothing: 78 },
  { day: 19, current: 56, deployAll: 15, doNothing: 79 },
  { day: 20, current: 55, deployAll: 15, doNothing: 80 },
  { day: 21, current: 57, deployAll: 14, doNothing: 81 },
  { day: 22, current: 56, deployAll: 14, doNothing: 82 },
  { day: 23, current: 58, deployAll: 14, doNothing: 83 },
  { day: 24, current: 57, deployAll: 15, doNothing: 84 },
  { day: 25, current: 59, deployAll: 15, doNothing: 85 },
  { day: 26, current: 58, deployAll: 15, doNothing: 85 },
  { day: 27, current: 60, deployAll: 15, doNothing: 85 },
  { day: 28, current: 59, deployAll: 15, doNothing: 85 },
  { day: 29, current: 61, deployAll: 15, doNothing: 85 },
  { day: 30, current: 60, deployAll: 15, doNothing: 85 },
];

// ============================================================
// SLA Bridge Waterfall
// ============================================================
export const SLA_WATERFALL_DATA: {
  label: string;
  value: number;
  type: 'gap' | 'positive' | 'negative' | 'projected';
}[] = [
  { label: 'Starting Gap', value: 42, type: 'gap' },
  { label: 'Patches Deployed', value: -18, type: 'positive' },
  { label: 'New CVEs', value: 8, type: 'negative' },
  { label: 'Auto-Remediated', value: -6, type: 'positive' },
  { label: 'Exceptions', value: 3, type: 'negative' },
  { label: 'Projected', value: 29, type: 'projected' },
];

// ============================================================
// Deployment Timeline
// ============================================================
export interface TimelineEvent {
  id: string;
  title: string;
  subtitle: string;
  status: 'complete' | 'running' | 'failed' | 'pending';
  deploymentType: 'standard' | 'wave' | 'workflow';
  time: string;
  progress?: number;
  details?: string;
}

export const DEPLOYMENT_TIMELINE_DATA: TimelineEvent[] = [
  {
    id: 'deploy-001',
    title: 'KB5034441 — Windows Recovery Env Update',
    subtitle: 'Production Web Servers · 34 endpoints',
    status: 'complete',
    deploymentType: 'standard',
    time: 'Today 6:00 AM',
    details: 'Completed in 22 min. 0 failures.',
  },
  {
    id: 'deploy-002',
    title: 'KB5035853 — Cumulative Update Win 11 22H2',
    subtitle: 'Wave 1: Finance + HR · 89 endpoints',
    status: 'complete',
    deploymentType: 'wave',
    time: 'Today 8:15 AM',
    details: 'Wave 1 of 3 complete. 2 reboots pending.',
  },
  {
    id: 'deploy-003',
    title: 'CVE-2024-21762 Mitigation — FortiOS Patch',
    subtitle: 'Critical Fast Track · 15 appliances',
    status: 'running',
    deploymentType: 'workflow',
    time: 'Today 10:30 AM',
    progress: 68,
    details: 'Applying to 10 of 15 appliances. ETA 8 min.',
  },
  {
    id: 'deploy-004',
    title: 'KB5034122 — .NET Framework 4.8.1 Update',
    subtitle: 'App Servers · 22 endpoints',
    status: 'running',
    deploymentType: 'standard',
    time: 'Today 11:45 AM',
    progress: 41,
    details: 'Downloading on 9 remaining endpoints.',
  },
  {
    id: 'deploy-005',
    title: 'KB5036893 — SharePoint Server 2019 CU',
    subtitle: 'SharePoint Farm · 4 servers',
    status: 'failed',
    deploymentType: 'standard',
    time: 'Today 1:00 PM',
    details: 'Pre-check failed: insufficient disk space on SP-APP-02.',
  },
  {
    id: 'deploy-006',
    title: 'KB5035857 — Cumulative Update Win 11 22H2',
    subtitle: 'Wave 2: Engineering · 47 endpoints',
    status: 'pending',
    deploymentType: 'wave',
    time: 'Today 3:00 PM',
    details: 'Scheduled. Awaiting Wave 1 verification.',
  },
  {
    id: 'deploy-007',
    title: 'RHSA-2024:1891 — OpenSSL Critical Update',
    subtitle: 'Linux Web Tier · 18 endpoints',
    status: 'pending',
    deploymentType: 'workflow',
    time: 'Today 5:00 PM',
    details: 'Queued after approval by sec-team@company.com.',
  },
];

// ============================================================
// Vulnerability Heatmap
// ============================================================
export interface HeatmapEndpoint {
  id: string;
  name: string;
  group: string;
  risk: number;
  cveCount: number;
}

export const VULN_HEATMAP_DATA: HeatmapEndpoint[] = [
  // Production Web group
  { id: 'pw-01', name: 'prod-web-01', group: 'Production Web', risk: 87, cveCount: 14 },
  { id: 'pw-02', name: 'prod-web-02', group: 'Production Web', risk: 72, cveCount: 11 },
  { id: 'pw-03', name: 'prod-web-03', group: 'Production Web', risk: 91, cveCount: 17 },
  { id: 'pw-04', name: 'prod-web-04', group: 'Production Web', risk: 34, cveCount: 4 },
  { id: 'pw-05', name: 'prod-web-05', group: 'Production Web', risk: 58, cveCount: 8 },
  { id: 'pw-06', name: 'prod-lb-01', group: 'Production Web', risk: 45, cveCount: 6 },
  { id: 'pw-07', name: 'prod-cdn-01', group: 'Production Web', risk: 22, cveCount: 2 },

  // Database group
  { id: 'db-01', name: 'db-primary-01', group: 'Database', risk: 78, cveCount: 12 },
  { id: 'db-02', name: 'db-replica-01', group: 'Database', risk: 78, cveCount: 12 },
  { id: 'db-03', name: 'db-replica-02', group: 'Database', risk: 65, cveCount: 9 },
  { id: 'db-04', name: 'db-analytics-01', group: 'Database', risk: 43, cveCount: 5 },
  { id: 'db-05', name: 'db-cache-01', group: 'Database', risk: 19, cveCount: 1 },
  { id: 'db-06', name: 'db-backup-01', group: 'Database', risk: 55, cveCount: 7 },
  { id: 'db-07', name: 'db-dw-01', group: 'Database', risk: 38, cveCount: 4 },

  // Development group
  { id: 'dev-01', name: 'dev-api-01', group: 'Development', risk: 31, cveCount: 3 },
  { id: 'dev-02', name: 'dev-api-02', group: 'Development', risk: 47, cveCount: 6 },
  { id: 'dev-03', name: 'dev-api-03', group: 'Development', risk: 62, cveCount: 9 },
  { id: 'dev-04', name: 'dev-build-01', group: 'Development', risk: 28, cveCount: 2 },
  { id: 'dev-05', name: 'dev-build-02', group: 'Development', risk: 53, cveCount: 7 },
  { id: 'dev-06', name: 'dev-runner-01', group: 'Development', risk: 41, cveCount: 5 },
  { id: 'dev-07', name: 'dev-runner-02', group: 'Development', risk: 36, cveCount: 4 },

  // Infrastructure group
  { id: 'inf-01', name: 'infra-dns-01', group: 'Infrastructure', risk: 15, cveCount: 1 },
  { id: 'inf-02', name: 'infra-dns-02', group: 'Infrastructure', risk: 15, cveCount: 1 },
  { id: 'inf-03', name: 'infra-ntp-01', group: 'Infrastructure', risk: 8, cveCount: 0 },
  { id: 'inf-04', name: 'infra-vpn-01', group: 'Infrastructure', risk: 95, cveCount: 19 },
  { id: 'inf-05', name: 'infra-vpn-02', group: 'Infrastructure', risk: 93, cveCount: 18 },
  { id: 'inf-06', name: 'infra-jump-01', group: 'Infrastructure', risk: 27, cveCount: 2 },
  { id: 'inf-07', name: 'infra-mon-01', group: 'Infrastructure', risk: 44, cveCount: 5 },
];

// ============================================================
// Compliance Rings
// ============================================================
export const COMPLIANCE_RINGS_DATA: {
  framework: string;
  score: number;
  color: string;
}[] = [
  { framework: 'NIST', score: 92, color: 'var(--color-primary)' },
  { framework: 'PCI-DSS', score: 88, color: 'var(--color-cyan)' },
  { framework: 'HIPAA', score: 95, color: 'var(--color-purple)' },
];

// ============================================================
// Agent Rollout
// ============================================================
export const AGENT_ROLLOUT_DATA: { stage: string; count: number }[] = [
  { stage: 'Total Targets', count: 500 },
  { stage: 'Installed', count: 380 },
  { stage: 'Enrolled', count: 260 },
  { stage: 'Healthy', count: 200 },
  { stage: 'Scanning', count: 160 },
];

// ============================================================
// SLA Countdown
// ============================================================
export interface SLACountdownPatch {
  id: string;
  patch: string;
  cve: string;
  hoursRemaining: number;
  totalHours: number;
}

export const SLA_COUNTDOWN_DATA: SLACountdownPatch[] = [
  {
    id: 'sla-001',
    patch: 'KB5034441 — Windows Recovery Env',
    cve: 'CVE-2024-20666',
    hoursRemaining: 4,
    totalHours: 72,
  },
  {
    id: 'sla-002',
    patch: 'KB5036893 — SharePoint Server 2019',
    cve: 'CVE-2024-38094',
    hoursRemaining: 18,
    totalHours: 72,
  },
  {
    id: 'sla-003',
    patch: 'RHSA-2024:1891 — OpenSSL 3.0.x',
    cve: 'CVE-2024-0727',
    hoursRemaining: 31,
    totalHours: 72,
  },
  {
    id: 'sla-004',
    patch: 'KB5035853 — Win 11 22H2 Cumulative',
    cve: 'CVE-2024-21338',
    hoursRemaining: 56,
    totalHours: 120,
  },
  {
    id: 'sla-005',
    patch: 'USN-6648-1 — Linux Kernel 6.5',
    cve: 'CVE-2024-1086',
    hoursRemaining: 84,
    totalHours: 168,
  },
  {
    id: 'sla-006',
    patch: 'KB5034122 — .NET Framework 4.8.1',
    cve: 'CVE-2024-21386',
    hoursRemaining: 110,
    totalHours: 168,
  },
];

// ============================================================
// Workflow Pipeline
// ============================================================
export interface WorkflowStage {
  name: string;
  status: 'complete' | 'running' | 'failed' | 'pending';
}

export interface WorkflowData {
  id: string;
  name: string;
  hostsTotal: number;
  hostsDone: number;
  startedAgo: string;
  stages: WorkflowStage[];
}

export const WORKFLOW_PIPELINE_DATA: WorkflowData[] = [
  {
    id: 'wf-001',
    name: 'Critical Patch Fast Track',
    hostsTotal: 1200,
    hostsDone: 450,
    startedAgo: '2h ago',
    stages: [
      { name: 'Trigger', status: 'complete' },
      { name: 'Filter', status: 'complete' },
      { name: 'Approval', status: 'complete' },
      { name: 'Deploy', status: 'running' },
      { name: 'Verify', status: 'pending' },
    ],
  },
  {
    id: 'wf-002',
    name: 'Standard Monthly Rollout',
    hostsTotal: 800,
    hostsDone: 0,
    startedAgo: '45m ago',
    stages: [
      { name: 'Trigger', status: 'complete' },
      { name: 'Filter', status: 'complete' },
      { name: 'Approval', status: 'running' },
      { name: 'Deploy', status: 'pending' },
      { name: 'Verify', status: 'pending' },
    ],
  },
  {
    id: 'wf-003',
    name: 'Database Server Update',
    hostsTotal: 240,
    hostsDone: 0,
    startedAgo: '1h ago',
    stages: [
      { name: 'Trigger', status: 'complete' },
      { name: 'Filter', status: 'failed' },
      { name: 'Approval', status: 'pending' },
      { name: 'Deploy', status: 'pending' },
      { name: 'Verify', status: 'pending' },
    ],
  },
  {
    id: 'wf-004',
    name: 'Linux Security Baseline',
    hostsTotal: 1800,
    hostsDone: 1800,
    startedAgo: '6h ago',
    stages: [
      { name: 'Trigger', status: 'complete' },
      { name: 'Filter', status: 'complete' },
      { name: 'Approval', status: 'complete' },
      { name: 'Deploy', status: 'complete' },
      { name: 'Verify', status: 'complete' },
    ],
  },
];

// ============================================================
// Patches Horizon (12-month forecast)
// Patch Tuesday pattern: spikes in Jan, Mar, Apr, Jul, Oct, Nov
// ============================================================
export const PATCHES_HORIZON_DATA: {
  date: string;
  critical: number;
  high: number;
  medium: number;
  low: number;
}[] = [
  // Jan — large Patch Tuesday + out-of-band
  { date: 'Jan 2025', critical: 14, high: 32, medium: 51, low: 22 },
  // Feb — lighter month
  { date: 'Feb 2025', critical: 5, high: 18, medium: 34, low: 14 },
  // Mar — big spring release
  { date: 'Mar 2025', critical: 11, high: 27, medium: 43, low: 19 },
  // Apr — moderate
  { date: 'Apr 2025', critical: 8, high: 22, medium: 38, low: 16 },
  // May — quiet
  { date: 'May 2025', critical: 4, high: 14, medium: 29, low: 11 },
  // Jun — mid-year surge
  { date: 'Jun 2025', critical: 9, high: 24, medium: 41, low: 17 },
  // Jul — major Patch Tuesday
  { date: 'Jul 2025', critical: 16, high: 35, medium: 58, low: 24 },
  // Aug — summer lull
  { date: 'Aug 2025', critical: 3, high: 11, medium: 22, low: 9 },
  // Sep — moderate
  { date: 'Sep 2025', critical: 7, high: 19, medium: 36, low: 15 },
  // Oct — large fall release
  { date: 'Oct 2025', critical: 13, high: 30, medium: 49, low: 21 },
  // Nov — large (pre-holiday)
  { date: 'Nov 2025', critical: 12, high: 28, medium: 46, low: 20 },
  // Dec — holiday freeze, minimal
  { date: 'Dec 2025', critical: 2, high: 8, medium: 17, low: 7 },
];

// ============================================================
// Top Critical CVEs
// ============================================================
export interface TopCveItem {
  id: string;
  description: string;
  cvss: number;
  severity: 'critical' | 'high' | 'medium';
  affectedEndpoints: number;
  daysOpen: number;
  patchAvailable: boolean;
}

export const TOP_CVES_DATA: TopCveItem[] = [
  {
    id: 'CVE-2024-21762',
    description: 'FortiOS SSL-VPN RCE',
    cvss: 9.8,
    severity: 'critical',
    affectedEndpoints: 73,
    daysOpen: 47,
    patchAvailable: true,
  },
  {
    id: 'CVE-2024-38094',
    description: 'SharePoint Server RCE',
    cvss: 9.0,
    severity: 'critical',
    affectedEndpoints: 175,
    daysOpen: 62,
    patchAvailable: true,
  },
  {
    id: 'CVE-2024-21338',
    description: 'Windows Kernel EoP',
    cvss: 7.8,
    severity: 'high',
    affectedEndpoints: 89,
    daysOpen: 35,
    patchAvailable: false,
  },
  {
    id: 'CVE-2024-49113',
    description: 'Windows LDAP DoS',
    cvss: 7.5,
    severity: 'high',
    affectedEndpoints: 40,
    daysOpen: 28,
    patchAvailable: true,
  },
  {
    id: 'CVE-2024-0727',
    description: 'OpenSSL NULL dereference',
    cvss: 7.1,
    severity: 'high',
    affectedEndpoints: 31,
    daysOpen: 19,
    patchAvailable: true,
  },
];

// ============================================================
// Mean Time to Patch (MTTP)
// ============================================================
export const MTTP_DATA = {
  current: 8.4,
  target: 7.0,
  trend: -0.8,
  sparkline: [12.1, 11.4, 10.8, 11.2, 10.5, 10.1, 9.7, 9.3, 9.8, 9.2, 8.9, 9.1, 8.7, 8.5, 8.4],
  breakdown: { critical: 3.2, high: 6.8, medium: 12.4, low: 18.9 },
};

// ============================================================
// Patch Success Rate
// ============================================================
export const PATCH_SUCCESS_RATE_DATA = {
  succeeded: 847,
  failed: 23,
  pending: 156,
  total: 1026,
  trend: 2.1,
};

// ============================================================
// CVE Age Distribution
// ============================================================
export const CVE_AGE_DATA: { bucket: string; count: number; color: string }[] = [
  { bucket: '0–7d', count: 8, color: 'var(--color-success)' },
  { bucket: '7–30d', count: 14, color: 'var(--color-caution)' },
  { bucket: '30–90d', count: 22, color: 'var(--color-warning)' },
  { bucket: '90d+', count: 11, color: 'var(--color-danger)' },
];

// ============================================================
// Endpoint Coverage Map
// ============================================================
export const ENDPOINT_COVERAGE_DATA: {
  dept: string;
  total: number;
  covered: number;
  pct: number;
}[] = [
  { dept: 'Engineering', total: 67, covered: 65, pct: 97 },
  { dept: 'Finance', total: 34, covered: 34, pct: 100 },
  { dept: 'HR', total: 28, covered: 24, pct: 86 },
  { dept: 'Sales', total: 52, covered: 38, pct: 73 },
  { dept: 'Operations', total: 43, covered: 40, pct: 93 },
  { dept: 'IT Infra', total: 32, covered: 32, pct: 100 },
];

// ============================================================
// Patch Failure Reasons
// ============================================================
export const PATCH_FAILURE_REASONS_DATA: { reason: string; count: number; color: string }[] = [
  { reason: 'Reboot Required', count: 9, color: 'var(--color-warning)' },
  { reason: 'Insufficient Disk', count: 6, color: 'var(--color-danger)' },
  { reason: 'Network Timeout', count: 4, color: 'var(--color-caution)' },
  { reason: 'Agent Offline', count: 3, color: 'var(--color-muted)' },
  { reason: 'Pre-check Failed', count: 1, color: 'var(--color-subtle)' },
];

// ============================================================
// Upcoming SLA Deadlines (7-day calendar)
// ============================================================
export interface UpcomingSLAPatch {
  id: string;
  cve: string;
  severity: 'critical' | 'high' | 'medium' | 'low';
}

export interface UpcomingSLADay {
  day: string;
  isToday?: boolean;
  patches: UpcomingSLAPatch[];
}

export const UPCOMING_SLA_DATA: UpcomingSLADay[] = [
  {
    day: 'Mon Mar 11',
    isToday: true,
    patches: [
      { id: 'KB5034441', cve: 'CVE-2024-20666', severity: 'critical' },
      { id: 'KB5036893', cve: 'CVE-2024-38094', severity: 'critical' },
    ],
  },
  {
    day: 'Tue Mar 12',
    patches: [{ id: 'RHSA-2024:1891', cve: 'CVE-2024-0727', severity: 'high' }],
  },
  {
    day: 'Wed Mar 13',
    patches: [
      { id: 'KB5035853', cve: 'CVE-2024-21338', severity: 'high' },
      { id: 'USN-6648-1', cve: 'CVE-2024-1086', severity: 'high' },
    ],
  },
  { day: 'Thu Mar 14', patches: [] },
  { day: 'Fri Mar 15', patches: [{ id: 'KB5034122', cve: 'CVE-2024-21386', severity: 'medium' }] },
  { day: 'Sat Mar 16', patches: [] },
  { day: 'Sun Mar 17', patches: [{ id: 'KB5035857', cve: 'CVE-2024-20744', severity: 'high' }] },
];
