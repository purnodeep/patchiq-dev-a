export type Severity = 'critical' | 'high' | 'medium' | 'low' | 'none';

export const SEVERITY_ORDER: Record<Severity, number> = {
  critical: 0,
  high: 1,
  medium: 2,
  low: 3,
  none: 4,
};
