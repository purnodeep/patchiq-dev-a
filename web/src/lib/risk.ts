/**
 * Canonical risk score calculation used across all endpoint views.
 * Score: 0-10 scale. Based on CVE severity counts.
 * Formula: (critical×3 + high×2 + medium×1) / 10, capped at 10.
 */
export function computeRiskScore(opts: {
  criticalCves?: number;
  highCves?: number;
  mediumCves?: number;
  cveCount?: number;
  pendingPatches?: number;
}): number {
  // If we have per-severity CVE counts, use the detailed formula
  if (opts.criticalCves !== undefined || opts.highCves !== undefined) {
    const c = opts.criticalCves ?? 0;
    const h = opts.highCves ?? 0;
    const m = opts.mediumCves ?? 0;
    return Math.min(10, Math.round(((c * 3 + h * 2 + m) / 10) * 10) / 10);
  }
  // Fallback: use total CVE count + pending patches
  const cves = opts.cveCount ?? 0;
  const pending = opts.pendingPatches ?? 0;
  return Math.min(10, Math.round(((cves * 3 + pending) / 10) * 10) / 10);
}

export function riskColor(score: number): string {
  if (score >= 7) return 'var(--signal-critical)';
  if (score >= 3) return 'var(--signal-warning)';
  return 'var(--signal-healthy)';
}

export function riskLabel(score: number): string {
  if (score >= 7) return 'Critical Risk';
  if (score >= 3) return 'Medium Risk';
  return 'Low Risk';
}
