export function mockEndpoint(overrides?: Partial<Record<string, unknown>>) {
  return {
    id: 'ep-001',
    hostname: 'web-server-01',
    os_family: 'linux',
    os_version: 'Ubuntu 22.04',
    status: 'online',
    agent_version: '1.0.0',
    last_heartbeat: new Date().toISOString(),
    pending_patch_count: 3,
    installed_count: 47,
    failed_count: 0,
    risk_score: 65,
    tags: [
      { key: 'env', value: 'production' },
      { key: 'os', value: 'linux' },
    ],
    ...overrides,
  };
}

export function mockPatch(overrides?: Partial<Record<string, unknown>>) {
  return {
    id: 'patch-001',
    name: 'Security Update 2026-03',
    severity: 'critical',
    os_family: 'linux',
    version: '1.2.3',
    cve_count: 2,
    published_at: new Date().toISOString(),
    ...overrides,
  };
}

export function mockDeployment(overrides?: Partial<Record<string, unknown>>) {
  return {
    id: 'dep-001',
    name: 'Critical Patch Rollout',
    status: 'running',
    source_type: 'policy',
    target_count: 50,
    completed_count: 23,
    failed_count: 2,
    created_at: new Date().toISOString(),
    waves: [],
    ...overrides,
  };
}

export function mockPolicy(overrides?: Partial<Record<string, unknown>>) {
  return {
    id: 'pol-001',
    name: 'Auto-patch Critical',
    mode: 'enforce',
    enabled: true,
    filter_type: 'tag',
    filters: { expression: 'env:production AND os:linux' },
    matched_endpoint_count: 42,
    ...overrides,
  };
}

export function mockCVE(overrides?: Partial<Record<string, unknown>>) {
  return {
    id: 'CVE-2026-1234',
    cvss_score: 9.8,
    severity: 'critical',
    description: 'Remote code execution vulnerability',
    kev_active: true,
    exploit_active: false,
    affected_endpoint_count: 15,
    ...overrides,
  };
}

export function mockComplianceFramework(overrides?: Partial<Record<string, unknown>>) {
  return {
    id: 'fw-001',
    name: 'CIS Benchmarks',
    enabled: true,
    score: 87,
    total_controls: 120,
    passing_controls: 104,
    failing_controls: 16,
    ...overrides,
  };
}

export function mockTag(overrides?: Partial<Record<string, unknown>>) {
  return {
    id: 'tag-001',
    key: 'env',
    value: 'production',
    endpoint_count: 42,
    created_at: new Date().toISOString(),
    ...overrides,
  };
}

export function mockUser(overrides?: Partial<Record<string, unknown>>) {
  return {
    id: 'user-001',
    name: 'Admin User',
    email: 'admin@patchiq.io',
    roles: ['admin'],
    ...overrides,
  };
}

export function mockDashboardSummary(overrides?: Partial<Record<string, unknown>>) {
  return {
    total_endpoints: 1247,
    online_count: 1198,
    offline_count: 49,
    total_patches: 342,
    critical_patches: 12,
    pending_deployments: 3,
    compliance_score: 87,
    ...overrides,
  };
}
