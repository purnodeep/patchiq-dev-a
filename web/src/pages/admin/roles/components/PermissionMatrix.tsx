import { useCallback, useMemo } from 'react';
import { cn } from '@patchiq/ui';
import { type Permission } from '@/api/hooks/useRoles';

export interface PermissionMatrixProps {
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

/** Union of all possible actions across all resources. */
const ALL_ACTIONS = [...new Set(RESOURCES.flatMap((r) => [...RESOURCE_ACTIONS[r]]))];

function isValidAction(resource: string, action: string): boolean {
  return RESOURCE_ACTIONS[resource]?.includes(action) ?? false;
}

export function PermissionMatrix({
  permissions,
  onChange,
  disabled = false,
}: PermissionMatrixProps) {
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

  // Scope is always '*' (tenant-wide).
  const togglePermission = useCallback(
    (resource: string, action: string) => {
      if (disabled) return;
      const exists = permissionSet.has(`${resource}:${action}`);
      if (exists) {
        onChange(permissions.filter((p) => !(p.resource === resource && p.action === action)));
      } else {
        onChange([...permissions, { resource, action, scope: '*' }]);
      }
    },
    [permissions, onChange, disabled, permissionSet],
  );

  const toggleRow = useCallback(
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

  const toggleColumn = useCallback(
    (action: string) => {
      if (disabled) return;
      const applicableResources = RESOURCES.filter((r) => isValidAction(r, action));
      const allChecked = applicableResources.every((r) => permissionSet.has(`${r}:${action}`));
      if (allChecked) {
        onChange(
          permissions.filter((p) => !(p.action === action && isValidAction(p.resource, action))),
        );
      } else {
        const withoutColumn = permissions.filter(
          (p) => !(p.action === action && applicableResources.includes(p.resource)),
        );
        const added = applicableResources.map((r) => ({ resource: r, action, scope: '*' }));
        onChange([...withoutColumn, ...added]);
      }
    },
    [permissions, onChange, disabled],
  );

  const isRowAllChecked = useCallback(
    (resource: string): boolean => {
      const validActions = RESOURCE_ACTIONS[resource] ?? [];
      return validActions.length > 0 && validActions.every((a) => hasP(resource, a));
    },
    [hasP],
  );

  const isRowIndeterminate = useCallback(
    (resource: string): boolean => {
      const validActions = RESOURCE_ACTIONS[resource] ?? [];
      const checked = validActions.filter((a) => hasP(resource, a)).length;
      return checked > 0 && checked < validActions.length;
    },
    [hasP],
  );

  const isColumnAllChecked = useCallback(
    (action: string): boolean => {
      const applicable = RESOURCES.filter((r) => isValidAction(r, action));
      return applicable.length > 0 && applicable.every((r) => hasP(r, action));
    },
    [hasP],
  );

  const isColumnIndeterminate = useCallback(
    (action: string): boolean => {
      const applicable = RESOURCES.filter((r) => isValidAction(r, action));
      const checked = applicable.filter((r) => hasP(r, action)).length;
      return checked > 0 && checked < applicable.length;
    },
    [hasP],
  );

  return (
    <div className="relative w-full overflow-auto rounded-lg border border-border">
      <table className="w-full caption-bottom text-sm">
        <thead className="border-b bg-muted/50">
          <tr>
            <th className="h-10 px-4 text-left align-middle font-medium text-muted-foreground">
              Resource
            </th>
            {ALL_ACTIONS.map((action) => (
              <th
                key={action}
                className={cn(
                  'h-10 px-3 text-center align-middle font-medium text-muted-foreground',
                  !disabled && 'cursor-pointer select-none hover:text-foreground',
                )}
                onClick={() => toggleColumn(action)}
              >
                {action}
              </th>
            ))}
            <th
              className={cn(
                'h-10 px-3 text-center align-middle font-medium text-muted-foreground',
                !disabled && 'cursor-pointer select-none hover:text-foreground',
              )}
            >
              All
            </th>
          </tr>
        </thead>
        <tbody>
          {RESOURCES.map((resource) => (
            <tr
              key={resource}
              className="border-b transition-colors hover:bg-muted/50 dark:hover:bg-muted/20"
            >
              <td className="px-4 py-3 align-middle font-medium capitalize">{resource}</td>
              {ALL_ACTIONS.map((action) => (
                <td key={action} className="px-3 py-3 text-center align-middle">
                  {isValidAction(resource, action) ? (
                    <input
                      type="checkbox"
                      checked={hasP(resource, action)}
                      onChange={() => togglePermission(resource, action)}
                      disabled={disabled}
                      aria-label={`${resource} ${action}`}
                      className={cn(
                        'h-4 w-4 rounded border border-primary/50 accent-primary',
                        disabled && 'cursor-not-allowed opacity-50',
                      )}
                    />
                  ) : (
                    <span
                      className="text-muted-foreground/50"
                      aria-label={`${resource} ${action} not available`}
                    >
                      —
                    </span>
                  )}
                </td>
              ))}
              <td className="px-3 py-3 text-center align-middle">
                <input
                  type="checkbox"
                  checked={isRowAllChecked(resource)}
                  ref={(el) => {
                    if (el) el.indeterminate = isRowIndeterminate(resource);
                  }}
                  onChange={() => toggleRow(resource)}
                  disabled={disabled}
                  aria-label={`${resource} all`}
                  className={cn(
                    'h-4 w-4 rounded border border-primary/50 accent-primary',
                    disabled && 'cursor-not-allowed opacity-50',
                  )}
                />
              </td>
            </tr>
          ))}
        </tbody>
        <tfoot className="border-t bg-muted/30">
          <tr>
            <td className="px-4 py-2 align-middle text-sm font-medium text-muted-foreground">
              Column
            </td>
            {ALL_ACTIONS.map((action) => (
              <td key={action} className="px-3 py-2 text-center align-middle">
                <input
                  type="checkbox"
                  checked={isColumnAllChecked(action)}
                  ref={(el) => {
                    if (el) el.indeterminate = isColumnIndeterminate(action);
                  }}
                  onChange={() => toggleColumn(action)}
                  disabled={disabled}
                  aria-label={`all ${action}`}
                  className={cn(
                    'h-4 w-4 rounded border border-primary/50 accent-primary',
                    disabled && 'cursor-not-allowed opacity-50',
                  )}
                />
              </td>
            ))}
            <td />
          </tr>
        </tfoot>
      </table>
    </div>
  );
}
