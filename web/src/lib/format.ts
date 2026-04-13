/**
 * Format a deployment ID as a human-readable short identifier.
 * Example: "01HXYZ..." → "D-01HXYZ"
 */
export function formatDeploymentId(id: string): string {
  return `D-${id.slice(0, 6).toUpperCase()}`;
}
