import { useAuth } from '../../app/auth/AuthContext';
import { useOrgDashboard } from '../../api/hooks/useOrganizations';

// MspDashboardPage renders aggregated metrics across all tenants the MSP
// operator has access to within the current organization. Data is served by
// GET /api/v1/organizations/{id}/dashboard.
export function MspDashboardPage() {
  const { user } = useAuth();
  const orgId = user.organization?.id;
  const { data, isLoading, isError } = useOrgDashboard(orgId);

  if (!orgId) {
    return (
      <div className="p-6">
        <h1 className="text-2xl font-semibold">MSP Dashboard</h1>
        <p className="text-muted-foreground mt-2">
          No organization context available for this session.
        </p>
      </div>
    );
  }
  if (isLoading) {
    return <div className="p-6">Loading MSP dashboard…</div>;
  }
  if (isError || !data) {
    return <div className="p-6 text-destructive">Failed to load dashboard data.</div>;
  }

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">MSP Dashboard</h1>
        <p className="text-muted-foreground">
          Aggregated view across {data.total_tenants} tenant(s) in {user.organization?.name}.
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <StatCard label="Total tenants" value={data.total_tenants} />
        <StatCard label="Total endpoints" value={data.total_endpoints} />
        <StatCard label="Organization type" value={user.organization?.type ?? 'direct'} />
      </div>

      <div className="rounded-md border border-border bg-card">
        <div className="border-b border-border px-4 py-2 font-medium">Per-tenant breakdown</div>
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-left text-muted-foreground">
              <th className="px-4 py-2">Tenant</th>
              <th className="px-4 py-2">Slug</th>
              <th className="px-4 py-2 text-right">Endpoints</th>
            </tr>
          </thead>
          <tbody>
            {data.tenants.map((row) => (
              <tr key={row.tenant_id} className="border-b border-border last:border-b-0">
                <td className="px-4 py-2 font-medium">{row.tenant_name}</td>
                <td className="px-4 py-2 text-muted-foreground">{row.tenant_id.slice(0, 8)}…</td>
                <td className="px-4 py-2 text-right">{row.endpoint_count}</td>
              </tr>
            ))}
            {data.tenants.length === 0 && (
              <tr>
                <td colSpan={3} className="px-4 py-6 text-center text-muted-foreground">
                  No tenants accessible in this organization.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function StatCard({ label, value }: { label: string; value: number | string }) {
  return (
    <div className="rounded-md border border-border bg-card p-4">
      <div className="text-xs uppercase tracking-wide text-muted-foreground">{label}</div>
      <div className="mt-2 text-2xl font-semibold">{value}</div>
    </div>
  );
}
