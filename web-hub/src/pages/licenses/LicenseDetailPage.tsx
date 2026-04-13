import { useState } from 'react';
import { useParams, Link, useNavigate } from 'react-router';
import { toast } from 'sonner';
import {
  useLicense,
  useRevokeLicense,
  useAssignLicense,
  useRenewLicense,
  useLicenseUsageHistory,
  useLicenseAuditTrail,
} from '../../api/hooks/useLicenses';
import { useClient, useClients } from '../../api/hooks/useClients';
import {
  Button,
  Skeleton,
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@patchiq/ui';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@patchiq/ui';
import {
  ArrowLeft,
  Copy,
  CheckCircle,
  MoreHorizontal,
  RefreshCw,
  UserPlus,
  XCircle,
} from 'lucide-react';
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
} from '@patchiq/ui';
import { computeStatus } from '../../lib/licenseUtils';
import { tierBadgeStyle, getTierFeatures } from '../../lib/tierUtils';

function daysRemaining(dateStr: string): number {
  return Math.ceil((new Date(dateStr).getTime() - Date.now()) / (1000 * 60 * 60 * 24));
}

function maskKey(key: string | undefined): string {
  if (!key) return '****-****-****';
  return `****-****-${key.slice(-4)}`;
}

export const LicenseDetailPage = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { data: license, isLoading, isError } = useLicense(id ?? '');
  const { data: assignedClient } = useClient(license?.client_id ?? undefined);
  const { data: clientsData } = useClients({ limit: 100 });
  const revokeMutation = useRevokeLicense();
  const assignMutation = useAssignLicense();
  const renewMutation = useRenewLicense();
  const [activeTab, setActiveTab] = useState<'overview' | 'usage' | 'audit'>('overview');
  const [copied, setCopied] = useState(false);
  const [assigning, setAssigning] = useState(false);
  const [selectedClientId, setSelectedClientId] = useState('');
  const [revokeDialogOpen, setRevokeDialogOpen] = useState(false);
  const [showRenewDialog, setShowRenewDialog] = useState(false);
  const [renewForm, setRenewForm] = useState({
    tier: '',
    max_endpoints: 0,
    expires_at: '',
  });

  const licenseId = id;

  const { data: usageData, isLoading: usageLoading } = useLicenseUsageHistory(licenseId, 90);
  const { data: auditData, isLoading: auditLoading } = useLicenseAuditTrail(licenseId, 50);

  if (isLoading) {
    return (
      <div className="p-6 space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-32 w-full" />
        <Skeleton className="h-48 w-full" />
      </div>
    );
  }

  if (isError || !license) {
    return (
      <div className="p-6">
        <Link
          to="/licenses"
          className="text-sm text-muted-foreground hover:underline flex items-center gap-1 mb-4"
        >
          <ArrowLeft className="h-4 w-4" /> Back to Licenses
        </Link>
        <div className="text-destructive">Failed to load license.</div>
      </div>
    );
  }

  const status = computeStatus(license);
  const days = daysRemaining(license.expires_at);
  const usagePct =
    license.client_endpoint_count != null
      ? Math.round((license.client_endpoint_count / license.max_endpoints) * 100)
      : null;

  const features = getTierFeatures(license.tier);

  const handleCopyKey = () => {
    if (license.license_key) {
      void navigator.clipboard.writeText(license.license_key);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  // Derive max capacity from usage history points if available
  const chartMax =
    usageData?.points && usageData.points.length > 0
      ? (usageData.points[0]?.endpoints_limit ?? license.max_endpoints)
      : license.max_endpoints;

  return (
    <div style={{ padding: '20px 24px', display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Header: name + actions */}
      <div>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            marginBottom: 6,
          }}
        >
          <h1
            style={{
              fontFamily: 'var(--font-sans)',
              fontSize: 20,
              fontWeight: 600,
              color: 'var(--text-emphasis)',
              margin: 0,
            }}
          >
            {license.customer_name}
          </h1>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, flexShrink: 0 }}>
            {assigning ? (
              <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                <Select
                  value={selectedClientId}
                  onValueChange={(v: string) => setSelectedClientId(v)}
                >
                  <SelectTrigger
                    className="h-7 text-sm"
                    style={{
                      width: 140,
                      borderColor: 'var(--border)',
                      background: 'var(--bg-card)',
                    }}
                  >
                    <SelectValue placeholder="Select client..." />
                  </SelectTrigger>
                  <SelectContent>
                    {(clientsData?.clients ?? [])
                      .filter((c) => c.status === 'approved')
                      .map((c) => (
                        <SelectItem key={c.id} value={c.id}>
                          {c.hostname}
                        </SelectItem>
                      ))}
                  </SelectContent>
                </Select>
                <button
                  onClick={() => {
                    if (selectedClientId) {
                      assignMutation.mutate(
                        { id: license.id, clientId: selectedClientId },
                        {
                          onSuccess: () => {
                            toast.success('License assigned');
                            setAssigning(false);
                            setSelectedClientId('');
                          },
                          onError: (err) => {
                            toast.error(
                              err instanceof Error ? err.message : 'Failed to assign license',
                            );
                          },
                        },
                      );
                    }
                  }}
                  disabled={!selectedClientId || assignMutation.isPending}
                  style={{
                    padding: '5px 12px',
                    borderRadius: 7,
                    border: 'none',
                    background: 'var(--accent)',
                    color: 'var(--btn-accent-text, #000)',
                    fontSize: 12,
                    fontWeight: 600,
                    cursor: !selectedClientId ? 'not-allowed' : 'pointer',
                    opacity: !selectedClientId ? 0.5 : 1,
                  }}
                >
                  Assign
                </button>
                <button
                  onClick={() => setAssigning(false)}
                  style={{
                    padding: '5px 10px',
                    borderRadius: 7,
                    border: '1px solid var(--border)',
                    background: 'none',
                    color: 'var(--text-muted)',
                    fontSize: 12,
                    cursor: 'pointer',
                  }}
                >
                  ✕
                </button>
              </div>
            ) : (
              status !== 'revoked' && (
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <button
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        width: 32,
                        height: 32,
                        borderRadius: 7,
                        border: '1px solid var(--border)',
                        background: 'none',
                        color: 'var(--text-muted)',
                        cursor: 'pointer',
                      }}
                    >
                      <MoreHorizontal style={{ width: 15, height: 15 }} />
                    </button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem
                      onClick={() => {
                        setRenewForm({
                          tier: license.tier ?? '',
                          max_endpoints: license.max_endpoints ?? 0,
                          expires_at: new Date(Date.now() + 365 * 86400000)
                            .toISOString()
                            .split('T')[0],
                        });
                        setShowRenewDialog(true);
                      }}
                    >
                      <RefreshCw style={{ width: 13, height: 13, marginRight: 8 }} /> Renew
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => setAssigning(true)}>
                      <UserPlus style={{ width: 13, height: 13, marginRight: 8 }} /> Reassign
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={handleCopyKey}>
                      <Copy style={{ width: 13, height: 13, marginRight: 8 }} />{' '}
                      {copied ? 'Copied!' : 'Copy Key'}
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      onClick={() => setRevokeDialogOpen(true)}
                      disabled={revokeMutation.isPending}
                      style={{ color: 'var(--signal-critical)' }}
                    >
                      <XCircle style={{ width: 13, height: 13, marginRight: 8 }} /> Revoke
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              )
            )}
          </div>
        </div>

        {/* Meta row */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span style={{ display: 'inline-flex', alignItems: 'center', gap: 5 }}>
            <span style={{ position: 'relative', display: 'inline-flex', width: 7, height: 7 }}>
              {status === 'active' && (
                <span
                  style={{
                    position: 'absolute',
                    inset: 0,
                    borderRadius: '50%',
                    background: 'var(--signal-healthy)',
                    opacity: 0.5,
                    animation: 'ping 1.5s ease-out infinite',
                  }}
                />
              )}
              <span
                style={{
                  position: 'relative',
                  width: 7,
                  height: 7,
                  borderRadius: '50%',
                  display: 'inline-block',
                  background:
                    status === 'active'
                      ? 'var(--signal-healthy)'
                      : status === 'expiring'
                        ? 'var(--signal-warning)'
                        : 'var(--signal-critical)',
                }}
              />
            </span>
            <span
              style={{
                fontSize: 11,
                fontWeight: 500,
                textTransform: 'capitalize',
                color:
                  status === 'active'
                    ? 'var(--signal-healthy)'
                    : status === 'expiring'
                      ? 'var(--signal-warning)'
                      : 'var(--signal-critical)',
              }}
            >
              {status}
            </span>
          </span>
          <span style={{ width: 1, height: 12, background: 'var(--border)' }} />
          <span
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              padding: '1px 7px',
              borderRadius: 4,
              fontSize: 10,
              fontFamily: 'var(--font-mono)',
              whiteSpace: 'nowrap',
              ...tierBadgeStyle(license.tier),
            }}
          >
            {license.tier}
          </span>
          {assignedClient && (
            <>
              <span style={{ width: 1, height: 12, background: 'var(--border)' }} />
              <Link
                to={`/clients/${assignedClient.id}`}
                style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: 4,
                  fontSize: 11,
                  fontFamily: 'var(--font-mono)',
                  color: 'var(--accent)',
                  textDecoration: 'none',
                }}
              >
                <span
                  style={{
                    width: 5,
                    height: 5,
                    borderRadius: '50%',
                    background:
                      assignedClient.status === 'approved'
                        ? 'var(--signal-healthy)'
                        : 'var(--text-muted)',
                  }}
                />
                {assignedClient.hostname}
              </Link>
            </>
          )}
          {license.customer_email && (
            <>
              <span style={{ width: 1, height: 12, background: 'var(--border)' }} />
              <span
                style={{ fontSize: 11, fontFamily: 'var(--font-mono)', color: 'var(--text-muted)' }}
              >
                {license.customer_email}
              </span>
            </>
          )}
        </div>
      </div>

      {/* Health strip */}
      <div
        style={{
          background: 'var(--bg-card)',
          border: '1px solid var(--border)',
          borderRadius: 8,
          display: 'flex',
          alignItems: 'center',
          height: 48,
          overflow: 'hidden',
        }}
      >
        {[
          {
            label: 'Capacity',
            value: license.max_endpoints.toLocaleString(),
            color: 'var(--text-emphasis)',
          },
          {
            label: 'In Use',
            value: license.client_endpoint_count?.toLocaleString() ?? '—',
            color: 'var(--text-emphasis)',
            bar: usagePct,
          },
          {
            label: 'Expires In',
            value: days > 0 ? `${days}d` : 'Expired',
            color:
              days > 60
                ? 'var(--signal-healthy)'
                : days > 30
                  ? 'var(--signal-warning)'
                  : 'var(--signal-critical)',
          },
          { label: 'Tier Features', value: String(features.length), color: 'var(--text-emphasis)' },
        ].map((m, i, arr) => (
          <div key={m.label} style={{ display: 'contents' }}>
            <div
              style={{
                flex: 1,
                padding: '0 16px',
                display: 'flex',
                flexDirection: 'column',
                gap: 1,
              }}
            >
              <span
                style={{
                  fontSize: 9,
                  fontFamily: 'var(--font-mono)',
                  color: 'var(--text-muted)',
                  textTransform: 'uppercase',
                  letterSpacing: '0.06em',
                }}
              >
                {m.label}
              </span>
              <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                <span
                  style={{
                    fontSize: 14,
                    fontFamily: 'var(--font-mono)',
                    fontWeight: 600,
                    color: m.color,
                  }}
                >
                  {m.value}
                </span>
                {m.bar != null && (
                  <>
                    <div
                      style={{
                        width: 48,
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
                          width: `${Math.min(100, m.bar)}%`,
                          borderRadius: 2,
                          background:
                            m.bar > 90
                              ? 'var(--signal-critical)'
                              : m.bar > 70
                                ? 'var(--signal-warning)'
                                : 'var(--signal-healthy)',
                        }}
                      />
                    </div>
                    <span
                      style={{
                        fontSize: 10,
                        color: 'var(--text-muted)',
                        fontFamily: 'var(--font-mono)',
                      }}
                    >
                      {m.bar}%
                    </span>
                  </>
                )}
              </div>
            </div>
            {i < arr.length - 1 && (
              <div style={{ width: 1, height: 24, background: 'var(--border)', flexShrink: 0 }} />
            )}
          </div>
        ))}
      </div>

      {/* Tabs */}
      <div>
        <div
          style={{
            display: 'flex',
            gap: 0,
            borderBottom: '1px solid var(--border)',
          }}
        >
          {(['overview', 'usage', 'audit'] as const).map((tab) => (
            <button
              key={tab}
              type="button"
              onClick={() => setActiveTab(tab)}
              style={{
                padding: '8px 16px',
                fontSize: 13,
                fontWeight: activeTab === tab ? 600 : 400,
                color: activeTab === tab ? 'var(--text-emphasis)' : 'var(--text-muted)',
                border: 'none',
                borderBottom:
                  activeTab === tab ? '2px solid var(--accent)' : '2px solid transparent',
                background: 'transparent',
                cursor: 'pointer',
                transition: 'color 150ms ease',
                marginBottom: -1,
              }}
            >
              {tab === 'usage' ? 'Usage History' : tab.charAt(0).toUpperCase() + tab.slice(1)}
            </button>
          ))}
        </div>

        <div style={{ paddingTop: 16 }}>
          {activeTab === 'overview' && (
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 12 }}>
              {/* License Details tile */}
              <div
                style={{
                  background: 'var(--bg-card)',
                  border: '1px solid var(--border)',
                  borderRadius: 10,
                  overflow: 'hidden',
                }}
              >
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: '10px 14px',
                    borderBottom: '1px solid var(--border)',
                  }}
                >
                  <span
                    style={{
                      fontSize: 11,
                      fontWeight: 600,
                      letterSpacing: '0.06em',
                      textTransform: 'uppercase',
                      color: 'var(--text-secondary)',
                      fontFamily: 'var(--font-mono)',
                    }}
                  >
                    License Details
                  </span>
                </div>
                <div
                  style={{ padding: '12px 14px', display: 'flex', flexDirection: 'column', gap: 8 }}
                >
                  {[
                    ['Customer', license.customer_name],
                    ['Email', license.customer_email ?? '—'],
                    ['Key', maskKey(license.license_key)],
                    ['Issued', new Date(license.issued_at).toLocaleDateString()],
                    ['Expires', new Date(license.expires_at).toLocaleDateString()],
                    ['Notes', license.notes ?? '—'],
                  ].map(([label, val]) => (
                    <div
                      key={label}
                      style={{
                        display: 'flex',
                        justifyContent: 'space-between',
                        alignItems: 'center',
                      }}
                    >
                      <span
                        style={{
                          fontSize: 11,
                          color: 'var(--text-muted)',
                          fontFamily: 'var(--font-mono)',
                        }}
                      >
                        {label}
                      </span>
                      <span
                        style={{
                          fontSize: 11,
                          color: 'var(--text-emphasis)',
                          fontFamily: 'var(--font-mono)',
                          maxWidth: '60%',
                          textAlign: 'right',
                          overflow: 'hidden',
                          textOverflow: 'ellipsis',
                          whiteSpace: 'nowrap',
                        }}
                      >
                        {val}
                      </span>
                    </div>
                  ))}
                </div>
              </div>

              {/* Assigned Client tile */}
              <div
                style={{
                  background: 'var(--bg-card)',
                  border: '1px solid var(--border)',
                  borderRadius: 10,
                  overflow: 'hidden',
                }}
              >
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: '10px 14px',
                    borderBottom: '1px solid var(--border)',
                  }}
                >
                  <span
                    style={{
                      fontSize: 11,
                      fontWeight: 600,
                      letterSpacing: '0.06em',
                      textTransform: 'uppercase',
                      color: 'var(--text-secondary)',
                      fontFamily: 'var(--font-mono)',
                    }}
                  >
                    Assigned Client
                  </span>
                </div>
                <div style={{ padding: '12px 14px' }}>
                  {assignedClient ? (
                    <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                        <Link
                          to={`/clients/${assignedClient.id}`}
                          style={{
                            fontSize: 12,
                            fontFamily: 'var(--font-mono)',
                            color: 'var(--accent)',
                            textDecoration: 'none',
                            fontWeight: 500,
                          }}
                        >
                          {assignedClient.hostname}
                        </Link>
                        <span
                          style={{
                            display: 'inline-flex',
                            alignItems: 'center',
                            gap: 4,
                            fontSize: 10,
                            color:
                              assignedClient.status === 'approved'
                                ? 'var(--signal-healthy)'
                                : 'var(--text-muted)',
                          }}
                        >
                          <span
                            style={{
                              width: 5,
                              height: 5,
                              borderRadius: '50%',
                              background:
                                assignedClient.status === 'approved'
                                  ? 'var(--signal-healthy)'
                                  : 'var(--text-muted)',
                            }}
                          />
                          {assignedClient.status}
                        </span>
                      </div>
                      {usagePct != null && (
                        <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
                          <div
                            style={{
                              display: 'flex',
                              justifyContent: 'space-between',
                              fontSize: 11,
                              fontFamily: 'var(--font-mono)',
                            }}
                          >
                            <span style={{ color: 'var(--text-muted)' }}>Endpoints</span>
                            <span style={{ color: 'var(--text-emphasis)' }}>
                              {license.client_endpoint_count} / {license.max_endpoints}
                            </span>
                          </div>
                          <div
                            style={{
                              width: '100%',
                              borderRadius: 9999,
                              height: 3,
                              background: 'var(--bg-inset)',
                            }}
                          >
                            <div
                              style={{
                                height: 3,
                                borderRadius: 9999,
                                width: `${Math.min(usagePct, 100)}%`,
                                background:
                                  usagePct > 90
                                    ? 'var(--signal-critical)'
                                    : usagePct > 70
                                      ? 'var(--signal-warning)'
                                      : 'var(--signal-healthy)',
                              }}
                            />
                          </div>
                        </div>
                      )}
                    </div>
                  ) : (
                    <p
                      style={{
                        fontSize: 11,
                        color: 'var(--text-muted)',
                        margin: 0,
                        fontFamily: 'var(--font-mono)',
                      }}
                    >
                      Not assigned
                    </p>
                  )}
                </div>
              </div>

              {/* Features tile */}
              <div
                style={{
                  background: 'var(--bg-card)',
                  border: '1px solid var(--border)',
                  borderRadius: 10,
                  overflow: 'hidden',
                }}
              >
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: '10px 14px',
                    borderBottom: '1px solid var(--border)',
                  }}
                >
                  <span
                    style={{
                      fontSize: 11,
                      fontWeight: 600,
                      letterSpacing: '0.06em',
                      textTransform: 'uppercase',
                      color: 'var(--text-secondary)',
                      fontFamily: 'var(--font-mono)',
                    }}
                  >
                    Features
                  </span>
                  <span
                    style={{
                      fontSize: 10,
                      color: 'var(--text-muted)',
                      fontFamily: 'var(--font-mono)',
                    }}
                  >
                    {features.length} included
                  </span>
                </div>
                <div style={{ padding: '12px 14px' }}>
                  <ul
                    style={{
                      listStyle: 'none',
                      padding: 0,
                      margin: 0,
                      display: 'flex',
                      flexDirection: 'column',
                      gap: 6,
                    }}
                  >
                    {features.map((f) => (
                      <li
                        key={f}
                        style={{
                          display: 'flex',
                          alignItems: 'center',
                          gap: 6,
                          fontSize: 11,
                          color: 'var(--text-primary)',
                          fontFamily: 'var(--font-mono)',
                        }}
                      >
                        <CheckCircle
                          style={{
                            width: 11,
                            height: 11,
                            flexShrink: 0,
                            color: 'var(--signal-healthy)',
                          }}
                        />
                        {f}
                      </li>
                    ))}
                  </ul>
                </div>
              </div>
            </div>
          )}

          {activeTab === 'usage' && (
            <div className="space-y-4">
              {usageLoading ? (
                <div
                  className="h-48 animate-pulse rounded"
                  style={{ background: 'var(--bg-canvas)' }}
                />
              ) : !usageData?.points?.length ? (
                <div className="bg-muted/30 rounded-lg p-8 text-center">
                  <p className="text-muted-foreground text-sm">
                    No usage history yet. Data appears after the first catalog sync from an assigned
                    client.
                  </p>
                </div>
              ) : (
                <div>
                  <div className="flex items-center justify-between mb-4">
                    <p className="text-sm font-medium">
                      Endpoint Usage vs License Capacity (last 90 days)
                    </p>
                    <span className="text-xs" style={{ color: 'var(--text-muted)' }}>
                      Max: {chartMax.toLocaleString()} endpoints
                    </span>
                  </div>
                  <svg viewBox="0 0 400 120" className="w-full h-32">
                    {/* Max endpoints capacity dashed line */}
                    <line
                      x1="0"
                      y1="20"
                      x2="400"
                      y2="20"
                      stroke="var(--signal-critical)"
                      strokeDasharray="4 4"
                      strokeWidth="1"
                      opacity="0.6"
                    />
                    <text x="4" y="16" fontSize="8" fill="var(--signal-critical)" opacity="0.8">
                      max
                    </text>
                    {/* Usage polyline */}
                    <polyline
                      fill="none"
                      stroke="var(--accent)"
                      strokeWidth="2"
                      strokeLinejoin="round"
                      points={usageData.points
                        .map((p, i) => {
                          const x = (i / Math.max(usageData.points.length - 1, 1)) * 400;
                          const y = chartMax > 0 ? 110 - (p.endpoints_used / chartMax) * 90 : 110;
                          return `${x},${Math.max(20, Math.min(110, y))}`;
                        })
                        .join(' ')}
                    />
                    {/* Dots at data points */}
                    {usageData.points.map((p, i) => {
                      const x = (i / Math.max(usageData.points.length - 1, 1)) * 400;
                      const y = chartMax > 0 ? 110 - (p.endpoints_used / chartMax) * 90 : 110;
                      return (
                        <circle
                          key={i}
                          cx={x}
                          cy={Math.max(20, Math.min(110, y))}
                          r="2.5"
                          fill="var(--accent)"
                        />
                      );
                    })}
                  </svg>
                  <div
                    className="flex justify-between text-xs mt-1"
                    style={{ color: 'var(--text-muted)' }}
                  >
                    <span>
                      {usageData.points[0]
                        ? new Date(usageData.points[0].date).toLocaleDateString()
                        : ''}
                    </span>
                    <span>
                      {usageData.points[usageData.points.length - 1]
                        ? new Date(
                            usageData.points[usageData.points.length - 1]!.date,
                          ).toLocaleDateString()
                        : ''}
                    </span>
                  </div>
                </div>
              )}
            </div>
          )}

          {activeTab === 'audit' && (
            <div className="space-y-4">
              {auditLoading ? (
                <div className="space-y-3">
                  {Array.from({ length: 3 }, (_, i) => (
                    <div
                      key={i}
                      className="h-16 animate-pulse rounded"
                      style={{ background: 'var(--bg-canvas)' }}
                    />
                  ))}
                </div>
              ) : !auditData?.events?.length ? (
                <div
                  className="rounded-lg p-8 text-center"
                  style={{ background: 'var(--bg-canvas)', border: '1px solid var(--border)' }}
                >
                  <p className="text-sm" style={{ color: 'var(--text-muted)' }}>
                    No audit events recorded yet.
                  </p>
                </div>
              ) : (
                <div className="space-y-3">
                  <p className="text-sm font-medium">
                    License Audit Trail ({auditData.total} events)
                  </p>
                  {auditData.events.map((event, idx) => (
                    <div key={event.id} className="flex gap-3 text-sm">
                      <div className="flex flex-col items-center">
                        <div
                          className="w-2 h-2 rounded-full mt-1.5 flex-shrink-0"
                          style={{ background: 'var(--accent)' }}
                        />
                        {idx < auditData.events.length - 1 && (
                          <div
                            className="w-px flex-1 mt-1"
                            style={{ background: 'var(--border)' }}
                          />
                        )}
                      </div>
                      <div className="pb-4 flex-1 min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          <span className="text-sm font-medium capitalize">
                            {event.event_type.replace(/\./g, ' ')}
                          </span>
                          <span className="text-xs" style={{ color: 'var(--text-muted)' }}>
                            {new Date(event.occurred_at).toLocaleString()}
                          </span>
                        </div>
                        <p className="text-xs mt-0.5" style={{ color: 'var(--text-muted)' }}>
                          {event.actor}
                        </p>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Renewal dialog */}
      <Dialog open={showRenewDialog} onOpenChange={setShowRenewDialog}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>Renew License</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <label className="text-sm font-medium">Tier</label>
              <select
                value={renewForm.tier}
                onChange={(e) => setRenewForm((f) => ({ ...f, tier: e.target.value }))}
                className="w-full mt-1 rounded-md border px-3 py-2 text-sm bg-background"
              >
                <option value="community">Community</option>
                <option value="professional">Professional</option>
                <option value="enterprise">Enterprise</option>
                <option value="msp">MSP</option>
              </select>
            </div>
            <div>
              <label className="text-sm font-medium">Max Endpoints</label>
              <input
                type="number"
                value={renewForm.max_endpoints}
                onChange={(e) =>
                  setRenewForm((f) => ({ ...f, max_endpoints: parseInt(e.target.value) || 0 }))
                }
                className="w-full mt-1 rounded-md border px-3 py-2 text-sm bg-background"
              />
            </div>
            <div>
              <label className="text-sm font-medium">New Expiry Date</label>
              <input
                type="date"
                value={renewForm.expires_at}
                onChange={(e) => setRenewForm((f) => ({ ...f, expires_at: e.target.value }))}
                className="w-full mt-1 rounded-md border px-3 py-2 text-sm bg-background"
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowRenewDialog(false)}>
              Cancel
            </Button>
            <Button
              onClick={() => {
                renewMutation.mutate(
                  {
                    id: license.id,
                    tier: renewForm.tier || undefined,
                    max_endpoints: renewForm.max_endpoints
                      ? Number(renewForm.max_endpoints)
                      : undefined,
                    expires_at: new Date(renewForm.expires_at).toISOString(),
                  },
                  {
                    onSuccess: () => {
                      setShowRenewDialog(false);
                      toast.success('License renewed successfully');
                    },
                    onError: (err) => {
                      toast.error(err instanceof Error ? err.message : 'Failed to renew license');
                    },
                  },
                );
              }}
              disabled={renewMutation.isPending || !renewForm.expires_at}
            >
              {renewMutation.isPending ? 'Renewing...' : 'Renew License'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Revoke dialog */}
      <Dialog open={revokeDialogOpen} onOpenChange={setRevokeDialogOpen}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>Revoke License</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            Are you sure you want to revoke the license for{' '}
            <span className="font-semibold text-foreground">{license.customer_name}</span>? This
            action cannot be undone.
          </p>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRevokeDialogOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              disabled={revokeMutation.isPending}
              onClick={() => {
                revokeMutation.mutate(license.id, {
                  onSuccess: () => {
                    toast.success('License revoked');
                    setRevokeDialogOpen(false);
                    void navigate('/licenses');
                  },
                  onError: (err) => {
                    toast.error(err instanceof Error ? err.message : 'Failed to revoke license');
                    setRevokeDialogOpen(false);
                  },
                });
              }}
            >
              {revokeMutation.isPending ? 'Revoking...' : 'Revoke'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
};
