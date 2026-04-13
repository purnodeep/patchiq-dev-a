import { useNavigate } from 'react-router';
import { RingGauge } from '@patchiq/ui';
import { useComplianceSummary } from '@/api/hooks/useCompliance';

interface Framework {
  name: string;
  shortName: string;
  rate: number;
  frameworkId: string;
}

const cardStyle: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: 8,
  boxShadow: 'var(--shadow-sm)',
  transition: 'border-color 150ms ease',
  display: 'flex',
  flexDirection: 'column',
};

const DISPLAY_NAMES: Record<string, { short: string; full: string }> = {
  cis: { short: 'CIS', full: 'CIS Benchmarks' },
  nist: { short: 'NIST', full: 'NIST CSF' },
  pci_dss: { short: 'PCI', full: 'PCI-DSS' },
  pci: { short: 'PCI', full: 'PCI-DSS' },
  hipaa: { short: 'HIPAA', full: 'HIPAA' },
  iso_27001: { short: 'ISO', full: 'ISO 27001' },
  soc_2: { short: 'SOC', full: 'SOC 2' },
};

function getNames(name: string): { short: string; full: string } {
  const lower = name.toLowerCase();
  if (DISPLAY_NAMES[lower]) return DISPLAY_NAMES[lower];
  for (const [key, val] of Object.entries(DISPLAY_NAMES)) {
    if (lower.includes(key)) return val;
  }
  return { short: name.slice(0, 4).toUpperCase(), full: name };
}

function FrameworkRing({ fw, onClick }: { fw: Framework; onClick: () => void }) {
  return (
    <div
      onClick={onClick}
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        gap: 6,
        cursor: 'pointer',
      }}
    >
      <RingGauge value={fw.rate} size={110} strokeWidth={7} />
      <span
        style={{
          fontSize: 11,
          color: 'var(--text-muted)',
          fontFamily: 'var(--font-mono)',
          letterSpacing: '0.06em',
        }}
      >
        {fw.name}
      </span>
    </div>
  );
}

export function ComplianceRings() {
  const { data: complianceData, isLoading } = useComplianceSummary();
  const navigate = useNavigate();

  const fws: Framework[] =
    complianceData?.frameworks && complianceData.frameworks.length > 0
      ? complianceData.frameworks.slice(0, 3).map((f) => {
          const names = getNames(f.name);
          return {
            name: names.full,
            shortName: names.short,
            rate: Math.round(parseFloat(f.score ?? '0')),
            frameworkId: f.framework_id,
          };
        })
      : [];

  return (
    <div
      style={cardStyle}
      onMouseEnter={(e) => {
        (e.currentTarget as HTMLDivElement).style.borderColor = 'var(--text-faint)';
      }}
      onMouseLeave={(e) => {
        (e.currentTarget as HTMLDivElement).style.borderColor = 'var(--border)';
      }}
    >
      <div
        style={{
          padding: '16px 20px 0',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <span style={{ fontSize: 13, color: 'var(--text-muted)', fontWeight: 400 }}>
          Compliance Frameworks
        </span>
      </div>
      <div
        style={{
          padding: '16px 20px 18px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-around',
          flex: 1,
        }}
      >
        {isLoading ? (
          <span style={{ fontSize: 12, color: 'var(--text-faint)' }}>Loading...</span>
        ) : fws.length === 0 ? (
          <span style={{ fontSize: 12, color: 'var(--text-faint)' }}>No frameworks enabled</span>
        ) : (
          fws.map((fw) => (
            <FrameworkRing
              key={fw.shortName}
              fw={fw}
              onClick={() => navigate(`/compliance/frameworks/${fw.frameworkId}`)}
            />
          ))
        )}
      </div>
    </div>
  );
}
