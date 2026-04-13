export interface CVSSMetric {
  key: string;
  name: string;
  value: string;
  label: string;
  severity: 'critical' | 'high' | 'medium' | 'low' | 'none';
}

const METRIC_NAMES: Record<string, string> = {
  AV: 'Attack Vector (AV)',
  AC: 'Attack Complexity (AC)',
  PR: 'Privileges Required (PR)',
  UI: 'User Interaction (UI)',
  S: 'Scope (S)',
  C: 'Confidentiality Impact (C)',
  I: 'Integrity Impact (I)',
  A: 'Availability Impact (A)',
};

const VALUE_LABELS: Record<string, Record<string, string>> = {
  AV: { N: 'Network', A: 'Adjacent', L: 'Local', P: 'Physical' },
  AC: { L: 'Low', H: 'High' },
  PR: { N: 'None', L: 'Low', H: 'High' },
  UI: { N: 'None', R: 'Required' },
  S: { U: 'Unchanged', C: 'Changed' },
  C: { N: 'None', L: 'Low', H: 'High' },
  I: { N: 'None', L: 'Low', H: 'High' },
  A: { N: 'None', L: 'Low', H: 'High' },
};

const VALUE_SEVERITY: Record<string, Record<string, CVSSMetric['severity']>> = {
  AV: { N: 'critical', A: 'high', L: 'medium', P: 'low' },
  AC: { L: 'critical', H: 'low' },
  PR: { N: 'critical', L: 'medium', H: 'low' },
  UI: { N: 'critical', R: 'low' },
  S: { C: 'high', U: 'low' },
  C: { H: 'critical', L: 'medium', N: 'none' },
  I: { H: 'critical', L: 'medium', N: 'none' },
  A: { H: 'critical', L: 'medium', N: 'none' },
};

export function parseCVSSVector(vector: string | null | undefined): CVSSMetric[] {
  if (!vector) return [];
  const match = vector.match(/^CVSS:3\.[01]\//);
  if (!match) return [];
  const parts = vector.slice(match[0].length).split('/');
  const metrics: CVSSMetric[] = [];
  for (const part of parts) {
    const [key, value] = part.split(':');
    if (!key || !value || !METRIC_NAMES[key]) continue;
    metrics.push({
      key,
      name: METRIC_NAMES[key],
      value,
      label: VALUE_LABELS[key]?.[value] ?? value,
      severity: VALUE_SEVERITY[key]?.[value] ?? 'none',
    });
  }
  return metrics;
}
