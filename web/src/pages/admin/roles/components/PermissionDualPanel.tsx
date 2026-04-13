import { useCallback, useMemo } from 'react';
import { type Permission } from '@/api/hooks/useRoles';

export interface PermissionDualPanelProps {
  permissions: Permission[];
  onChange: (permissions: Permission[]) => void;
  disabled?: boolean;
}

/** Resource definitions with their valid actions. */
const RESOURCE_ACTIONS: Record<string, readonly string[]> = {
  endpoints: ['read', 'create', 'update', 'delete', 'scan', 'tag'],
  groups: ['read', 'create', 'update', 'delete'],
  patches: ['read', 'sync'],
  policies: ['read', 'create', 'update', 'delete', 'evaluate'],
  deployments: ['read', 'create', 'update', 'delete', 'approve', 'cancel', 'retry'],
  reports: ['read', 'create', 'export'],
  audit: ['read'],
  users: ['read', 'create', 'update', 'delete'],
  roles: ['read', 'create', 'update', 'delete'],
  settings: ['read', 'update'],
} as const;

const RESOURCES = Object.keys(RESOURCE_ACTIONS);

/** Total count of all possible permissions across all resources. */
const TOTAL_PERMISSIONS = RESOURCES.reduce((sum, r) => sum + RESOURCE_ACTIONS[r].length, 0);

const panelStyle: React.CSSProperties = {
  flex: 1,
  display: 'flex',
  flexDirection: 'column',
  minWidth: 0,
};

const panelHeaderStyle: React.CSSProperties = {
  fontSize: '10px',
  fontWeight: 600,
  textTransform: 'uppercase',
  letterSpacing: '0.06em',
  color: 'var(--text-muted)',
  fontFamily: 'var(--font-mono)',
  marginBottom: '8px',
};

const scrollAreaStyle: React.CSSProperties = {
  maxHeight: '280px',
  overflowY: 'auto',
  display: 'flex',
  flexDirection: 'column',
  gap: '2px',
  paddingRight: '2px',
};

export function PermissionDualPanel({
  permissions,
  onChange,
  disabled = false,
}: PermissionDualPanelProps) {
  const permissionSet = useMemo(() => {
    const set = new Set<string>();
    for (const p of permissions) {
      set.add(`${p.resource}:${p.action}`);
    }
    return set;
  }, [permissions]);

  const hasP = useCallback(
    (resource: string, action: string): boolean => permissionSet.has(`${resource}:${action}`),
    [permissionSet],
  );

  const toggleAction = useCallback(
    (resource: string, action: string) => {
      if (disabled) return;
      if (permissionSet.has(`${resource}:${action}`)) {
        onChange(permissions.filter((p) => !(p.resource === resource && p.action === action)));
      } else {
        onChange([...permissions, { resource, action, scope: '*' }]);
      }
    },
    [permissions, onChange, disabled, permissionSet],
  );

  const toggleResource = useCallback(
    (resource: string) => {
      if (disabled) return;
      const validActions = RESOURCE_ACTIONS[resource] ?? [];
      const allChecked = validActions.every((a) => permissionSet.has(`${resource}:${a}`));
      if (allChecked) {
        onChange(permissions.filter((p) => p.resource !== resource));
      } else {
        const existing = permissions.filter((p) => p.resource !== resource);
        const added = validActions.map((a) => ({ resource, action: a, scope: '*' }));
        onChange([...existing, ...added]);
      }
    },
    [permissions, onChange, disabled, permissionSet],
  );

  const moveAll = useCallback(
    (select: boolean) => {
      if (disabled) return;
      if (select) {
        const all: Permission[] = [];
        for (const resource of RESOURCES) {
          for (const action of RESOURCE_ACTIONS[resource]) {
            all.push({ resource, action, scope: '*' });
          }
        }
        onChange(all);
      } else {
        onChange([]);
      }
    },
    [disabled, onChange],
  );

  const removePermission = useCallback(
    (resource: string, action: string) => {
      if (disabled) return;
      onChange(permissions.filter((p) => !(p.resource === resource && p.action === action)));
    },
    [permissions, onChange, disabled],
  );

  const isResourceAllChecked = useCallback(
    (resource: string): boolean => {
      const validActions = RESOURCE_ACTIONS[resource] ?? [];
      return validActions.length > 0 && validActions.every((a) => hasP(resource, a));
    },
    [hasP],
  );

  const isResourceIndeterminate = useCallback(
    (resource: string): boolean => {
      const validActions = RESOURCE_ACTIONS[resource] ?? [];
      const checked = validActions.filter((a) => hasP(resource, a)).length;
      return checked > 0 && checked < validActions.length;
    },
    [hasP],
  );

  const availableCount = TOTAL_PERMISSIONS - permissions.length;

  return (
    <div
      style={{
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: '8px',
        padding: '16px',
        boxShadow: 'var(--shadow-sm)',
      }}
    >
      <div style={{ display: 'flex', gap: '16px' }}>
        {/* Left Panel — Available */}
        <div style={panelStyle}>
          <div style={{ ...panelHeaderStyle, display: 'flex', gap: '6px' }}>
            Available
            <span style={{ color: 'var(--accent)', fontFamily: 'var(--font-mono)' }}>
              ({availableCount})
            </span>
          </div>
          <div style={scrollAreaStyle}>
            {RESOURCES.map((resource) => {
              const validActions = RESOURCE_ACTIONS[resource] ?? [];
              const allChecked = isResourceAllChecked(resource);
              const indeterminate = isResourceIndeterminate(resource);
              return (
                <div key={resource} style={{ marginBottom: '2px' }}>
                  {/* Resource-level row */}
                  <label
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: '8px',
                      borderRadius: '5px',
                      padding: '5px 8px',
                      fontSize: '12px',
                      fontWeight: 600,
                      textTransform: 'capitalize',
                      color: 'var(--text-primary)',
                      fontFamily: 'var(--font-sans)',
                      cursor: disabled ? 'not-allowed' : 'pointer',
                      opacity: disabled ? 0.5 : 1,
                      transition: 'background 0.1s',
                    }}
                    onMouseEnter={(e) => {
                      if (!disabled)
                        (e.currentTarget as HTMLElement).style.background = 'var(--bg-inset)';
                    }}
                    onMouseLeave={(e) => {
                      (e.currentTarget as HTMLElement).style.background = 'transparent';
                    }}
                  >
                    <input
                      type="checkbox"
                      checked={allChecked}
                      ref={(el) => {
                        if (el) el.indeterminate = indeterminate;
                      }}
                      onChange={() => toggleResource(resource)}
                      disabled={disabled}
                      aria-label={`${resource} all`}
                      style={{
                        width: '13px',
                        height: '13px',
                        accentColor: 'var(--accent)',
                        cursor: disabled ? 'not-allowed' : 'pointer',
                        flexShrink: 0,
                      }}
                    />
                    <span>{resource}</span>
                  </label>
                  {/* Action-level rows */}
                  <div
                    style={{
                      marginLeft: '20px',
                      display: 'flex',
                      flexDirection: 'column',
                      gap: '1px',
                    }}
                  >
                    {validActions.map((action) => (
                      <label
                        key={action}
                        style={{
                          display: 'flex',
                          alignItems: 'center',
                          gap: '8px',
                          borderRadius: '4px',
                          padding: '3px 8px',
                          fontSize: '11px',
                          color: hasP(resource, action)
                            ? 'var(--text-faint)'
                            : 'var(--text-secondary)',
                          fontFamily: 'var(--font-sans)',
                          cursor: disabled ? 'not-allowed' : 'pointer',
                          opacity: disabled ? 0.5 : 1,
                          transition: 'background 0.1s',
                        }}
                        onMouseEnter={(e) => {
                          if (!disabled)
                            (e.currentTarget as HTMLElement).style.background = 'var(--bg-inset)';
                        }}
                        onMouseLeave={(e) => {
                          (e.currentTarget as HTMLElement).style.background = 'transparent';
                        }}
                      >
                        <input
                          type="checkbox"
                          checked={hasP(resource, action)}
                          onChange={() => toggleAction(resource, action)}
                          disabled={disabled}
                          aria-label={`${resource} ${action}`}
                          style={{
                            width: '11px',
                            height: '11px',
                            accentColor: 'var(--accent)',
                            cursor: disabled ? 'not-allowed' : 'pointer',
                            flexShrink: 0,
                          }}
                        />
                        <span>{action}</span>
                      </label>
                    ))}
                  </div>
                </div>
              );
            })}
          </div>
        </div>

        {/* Center — Arrow buttons */}
        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            gap: '8px',
            padding: '0 4px',
          }}
        >
          <button
            type="button"
            onClick={() => moveAll(true)}
            disabled={disabled}
            aria-label="Select all permissions"
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              width: '28px',
              height: '28px',
              background: 'transparent',
              border: '1px solid var(--border)',
              borderRadius: '5px',
              fontSize: '16px',
              color: 'var(--text-muted)',
              cursor: disabled ? 'not-allowed' : 'pointer',
              opacity: disabled ? 0.4 : 1,
              transition: 'border-color 0.1s, color 0.1s',
            }}
          >
            ›
          </button>
          <button
            type="button"
            onClick={() => moveAll(false)}
            disabled={disabled}
            aria-label="Deselect all permissions"
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              width: '28px',
              height: '28px',
              background: 'transparent',
              border: '1px solid var(--border)',
              borderRadius: '5px',
              fontSize: '16px',
              color: 'var(--text-muted)',
              cursor: disabled ? 'not-allowed' : 'pointer',
              opacity: disabled ? 0.4 : 1,
              transition: 'border-color 0.1s, color 0.1s',
            }}
          >
            ‹
          </button>
        </div>

        {/* Right Panel — Selected */}
        <div style={panelStyle}>
          <div style={{ ...panelHeaderStyle, display: 'flex', gap: '6px' }}>
            Selected
            <span style={{ color: 'var(--accent)', fontFamily: 'var(--font-mono)' }}>
              ({permissions.length})
            </span>
          </div>
          <div style={scrollAreaStyle}>
            {permissions.length === 0 ? (
              <p
                style={{
                  padding: '16px 8px',
                  textAlign: 'center',
                  fontSize: '12px',
                  color: 'var(--text-faint)',
                  fontFamily: 'var(--font-sans)',
                }}
              >
                No permissions selected
              </p>
            ) : (
              permissions.map((p) => (
                <div
                  key={`${p.resource}:${p.action}`}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    gap: '8px',
                    borderRadius: '5px',
                    border: '1px solid var(--border)',
                    background: 'var(--bg-inset)',
                    padding: '5px 8px',
                    fontSize: '11px',
                  }}
                >
                  <span style={{ color: 'var(--text-primary)', fontFamily: 'var(--font-sans)' }}>
                    {p.action} <span style={{ color: 'var(--text-muted)' }}>{p.resource}</span>
                  </span>
                  <button
                    type="button"
                    onClick={() => removePermission(p.resource, p.action)}
                    disabled={disabled}
                    aria-label={`Remove ${p.action} ${p.resource}`}
                    style={{
                      flexShrink: 0,
                      background: 'none',
                      border: 'none',
                      cursor: disabled ? 'not-allowed' : 'pointer',
                      color: 'var(--text-muted)',
                      fontSize: '14px',
                      lineHeight: 1,
                      padding: 0,
                      opacity: disabled ? 0.4 : 1,
                      transition: 'color 0.1s',
                      display: 'flex',
                    }}
                    onMouseEnter={(e) => {
                      if (!disabled)
                        (e.currentTarget as HTMLElement).style.color = 'var(--signal-critical)';
                    }}
                    onMouseLeave={(e) => {
                      (e.currentTarget as HTMLElement).style.color = 'var(--text-muted)';
                    }}
                  >
                    ×
                  </button>
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
