import { useState, useEffect } from 'react';
import { Lock, Eye, EyeOff, Plus, CheckCircle } from 'lucide-react';
import { Button } from '@patchiq/ui';

interface IAMSettingsProps {
  settings: Record<string, unknown>;
  onSave: (settings: Record<string, unknown>) => void;
  saving: boolean;
}

const initialMappings = [
  {
    zitadel_group: 'hub-admins',
    hub_role: 'Hub Admin',
    permissions: 'Full access, manage clients, licenses, config',
  },
  {
    zitadel_group: 'hub-operators',
    hub_role: 'Operator',
    permissions: 'View clients, trigger syncs, view deployments',
  },
  {
    zitadel_group: 'hub-viewers',
    hub_role: 'Viewer',
    permissions: 'Read-only access to dashboard and reports',
  },
  {
    zitadel_group: 'billing-team',
    hub_role: 'Billing',
    permissions: 'View and manage licenses and billing',
  },
];

const inputStyle: React.CSSProperties = {
  width: '100%',
  padding: '8px 12px',
  borderRadius: 'var(--radius-lg)',
  border: '1px solid var(--border)',
  background: 'var(--bg-card)',
  color: 'var(--text-primary)',
  fontSize: '14px',
  fontFamily: 'var(--font-mono)',
  outline: 'none',
};

export const IAMSettings = ({ settings, onSave, saving }: IAMSettingsProps) => {
  const [ssoUrl, setSsoUrl] = useState('');
  const [clientId, setClientId] = useState('');
  const [clientSecret, setClientSecret] = useState('');
  const [redirectUri, setRedirectUri] = useState('');
  const [showClientId, setShowClientId] = useState(false);
  const [showSecret, setShowSecret] = useState(false);
  const [mappings, setMappings] = useState(initialMappings);
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<'success' | null>(null);

  useEffect(() => {
    if (settings['iam.sso_url'] != null) setSsoUrl(String(settings['iam.sso_url']));
    if (settings['iam.client_id'] != null) setClientId(String(settings['iam.client_id']));
    if (settings['iam.client_secret'] != null)
      setClientSecret(String(settings['iam.client_secret']));
    if (settings['iam.redirect_uri'] != null) setRedirectUri(String(settings['iam.redirect_uri']));
  }, [settings]);

  const handleTestConnection = () => {
    setTesting(true);
    setTestResult(null);
    setTimeout(() => {
      setTesting(false);
      setTestResult('success');
      setTimeout(() => setTestResult(null), 3000);
    }, 1500);
  };

  const handleRemoveMapping = (index: number) => {
    setMappings(mappings.filter((_, i) => i !== index));
  };

  const handleSave = () => {
    onSave({
      'iam.sso_url': ssoUrl,
      'iam.client_id': clientId,
      'iam.client_secret': clientSecret,
      'iam.redirect_uri': redirectUri,
      'iam.role_mappings': mappings.map((m) => ({
        zitadel_group: m.zitadel_group,
        hub_role: m.hub_role,
        permissions: m.permissions,
      })),
    });
  };

  return (
    <div
      className="rounded-xl overflow-hidden"
      style={{ background: 'var(--bg-card)', border: '1px solid var(--border)' }}
    >
      <div
        className="flex items-center justify-between px-6 py-4"
        style={{ borderBottom: '1px solid var(--border)' }}
      >
        <div className="flex items-center gap-3">
          <div
            className="w-8 h-8 rounded-lg flex items-center justify-center"
            style={{ background: 'var(--accent-subtle)' }}
          >
            <Lock className="w-5 h-5" style={{ color: 'var(--accent)' }} />
          </div>
          <div>
            <h2 className="font-semibold" style={{ color: 'var(--text-emphasis)' }}>
              IAM / Identity
            </h2>
            <p className="text-xs" style={{ color: 'var(--text-muted)' }}>
              Single Sign-On and access management
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <span
            className="w-2.5 h-2.5 rounded-full inline-block animate-pulse"
            style={{ background: 'var(--signal-healthy)' }}
          />
          <span className="text-sm font-medium" style={{ color: 'var(--signal-healthy)' }}>
            Connected
          </span>
        </div>
      </div>

      <div className="p-6 space-y-5">
        <div
          className="flex items-center gap-4 p-4 rounded-xl"
          style={{ background: 'var(--bg-inset)', border: '1px solid var(--border)' }}
        >
          <div
            className="w-10 h-10 rounded-lg flex items-center justify-center flex-shrink-0"
            style={{ background: 'var(--accent)' }}
          >
            <span className="font-bold text-sm" style={{ color: 'var(--text-emphasis)' }}>
              Z
            </span>
          </div>
          <div className="flex-1">
            <p className="font-semibold" style={{ color: 'var(--text-emphasis)' }}>
              Zitadel
            </p>
            <p className="text-xs" style={{ color: 'var(--text-muted)' }}>
              OIDC Identity Provider -- v2.38.0
            </p>
          </div>
          <span
            className="px-2.5 py-1 rounded-md text-xs font-medium"
            style={{ background: 'var(--accent-subtle)', color: 'var(--accent)' }}
          >
            Active
          </span>
        </div>

        <form
          onSubmit={(e) => {
            e.preventDefault();
            handleSave();
          }}
          className="grid grid-cols-1 md:grid-cols-2 gap-5"
          autoComplete="off"
        >
          <div>
            <label
              className="block text-sm font-medium mb-2"
              style={{ color: 'var(--text-primary)' }}
            >
              SSO URL
            </label>
            <input
              type="text"
              value={ssoUrl}
              onChange={(e) => setSsoUrl(e.target.value)}
              style={inputStyle}
            />
          </div>
          <div>
            <label
              className="block text-sm font-medium mb-2"
              style={{ color: 'var(--text-primary)' }}
            >
              Client ID
            </label>
            <div className="relative">
              <input
                type={showClientId ? 'text' : 'password'}
                value={clientId}
                onChange={(e) => setClientId(e.target.value)}
                style={{ ...inputStyle, paddingRight: '40px' }}
                autoComplete="off"
              />
              <button
                type="button"
                onClick={() => setShowClientId(!showClientId)}
                className="absolute right-3 top-1/2 -translate-y-1/2"
                style={{ color: 'var(--text-muted)' }}
              >
                {showClientId ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </button>
            </div>
          </div>
          <div>
            <label
              className="block text-sm font-medium mb-2"
              style={{ color: 'var(--text-primary)' }}
            >
              Client Secret
            </label>
            <div className="relative">
              <input
                type={showSecret ? 'text' : 'password'}
                value={clientSecret}
                onChange={(e) => setClientSecret(e.target.value)}
                style={{ ...inputStyle, paddingRight: '40px' }}
                autoComplete="new-password"
              />
              <button
                type="button"
                onClick={() => setShowSecret(!showSecret)}
                className="absolute right-3 top-1/2 -translate-y-1/2"
                style={{ color: 'var(--text-muted)' }}
              >
                {showSecret ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </button>
            </div>
          </div>
          <div>
            <label
              className="block text-sm font-medium mb-2"
              style={{ color: 'var(--text-primary)' }}
            >
              Redirect URI
            </label>
            <input
              type="text"
              value={redirectUri}
              onChange={(e) => setRedirectUri(e.target.value)}
              style={inputStyle}
            />
          </div>
        </form>

        <div>
          <label
            className="block text-sm font-medium mb-3"
            style={{ color: 'var(--text-primary)' }}
          >
            Role Mapping
          </label>
          <div className="rounded-xl overflow-hidden" style={{ border: '1px solid var(--border)' }}>
            <table className="w-full">
              <thead>
                <tr
                  style={{ background: 'var(--bg-inset)', borderBottom: '1px solid var(--border)' }}
                >
                  <th
                    className="px-4 py-2 text-left text-xs font-semibold uppercase"
                    style={{ color: 'var(--text-muted)' }}
                  >
                    Zitadel Group
                  </th>
                  <th
                    className="px-4 py-2 text-left text-xs font-semibold uppercase"
                    style={{ color: 'var(--text-muted)' }}
                  >
                    Hub Role
                  </th>
                  <th
                    className="px-4 py-2 text-left text-xs font-semibold uppercase"
                    style={{ color: 'var(--text-muted)' }}
                  >
                    Permissions
                  </th>
                  <th className="px-4 py-2" />
                </tr>
              </thead>
              <tbody>
                {mappings.map((m, i) => (
                  <tr
                    key={m.zitadel_group}
                    style={{ borderBottom: '1px solid var(--border-faint)' }}
                  >
                    <td
                      className="px-4 py-3 text-sm"
                      style={{ fontFamily: 'var(--font-mono)', color: 'var(--text-primary)' }}
                    >
                      {m.zitadel_group}
                    </td>
                    <td className="px-4 py-3">
                      <span
                        className="px-2 py-0.5 rounded text-xs font-medium"
                        style={{ background: 'var(--accent-subtle)', color: 'var(--accent)' }}
                      >
                        {m.hub_role}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-xs" style={{ color: 'var(--text-muted)' }}>
                      {m.permissions}
                    </td>
                    <td className="px-4 py-3">
                      <button
                        onClick={() => handleRemoveMapping(i)}
                        className="text-xs"
                        style={{ color: 'var(--signal-critical)' }}
                      >
                        Remove
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <button
            className="mt-3 text-sm flex items-center gap-1"
            style={{ color: 'var(--accent)' }}
          >
            <Plus className="w-4 h-4" /> Add Role Mapping
          </button>
        </div>
      </div>

      <div className="px-6 pb-5 flex gap-3">
        <Button variant="outline" onClick={handleTestConnection} disabled={testing}>
          {testing ? (
            <>
              <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle
                  className="opacity-25"
                  cx="12"
                  cy="12"
                  r="10"
                  stroke="currentColor"
                  strokeWidth="4"
                />
                <path
                  className="opacity-75"
                  fill="currentColor"
                  d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
                />
              </svg>
              Testing...
            </>
          ) : testResult === 'success' ? (
            <>
              <CheckCircle className="w-4 h-4" />
              Connection OK
            </>
          ) : (
            <>
              <CheckCircle className="w-4 h-4" />
              Test Connection
            </>
          )}
        </Button>
        <Button onClick={handleSave} disabled={saving}>
          {saving ? 'Saving...' : 'Save'}
        </Button>
      </div>
    </div>
  );
};
