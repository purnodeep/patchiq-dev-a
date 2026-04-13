import { useState, useEffect } from 'react';
import { useParams, useNavigate, useSearchParams } from 'react-router';
import { useQueryClient } from '@tanstack/react-query';
import { useCan } from '../../app/auth/AuthContext';
import { computeRiskScore } from '../../lib/risk';
import { toast } from 'sonner';
import { Loader2, RefreshCw, Zap, MoreHorizontal, Clock, Download, Trash2 } from 'lucide-react';
import {
  Skeleton,
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
  Button,
  ErrorState,
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
} from '@patchiq/ui';
import {
  useEndpoint,
  useEndpointPatches,
  useEndpointCVEs,
  useTriggerScan,
  useDecommissionEndpoint,
} from '../../api/hooks/useEndpoints';
import { useCommand, useActiveScan, isTerminalCommandStatus } from '../../api/hooks/useCommand';
import { useEndpointCompliance } from '../../api/hooks/useCompliance';
import { timeAgo } from '../../lib/time';
import { deriveStatus } from './deriveStatus';
import { DeploymentWizard } from '../../components/DeploymentWizard';
import { buildEndpointReportCsv, downloadCsvString } from './export-csv';
import { OverviewTab } from './tabs/OverviewTab';
import { HardwareTab } from './tabs/HardwareTab';
import { SoftwareTab } from './tabs/SoftwareTab';
import { VulnerabilitiesTab } from './tabs/VulnerabilitiesTab';
import { PatchesTab } from './tabs/PatchesTab';
import { HistoryTab } from './tabs/HistoryTab';
import { AuditTab } from './tabs/AuditTab';

const STATUS_COLORS: Record<string, { dot: string; label: string; text: string }> = {
  online: { dot: 'var(--signal-healthy)', label: 'Online', text: 'var(--signal-healthy)' },
  offline: { dot: 'var(--signal-critical)', label: 'Offline', text: 'var(--signal-critical)' },
  pending: { dot: 'var(--signal-warning)', label: 'Pending', text: 'var(--signal-warning)' },
  stale: { dot: 'var(--text-faint)', label: 'Stale', text: 'var(--text-faint)' },
};

const TABS = [
  { key: 'overview', label: 'Overview' },
  { key: 'hardware', label: 'Hardware' },
  { key: 'software', label: 'Software' },
  { key: 'patches', label: 'Patches' },
  { key: 'cves', label: 'CVE Exposure' },
  { key: 'deployments', label: 'Deployments' },
  { key: 'audit', label: 'Audit' },
] as const;

type TabKey = (typeof TABS)[number]['key'];

/** Single horizontal health strip with 4 metrics and vertical dividers. */
function HealthStrip({
  riskScore,
  patchCoverage,
  complianceStr,
  lastScan,
}: {
  riskScore: number;
  patchCoverage: number | null;
  complianceStr: string;
  lastScan: string | null;
}) {
  const riskColor =
    riskScore >= 7
      ? 'var(--signal-critical)'
      : riskScore >= 3
        ? 'var(--signal-warning)'
        : 'var(--signal-healthy)';
  const coveragePct = patchCoverage ?? 0;
  const coverageColor =
    coveragePct >= 95
      ? 'var(--signal-healthy)'
      : coveragePct >= 70
        ? 'var(--signal-warning)'
        : 'var(--signal-critical)';

  const metricStyle: React.CSSProperties = {
    display: 'flex',
    alignItems: 'center',
    gap: 10,
    flex: 1,
    padding: '0 20px',
  };

  const labelStyle: React.CSSProperties = {
    fontSize: 10,
    fontFamily: 'var(--font-mono)',
    color: 'var(--text-muted)',
    textTransform: 'uppercase',
    letterSpacing: '0.06em',
    whiteSpace: 'nowrap',
  };

  const valueStyle: React.CSSProperties = {
    fontSize: 15,
    fontFamily: 'var(--font-mono)',
    fontWeight: 600,
    color: 'var(--text-emphasis)',
    whiteSpace: 'nowrap',
  };

  function MiniBar({ pct, color }: { pct: number; color: string }) {
    return (
      <div
        style={{
          width: 64,
          height: 3,
          borderRadius: 2,
          background: 'var(--bg-inset)',
          overflow: 'hidden',
          flexShrink: 0,
        }}
      >
        <div
          style={{
            height: '100%',
            width: `${Math.min(100, pct)}%`,
            background: color,
            borderRadius: 2,
          }}
        />
      </div>
    );
  }

  return (
    <div
      style={{
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: 8,
        display: 'flex',
        alignItems: 'center',
        height: 52,
        overflow: 'hidden',
      }}
    >
      {/* Risk Score */}
      <div style={metricStyle}>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          <span style={labelStyle}>Risk Score</span>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <span style={{ ...valueStyle, color: riskColor }}>{riskScore.toFixed(1)}</span>
            <span
              style={{ fontSize: 11, color: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}
            >
              / 10
            </span>
            <MiniBar pct={(riskScore / 10) * 100} color={riskColor} />
          </div>
        </div>
      </div>

      <div style={{ width: 1, height: 28, background: 'var(--border)', flexShrink: 0 }} />

      {/* Patch Coverage */}
      <div style={metricStyle}>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          <span style={labelStyle}>Patch Coverage</span>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <span style={{ ...valueStyle, color: coverageColor }}>
              {patchCoverage != null ? `${coveragePct.toFixed(1)}%` : '—'}
            </span>
            <MiniBar pct={coveragePct} color={coverageColor} />
          </div>
        </div>
      </div>

      <div style={{ width: 1, height: 28, background: 'var(--border)', flexShrink: 0 }} />

      {/* Compliance */}
      <div style={metricStyle}>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          <span style={labelStyle}>Compliance</span>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <span style={valueStyle}>{complianceStr}</span>
            <MiniBar pct={complianceStr === '—' ? 0 : 100} color="var(--signal-healthy)" />
          </div>
        </div>
      </div>

      <div style={{ width: 1, height: 28, background: 'var(--border)', flexShrink: 0 }} />

      {/* Last Scan */}
      <div style={{ ...metricStyle, flex: '0 0 auto' }}>
        <Clock style={{ width: 13, height: 13, color: 'var(--text-muted)', flexShrink: 0 }} />
        <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          <span style={labelStyle}>Last Scan</span>
          <span style={valueStyle}>{lastScan ? timeAgo(lastScan) : '—'}</span>
        </div>
      </div>
    </div>
  );
}

export function EndpointDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { data: endpoint, isLoading, isError, refetch } = useEndpoint(id ?? '');
  const { data: patchesData } = useEndpointPatches(id ?? '');
  const { data: cvesData } = useEndpointCVEs(id ?? '');

  const { data: complianceData } = useEndpointCompliance(id ?? '');
  const triggerScan = useTriggerScan();
  const decommission = useDecommissionEndpoint();
  const navigate = useNavigate();
  const can = useCan();
  const queryClient = useQueryClient();
  const [activeTab, setActiveTab] = useState<TabKey>('overview');
  const [deployOpen, setDeployOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [searchParams, setSearchParams] = useSearchParams();
  const urlCommandId = searchParams.get('scan');
  const { activeCommandId: serverCommandId } = useActiveScan(id);
  const activeCommandId = urlCommandId ?? serverCommandId;

  const { data: activeCommand } = useCommand(activeCommandId);
  const isScanning =
    activeCommandId != null &&
    (!activeCommand || !isTerminalCommandStatus(activeCommand.status));

  useEffect(() => {
    if (!activeCommand || !activeCommandId) return;
    if (!isTerminalCommandStatus(activeCommand.status)) return;

    if (activeCommand.status === 'succeeded') {
      toast.success(`Scan completed for ${endpoint?.hostname ?? 'endpoint'}`);
    } else if (activeCommand.status === 'failed') {
      toast.error(`Scan failed: ${activeCommand.error_message ?? 'unknown error'}`);
    } else {
      toast.warning(`Scan ${activeCommand.status}`);
    }

    void queryClient.invalidateQueries({ queryKey: ['endpoints', id] });
    void queryClient.invalidateQueries({ queryKey: ['endpoints', id, 'packages'] });
    void queryClient.invalidateQueries({ queryKey: ['endpoints', id, 'active-scan'] });

    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      next.delete('scan');
      return next;
    }, { replace: true });
  }, [activeCommand, activeCommandId, id, endpoint?.hostname, queryClient, setSearchParams]);

  if (!id) {
    return (
      <div style={{ padding: 24, fontSize: 13, color: 'var(--signal-critical)' }}>
        Endpoint not found
      </div>
    );
  }

  if (isLoading) {
    return (
      <div style={{ padding: '20px 24px', display: 'flex', flexDirection: 'column', gap: 16 }}>
        <Skeleton className="h-16 w-full rounded-lg" />
        <Skeleton className="h-14 w-full rounded-lg" />
        <Skeleton className="h-8 w-96" />
        <Skeleton className="h-[500px] rounded-xl" />
      </div>
    );
  }

  if (isError || !endpoint) {
    return (
      <div style={{ padding: 24 }}>
        <ErrorState
          title="Failed to load endpoint"
          message="Unable to fetch endpoint data from the server."
          onRetry={refetch}
        />
      </div>
    );
  }

  const displayStatus = deriveStatus(endpoint.status, endpoint.last_seen);
  const statusColors = STATUS_COLORS[displayStatus] ?? STATUS_COLORS.offline;

  // Health strip calculations
  const patches = patchesData?.data ?? [];
  const installedCount = patches.filter((p) => p.status === 'installed').length;
  const pendingPatches = patches.filter((p) => p.status === 'pending' || p.status === 'available');
  const pendingCount = pendingPatches.length;
  const totalCount = installedCount + pendingCount;
  const patchCoverage = totalCount > 0 ? (installedCount / totalCount) * 100 : null;

  const complianceEvals = Array.isArray(complianceData) ? complianceData : [];
  const compliantCount = complianceEvals.filter(
    (e) => (e as { state: string }).state?.toUpperCase() === 'COMPLIANT',
  ).length;
  const complianceStr =
    complianceEvals.length > 0 ? `${compliantCount}/${complianceEvals.length}` : '—';

  // Risk score — derived from CVE severity counts (canonical source of vulnerability data).
  const cves = cvesData?.data ?? [];
  const riskScore = computeRiskScore({
    criticalCves: cves.filter((c) => c.cve_severity === 'critical').length,
    highCves: cves.filter((c) => c.cve_severity === 'high').length,
    mediumCves: cves.filter((c) => c.cve_severity === 'medium').length,
  });

  return (
    <div
      style={{
        padding: '20px 24px',
        display: 'flex',
        flexDirection: 'column',
        gap: 16,
        maxWidth: '100%',
      }}
    >
      {/* Header — 2 rows, no card */}
      <div>
        {/* Row 1: hostname + actions */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            marginBottom: 8,
          }}
        >
          <h1
            style={{
              fontFamily: 'var(--font-sans)',
              fontSize: 22,
              fontWeight: 600,
              color: 'var(--text-emphasis)',
              margin: 0,
              letterSpacing: '-0.01em',
            }}
          >
            {endpoint.hostname}
          </h1>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <button
              disabled={!can('deployments', 'create')}
              title={!can('deployments', 'create') ? "You don't have permission" : undefined}
              onClick={() => setDeployOpen(true)}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 6,
                padding: '7px 14px',
                borderRadius: 7,
                border: 'none',
                background: 'var(--accent)',
                color: 'var(--btn-accent-text, #000)',
                fontSize: 12,
                fontWeight: 600,
                cursor: !can('deployments', 'create') ? 'not-allowed' : 'pointer',
                opacity: !can('deployments', 'create') ? 0.5 : 1,
              }}
            >
              <Zap style={{ width: 13, height: 13 }} />
              Deploy Patches
            </button>
            <button
              disabled={!can('endpoints', 'scan') || triggerScan.isPending || isScanning}
              title={!can('endpoints', 'scan') ? "You don't have permission" : undefined}
              onClick={() => {
                triggerScan.mutate(id, {
                  onSuccess: (data) => {
                    const cmdId = data?.command_id;
                    if (!cmdId) {
                      toast.warning('Scan triggered but no command ID returned');
                      return;
                    }
                    toast.success(`Scan triggered for ${endpoint.hostname}`);
                    setSearchParams((prev) => {
                      const next = new URLSearchParams(prev);
                      next.set('scan', cmdId);
                      return next;
                    }, { replace: true });
                  },
                  onError: (err) => {
                    toast.error(err instanceof Error ? err.message : 'Failed to trigger scan');
                  },
                });
              }}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 6,
                padding: '7px 14px',
                borderRadius: 7,
                border: '1px solid var(--border)',
                background: 'none',
                color: 'var(--text-secondary)',
                fontSize: 12,
                fontWeight: 500,
                cursor:
                  !can('endpoints', 'scan') || triggerScan.isPending || isScanning ? 'not-allowed' : 'pointer',
                opacity: !can('endpoints', 'scan') || triggerScan.isPending || isScanning ? 0.7 : 1,
              }}
            >
              {triggerScan.isPending || isScanning ? (
                <>
                  <Loader2 style={{ width: 13, height: 13 }} className="animate-spin" />
                  {isScanning && activeCommand?.created_at
                    ? `Scanning... ${Math.floor((Date.now() - new Date(activeCommand.created_at).getTime()) / 1000)}s`
                    : 'Scanning...'}
                </>
              ) : (
                <>
                  <RefreshCw style={{ width: 13, height: 13 }} />
                  Scan Now
                </>
              )}
            </button>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <button
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    width: 34,
                    height: 34,
                    borderRadius: 7,
                    border: '1px solid var(--border)',
                    background: 'none',
                    color: 'var(--text-muted)',
                    cursor: 'pointer',
                  }}
                >
                  <MoreHorizontal style={{ width: 16, height: 16 }} />
                </button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem
                  onClick={() => {
                    const csv = buildEndpointReportCsv(endpoint, patches);
                    const date = new Date().toISOString().slice(0, 10);
                    downloadCsvString(csv, `${endpoint.hostname}-report-${date}.csv`);
                    toast.success('Report exported');
                  }}
                >
                  <Download style={{ width: 13, height: 13, marginRight: 8 }} />
                  Export Report
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  disabled={!can('endpoints', 'delete')}
                  title={!can('endpoints', 'delete') ? "You don't have permission" : undefined}
                  onClick={() => setDeleteOpen(true)}
                  style={{
                    color: 'var(--signal-critical)',
                    opacity: !can('endpoints', 'delete') ? 0.5 : 1,
                  }}
                >
                  <Trash2 style={{ width: 13, height: 13, marginRight: 8 }} />
                  Delete Endpoint
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>

        {/* Row 2: status dot + meta chips */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          {/* Status: dot + text only, no pill */}
          <span style={{ display: 'inline-flex', alignItems: 'center', gap: 5 }}>
            <span style={{ position: 'relative', display: 'inline-flex', width: 8, height: 8 }}>
              {endpoint.status === 'online' && (
                <span
                  style={{
                    position: 'absolute',
                    inset: 0,
                    borderRadius: '50%',
                    background: statusColors.dot,
                    opacity: 0.5,
                    animation: 'ping 1.5s ease-out infinite',
                  }}
                />
              )}
              <span
                style={{
                  position: 'relative',
                  width: 8,
                  height: 8,
                  borderRadius: '50%',
                  background: statusColors.dot,
                  display: 'inline-block',
                }}
              />
            </span>
            <span style={{ fontSize: 12, fontWeight: 500, color: statusColors.text }}>
              {statusColors.label}
            </span>
          </span>

          {/* Divider */}
          <span style={{ width: 1, height: 12, background: 'var(--border)' }} />

          {/* Meta chips */}
          {[
            endpoint.os_version,
            endpoint.agent_version ? `v${endpoint.agent_version}` : null,
            endpoint.ip_address,
            endpoint.enrolled_at ? `Enrolled ${timeAgo(endpoint.enrolled_at)}` : null,
          ]
            .filter(Boolean)
            .map((val, i) => (
              <span
                key={i}
                style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  padding: '2px 8px',
                  borderRadius: 4,
                  border: '1px solid var(--border)',
                  background: 'none',
                  fontSize: 11,
                  fontFamily: 'var(--font-mono)',
                  color: 'var(--text-muted)',
                  whiteSpace: 'nowrap',
                }}
              >
                {val}
              </span>
            ))}
        </div>
      </div>

      {/* Health strip */}
      <HealthStrip
        riskScore={riskScore}
        patchCoverage={patchCoverage}
        complianceStr={complianceStr}
        lastScan={endpoint.last_scan}
      />

      {/* Flat underline tabs */}
      <div style={{ borderBottom: '1px solid var(--border)' }}>
        <div style={{ display: 'flex', gap: 0 }}>
          {TABS.map((tab) => {
            const isActive = activeTab === tab.key;
            return (
              <button
                key={tab.key}
                onClick={() => setActiveTab(tab.key)}
                style={{
                  padding: '8px 16px',
                  background: 'none',
                  border: 'none',
                  borderBottom: isActive ? '2px solid var(--accent)' : '2px solid transparent',
                  marginBottom: -1,
                  fontSize: 13,
                  fontWeight: isActive ? 600 : 400,
                  color: isActive ? 'var(--text-emphasis)' : 'var(--text-muted)',
                  cursor: 'pointer',
                  whiteSpace: 'nowrap',
                  transition: 'color 150ms, border-color 150ms',
                }}
              >
                {tab.label}
              </button>
            );
          })}
        </div>
      </div>

      {/* Tab content */}
      <div style={{ marginTop: 16 }}>
        {activeTab === 'overview' && (
          <OverviewTab endpoint={endpoint} onTabChange={(tab) => setActiveTab(tab as TabKey)} />
        )}
        {activeTab === 'hardware' && <HardwareTab endpointId={id} />}
        {activeTab === 'software' && (
          <SoftwareTab endpointId={id} packageCount={endpoint.package_count} />
        )}
        {activeTab === 'patches' && <PatchesTab endpointId={id} />}
        {activeTab === 'cves' && (
          <VulnerabilitiesTab endpointId={id} vulnerableCveCount={endpoint.vulnerable_cve_count} />
        )}
        {activeTab === 'deployments' && <HistoryTab endpointId={id} />}
        {activeTab === 'audit' && <AuditTab endpointId={id} />}
      </div>

      <DeploymentWizard open={deployOpen} onOpenChange={setDeployOpen} />

      <Dialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle>Delete Endpoint</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete{' '}
              <span
                style={{
                  fontFamily: 'var(--font-mono)',
                  color: 'var(--text-emphasis)',
                  fontWeight: 600,
                }}
              >
                {endpoint.hostname}
              </span>
              ? The endpoint will be marked as decommissioned and can be found under the Deleted
              filter.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="gap-2 sm:gap-0">
            <Button variant="outline" onClick={() => setDeleteOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              disabled={decommission.isPending}
              onClick={() => {
                decommission.mutate(id, {
                  onSuccess: () => {
                    setDeleteOpen(false);
                    navigate('/endpoints');
                  },
                  onError: (err) => {
                    toast.error(err instanceof Error ? err.message : 'Failed to delete endpoint');
                  },
                });
              }}
            >
              {decommission.isPending ? <Loader2 className="mr-1.5 h-4 w-4 animate-spin" /> : null}
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
