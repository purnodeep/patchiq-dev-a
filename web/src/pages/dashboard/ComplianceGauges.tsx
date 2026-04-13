import { RingGauge } from '@patchiq/ui';

export interface ComplianceFramework {
  name: string;
  rate: number;
}

export interface ComplianceGaugesProps {
  complianceRate: number;
  frameworks: ComplianceFramework[];
}

export function ComplianceGauges({ complianceRate, frameworks }: ComplianceGaugesProps) {
  const hasFrameworks = frameworks.length > 0;

  return (
    <div
      className="rounded-lg border p-4"
      style={{
        background: 'var(--bg-card)',
        borderColor: 'var(--border)',
        boxShadow: 'var(--shadow-sm)',
      }}
    >
      <h3
        className="mb-4 text-sm font-semibold"
        style={{ color: 'var(--text-emphasis)', fontFamily: 'var(--font-sans)' }}
      >
        Compliance Frameworks
      </h3>

      <div className="flex flex-col items-center gap-4">
        {hasFrameworks ? (
          <>
            {/* Overall gauge */}
            <RingGauge
              value={Math.round(complianceRate)}
              label="Overall"
              size={100}
              strokeWidth={8}
              colorByValue
            />

            {/* Per-framework gauges */}
            <div className="flex justify-center gap-4">
              {frameworks.map((framework) => (
                <RingGauge
                  key={framework.name}
                  value={framework.rate}
                  label={framework.name}
                  size={64}
                  strokeWidth={5}
                  colorByValue
                />
              ))}
            </div>
          </>
        ) : (
          <div className="flex flex-col items-center gap-2 py-6">
            <RingGauge value={0} size={80} strokeWidth={6} />
            <p className="text-sm" style={{ color: 'var(--text-muted)' }}>
              Not Configured
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
