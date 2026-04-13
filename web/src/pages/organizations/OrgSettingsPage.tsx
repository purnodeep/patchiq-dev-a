import {
  Badge,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  PageHeader,
} from '@patchiq/ui';
import { useAuth } from '../../app/auth/AuthContext';

// OrgSettingsPage shows a read-only view of the user's current organization.
// Editing is not yet implemented; this page exists primarily so MSP operators
// have a stable place to confirm scoping and Zitadel binding.
export function OrgSettingsPage() {
  const { user } = useAuth();
  const org = user.organization;

  if (!org) {
    return (
      <div className="p-6 space-y-6">
        <PageHeader
          title="Organization Settings"
          subtitle="Read-only view of your current organization"
        />
        <Card>
          <CardContent className="py-6 text-sm text-muted-foreground">
            No organization context is available for this session.
          </CardContent>
        </Card>
      </div>
    );
  }

  // Zitadel binding is reflected on the AuthContext only when present;
  // we cannot fetch zitadel_org_id without an extra API call, so we report
  // it as "configured" only when the JWT/me endpoint surfaced it later.
  const typeLabel = org.type.toUpperCase();

  return (
    <div className="p-6 space-y-6">
      <PageHeader
        title="Organization Settings"
        subtitle="Read-only view of your current organization"
      />

      <Card>
        <CardHeader>
          <CardTitle>{org.name}</CardTitle>
          <CardDescription>Organization scope for the current session</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <Field label="Name" value={org.name} />
          <Field label="Slug" value={org.slug} mono />
          <div className="flex items-center gap-3">
            <span className="text-xs uppercase tracking-wide text-muted-foreground w-32">
              Type
            </span>
            <Badge variant={org.type === 'msp' ? 'default' : 'secondary'}>{typeLabel}</Badge>
          </div>
          <div className="flex items-center gap-3">
            <span className="text-xs uppercase tracking-wide text-muted-foreground w-32">
              Zitadel binding
            </span>
            <Badge variant="secondary">Managed by IAM admin</Badge>
          </div>
          <Field label="Organization ID" value={org.id} mono />
        </CardContent>
      </Card>
    </div>
  );
}

function Field({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex items-center gap-3">
      <span className="text-xs uppercase tracking-wide text-muted-foreground w-32">{label}</span>
      <span className={mono ? 'font-mono text-sm' : 'text-sm'}>{value}</span>
    </div>
  );
}
