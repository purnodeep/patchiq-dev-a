import { useState } from 'react';
import {
  Button,
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@patchiq/ui';
import { useGenerateReport } from '../../api/hooks/useReports';

interface GenerateReportDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  defaultType?: string;
  defaultFilters?: Record<string, string>;
}

const REPORT_TYPES = [
  { value: 'endpoints', label: 'Endpoints' },
  { value: 'patches', label: 'Patches' },
  { value: 'cves', label: 'CVEs' },
  { value: 'deployments', label: 'Deployments' },
  { value: 'compliance', label: 'Compliance' },
  { value: 'executive', label: 'Executive' },
];

const FORMATS = ['pdf', 'csv', 'xlsx'] as const;

const selectStyle: React.CSSProperties = {
  background: 'var(--bg-inset)',
  border: '1px solid var(--border)',
  borderRadius: 6,
  padding: '6px 10px',
  fontSize: 12,
  color: 'var(--text-primary)',
  fontFamily: 'var(--font-sans)',
  outline: 'none',
  width: '100%',
};

const labelStyle: React.CSSProperties = {
  fontSize: 11,
  fontWeight: 600,
  fontFamily: 'var(--font-mono)',
  color: 'var(--text-muted)',
  textTransform: 'uppercase',
  letterSpacing: '0.05em',
  marginBottom: 4,
};

export function GenerateReportDialog({
  open,
  onOpenChange,
  defaultType,
  defaultFilters,
}: GenerateReportDialogProps) {
  const [reportType, setReportType] = useState(defaultType ?? 'endpoints');
  const [format, setFormat] = useState<string>('pdf');
  const [filters, setFilters] = useState<Record<string, string>>(defaultFilters ?? {});
  const [error, setError] = useState<string | null>(null);
  const generateMutation = useGenerateReport();

  const handleGenerate = async () => {
    setError(null);
    if (reportType === 'compliance' && !filters.framework) {
      setError('Framework is required for compliance reports');
      return;
    }
    try {
      await generateMutation.mutateAsync({
        report_type: reportType,
        format,
        filters,
      });
      onOpenChange(false);
      setFilters({});
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to generate report');
    }
  };

  const updateFilter = (key: string, value: string) => {
    setFilters((prev) => {
      if (!value) {
        const next = { ...prev };
        delete next[key];
        return next;
      }
      return { ...prev, [key]: value };
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle style={{ fontFamily: 'var(--font-display)' }}>Generate Report</DialogTitle>
        </DialogHeader>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 14, padding: '4px 0' }}>
          {/* Report Type */}
          <div>
            <div style={labelStyle}>Report Type</div>
            <select
              value={reportType}
              onChange={(e) => {
                setReportType(e.target.value);
                setFilters({});
                setError(null);
              }}
              style={selectStyle}
            >
              {REPORT_TYPES.map((t) => (
                <option key={t.value} value={t.value}>
                  {t.label}
                </option>
              ))}
            </select>
          </div>

          {/* Format */}
          <div>
            <div style={labelStyle}>Format</div>
            <div style={{ display: 'flex', gap: 4 }}>
              {FORMATS.map((f) => (
                <button
                  key={f}
                  type="button"
                  onClick={() => setFormat(f)}
                  style={{
                    padding: '4px 12px',
                    borderRadius: 4,
                    fontSize: 11,
                    fontWeight: 600,
                    fontFamily: 'var(--font-mono)',
                    textTransform: 'uppercase',
                    cursor: 'pointer',
                    border: `1px solid ${format === f ? 'var(--accent)' : 'var(--border)'}`,
                    background:
                      format === f
                        ? 'color-mix(in srgb, var(--accent) 10%, transparent)'
                        : 'var(--bg-inset)',
                    color: format === f ? 'var(--accent)' : 'var(--text-muted)',
                    transition: 'all 0.15s',
                  }}
                >
                  {f}
                </button>
              ))}
            </div>
          </div>

          {/* Type-specific filters */}
          {reportType === 'endpoints' && (
            <>
              <div>
                <div style={labelStyle}>Status</div>
                <select
                  value={filters.status ?? ''}
                  onChange={(e) => updateFilter('status', e.target.value)}
                  style={selectStyle}
                >
                  <option value="">All</option>
                  <option value="online">Online</option>
                  <option value="offline">Offline</option>
                  <option value="stale">Stale</option>
                </select>
              </div>
              <div>
                <div style={labelStyle}>OS Family</div>
                <select
                  value={filters.os_family ?? ''}
                  onChange={(e) => updateFilter('os_family', e.target.value)}
                  style={selectStyle}
                >
                  <option value="">All</option>
                  <option value="windows">Windows</option>
                  <option value="linux">Linux</option>
                  <option value="macos">macOS</option>
                </select>
              </div>
            </>
          )}

          {reportType === 'patches' && (
            <>
              <div>
                <div style={labelStyle}>Severity</div>
                <select
                  value={filters.severity ?? ''}
                  onChange={(e) => updateFilter('severity', e.target.value)}
                  style={selectStyle}
                >
                  <option value="">All</option>
                  <option value="critical">Critical</option>
                  <option value="high">High</option>
                  <option value="medium">Medium</option>
                  <option value="low">Low</option>
                </select>
              </div>
              <div>
                <div style={labelStyle}>OS Family</div>
                <select
                  value={filters.os_family ?? ''}
                  onChange={(e) => updateFilter('os_family', e.target.value)}
                  style={selectStyle}
                >
                  <option value="">All</option>
                  <option value="windows">Windows</option>
                  <option value="linux">Linux</option>
                  <option value="macos">macOS</option>
                </select>
              </div>
            </>
          )}

          {reportType === 'cves' && (
            <>
              <div>
                <div style={labelStyle}>Severity</div>
                <select
                  value={filters.severity ?? ''}
                  onChange={(e) => updateFilter('severity', e.target.value)}
                  style={selectStyle}
                >
                  <option value="">All</option>
                  <option value="critical">Critical</option>
                  <option value="high">High</option>
                  <option value="medium">Medium</option>
                  <option value="low">Low</option>
                </select>
              </div>
              <div>
                <div style={labelStyle}>Exploit Available</div>
                <select
                  value={filters.exploit_available ?? ''}
                  onChange={(e) => updateFilter('exploit_available', e.target.value)}
                  style={selectStyle}
                >
                  <option value="">All</option>
                  <option value="true">Yes</option>
                  <option value="false">No</option>
                </select>
              </div>
              <div>
                <div style={labelStyle}>CISA KEV</div>
                <select
                  value={filters.cisa_kev ?? ''}
                  onChange={(e) => updateFilter('cisa_kev', e.target.value)}
                  style={selectStyle}
                >
                  <option value="">All</option>
                  <option value="true">Yes</option>
                  <option value="false">No</option>
                </select>
              </div>
            </>
          )}

          {reportType === 'deployments' && (
            <>
              <div>
                <div style={labelStyle}>Status</div>
                <select
                  value={filters.status ?? ''}
                  onChange={(e) => updateFilter('status', e.target.value)}
                  style={selectStyle}
                >
                  <option value="">All</option>
                  <option value="running">Running</option>
                  <option value="completed">Completed</option>
                  <option value="failed">Failed</option>
                </select>
              </div>
              <div>
                <div style={labelStyle}>Date Range</div>
                <div style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
                  <input
                    type="date"
                    value={filters.from ?? ''}
                    onChange={(e) => updateFilter('from', e.target.value)}
                    style={{ ...selectStyle, width: 'auto', flex: 1 }}
                  />
                  <span style={{ color: 'var(--text-muted)', fontSize: 11 }}>&mdash;</span>
                  <input
                    type="date"
                    value={filters.to ?? ''}
                    onChange={(e) => updateFilter('to', e.target.value)}
                    style={{ ...selectStyle, width: 'auto', flex: 1 }}
                  />
                </div>
              </div>
            </>
          )}

          {reportType === 'compliance' && (
            <div>
              <div style={labelStyle}>
                Framework <span style={{ color: 'var(--signal-critical)' }}>*</span>
              </div>
              <select
                value={filters.framework ?? ''}
                onChange={(e) => updateFilter('framework', e.target.value)}
                style={selectStyle}
              >
                <option value="">Select framework...</option>
                <option value="cis">CIS</option>
                <option value="pci-dss">PCI-DSS</option>
                <option value="hipaa">HIPAA</option>
                <option value="nist">NIST</option>
                <option value="iso-27001">ISO 27001</option>
                <option value="soc-2">SOC 2</option>
              </select>
            </div>
          )}

          {reportType === 'executive' && (
            <div>
              <div style={labelStyle}>Date Range</div>
              <div style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
                <input
                  type="date"
                  value={filters.from ?? ''}
                  onChange={(e) => updateFilter('from', e.target.value)}
                  style={{ ...selectStyle, width: 'auto', flex: 1 }}
                />
                <span style={{ color: 'var(--text-muted)', fontSize: 11 }}>&mdash;</span>
                <input
                  type="date"
                  value={filters.to ?? ''}
                  onChange={(e) => updateFilter('to', e.target.value)}
                  style={{ ...selectStyle, width: 'auto', flex: 1 }}
                />
              </div>
            </div>
          )}

          {error && (
            <div style={{ fontSize: 12, color: 'var(--signal-critical)', padding: '4px 0' }}>
              {error}
            </div>
          )}
        </div>

        <DialogFooter className="gap-2">
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={generateMutation.isPending}
          >
            Cancel
          </Button>
          <Button onClick={handleGenerate} disabled={generateMutation.isPending}>
            {generateMutation.isPending ? 'Generating...' : 'Generate'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
