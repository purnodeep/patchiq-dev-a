import { useCallback, useEffect, useState } from 'react';
import { useCan } from '../../../app/auth/AuthContext';
import { useNavigate, useParams } from 'react-router';
import { Save } from 'lucide-react';
import { Skeleton } from '@patchiq/ui';
import {
  useRole,
  useRolePermissions,
  useCreateRole,
  useUpdateRole,
  type Permission,
} from '../../../api/hooks/useRoles';
import { PermissionDualPanel } from './components/PermissionDualPanel';

const inputStyle: React.CSSProperties = {
  width: '100%',
  height: '36px',
  padding: '0 10px',
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: '6px',
  fontSize: '13px',
  color: 'var(--text-primary)',
  fontFamily: 'var(--font-sans)',
  outline: 'none',
  transition: 'border-color 0.15s, box-shadow 0.15s',
  boxSizing: 'border-box',
};

const inputFocusStyle: React.CSSProperties = {
  borderColor: 'var(--accent)',
  boxShadow: '0 0 0 2px color-mix(in srgb, var(--accent) 15%, transparent)',
};

function FocusInput(props: React.InputHTMLAttributes<HTMLInputElement>) {
  const [focused, setFocused] = useState(false);
  const { style, disabled, ...rest } = props;
  return (
    <input
      disabled={disabled}
      style={{
        ...inputStyle,
        ...(focused && !disabled ? inputFocusStyle : {}),
        ...(disabled ? { opacity: 0.5, cursor: 'not-allowed', background: 'var(--bg-inset)' } : {}),
        ...style,
      }}
      onFocus={() => setFocused(true)}
      onBlur={() => setFocused(false)}
      {...rest}
    />
  );
}

function FocusTextarea(props: React.TextareaHTMLAttributes<HTMLTextAreaElement>) {
  const [focused, setFocused] = useState(false);
  const { style, disabled, ...rest } = props;
  return (
    <textarea
      disabled={disabled}
      style={{
        width: '100%',
        minHeight: '80px',
        padding: '10px 12px',
        background: disabled ? 'var(--bg-inset)' : 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: '6px',
        fontSize: '13px',
        color: 'var(--text-primary)',
        fontFamily: 'var(--font-sans)',
        outline: 'none',
        transition: 'border-color 0.15s, box-shadow 0.15s',
        resize: 'vertical',
        boxSizing: 'border-box',
        cursor: disabled ? 'not-allowed' : 'text',
        opacity: disabled ? 0.5 : 1,
        ...(focused && !disabled ? inputFocusStyle : {}),
        ...style,
      }}
      onFocus={() => setFocused(true)}
      onBlur={() => setFocused(false)}
      {...rest}
    />
  );
}

const labelStyle: React.CSSProperties = {
  display: 'block',
  fontSize: '10px',
  fontWeight: 600,
  textTransform: 'uppercase',
  letterSpacing: '0.06em',
  color: 'var(--text-muted)',
  marginBottom: '6px',
  fontFamily: 'var(--font-mono)',
};

export function RoleEditPage() {
  const can = useCan();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const isNewMode = !id || id === 'new';
  const canSave = isNewMode ? can('roles', 'create') : can('roles', 'update');

  const { data: role, isLoading: isRoleLoading } = useRole(isNewMode ? '' : id);
  const { data: rolePermissions, isLoading: isPermissionsLoading } = useRolePermissions(
    isNewMode ? '' : id,
  );
  const createRole = useCreateRole();
  const updateRole = useUpdateRole();

  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [permissions, setPermissions] = useState<Permission[]>([]);
  const [initialized, setInitialized] = useState(false);

  const isSystem = role?.is_system ?? false;
  const isLoading = !isNewMode && (isRoleLoading || isPermissionsLoading);
  const isSaving = createRole.isPending || updateRole.isPending;

  // Initialize form + permissions when both load (edit mode)
  useEffect(() => {
    if (!isNewMode && role && rolePermissions && !initialized) {
      setName(role.name);
      setDescription(role.description ?? '');
      setPermissions(
        rolePermissions.map((rp) => ({
          resource: rp.resource,
          action: rp.action,
          scope: rp.scope,
        })),
      );
      setInitialized(true);
    }
  }, [isNewMode, role, rolePermissions, initialized]);

  const handleSave = useCallback(async () => {
    if (!name.trim() || isSaving) return;

    const body = {
      name: name.trim(),
      description: description.trim() || undefined,
      permissions,
    };

    try {
      if (isNewMode) {
        const created = await createRole.mutateAsync(body);
        navigate(`/settings/roles/${created.id}/edit`, { replace: true });
      } else {
        await updateRole.mutateAsync({ id: id!, body });
        navigate('/settings/roles');
      }
    } catch {
      // Error state is surfaced via mutation.isError below.
    }
  }, [name, description, permissions, isSaving, isNewMode, id, createRole, updateRole, navigate]);

  const handleBack = useCallback(() => {
    navigate('/settings/roles');
  }, [navigate]);

  if (isLoading) {
    return (
      <div style={{ padding: '24px', display: 'flex', flexDirection: 'column', gap: '24px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <Skeleton className="h-9 w-9 rounded-md" />
          <Skeleton className="h-8 w-48" />
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '12px', maxWidth: '480px' }}>
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
        </div>
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  return (
    <div
      style={{
        padding: '24px',
        background: 'var(--bg-page)',
        minHeight: '100%',
        display: 'flex',
        flexDirection: 'column',
        gap: '24px',
      }}
    >
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
        <div>
          <h1
            style={{
              fontSize: '22px',
              fontWeight: 600,
              color: 'var(--text-emphasis)',
              letterSpacing: '-0.02em',
              fontFamily: 'var(--font-sans)',
            }}
          >
            {isNewMode ? 'New Role' : `Edit Role`}
          </h1>
          {!isNewMode && role?.name && (
            <p
              style={{
                fontSize: '12px',
                color: 'var(--text-muted)',
                fontFamily: 'var(--font-sans)',
                marginTop: '2px',
              }}
            >
              {role.name}
              {isSystem && (
                <span
                  style={{
                    marginLeft: '8px',
                    fontSize: '11px',
                    color: 'var(--text-faint)',
                    fontFamily: 'var(--font-sans)',
                  }}
                >
                  · System role (read-only)
                </span>
              )}
            </p>
          )}
        </div>
      </div>

      {/* Form fields */}
      <div
        style={{
          background: 'var(--bg-card)',
          border: '1px solid var(--border)',
          borderRadius: '8px',
          boxShadow: 'var(--shadow-sm)',
          padding: '20px',
          maxWidth: '520px',
          display: 'flex',
          flexDirection: 'column',
          gap: '16px',
        }}
      >
        <div>
          <label htmlFor="role-name" style={labelStyle}>
            Name <span style={{ color: 'var(--signal-critical)' }}>*</span>
          </label>
          <FocusInput
            id="role-name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g. Patch Reviewer"
            disabled={isSystem}
          />
        </div>

        <div>
          <label htmlFor="role-description" style={labelStyle}>
            Description
          </label>
          <FocusTextarea
            id="role-description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Optional description for this role"
            disabled={isSystem}
          />
        </div>
      </div>

      {/* Permissions */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
        <div style={{ display: 'flex', alignItems: 'baseline', gap: '8px' }}>
          <h2
            style={{
              fontSize: '14px',
              fontWeight: 600,
              color: 'var(--text-emphasis)',
              fontFamily: 'var(--font-sans)',
            }}
          >
            Permissions
          </h2>
          <span
            style={{
              fontSize: '12px',
              color: 'var(--text-muted)',
              fontFamily: 'var(--font-mono)',
            }}
          >
            {permissions.length} granted
          </span>
        </div>
        <PermissionDualPanel
          permissions={permissions}
          onChange={setPermissions}
          disabled={isSystem}
        />
      </div>

      {/* Actions */}
      {!isSystem && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
          {(createRole.isError || updateRole.isError) && (
            <p
              style={{
                fontSize: '13px',
                color: 'var(--signal-critical)',
                fontFamily: 'var(--font-sans)',
              }}
            >
              {(createRole.error ?? updateRole.error)?.message ?? 'Failed to save role'}
            </p>
          )}
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <button
              type="button"
              onClick={handleSave}
              disabled={!name.trim() || isSaving || !canSave}
              title={!canSave ? "You don't have permission" : undefined}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: '6px',
                height: '36px',
                padding: '0 16px',
                background:
                  !name.trim() || isSaving || !canSave
                    ? 'color-mix(in srgb, var(--accent) 40%, transparent)'
                    : 'var(--accent)',
                border: 'none',
                borderRadius: '6px',
                fontSize: '13px',
                fontWeight: 600,
                color: 'var(--text-on-color, #fff)',
                cursor: !name.trim() || isSaving || !canSave ? 'not-allowed' : 'pointer',
                fontFamily: 'var(--font-sans)',
              }}
            >
              <Save style={{ width: '14px', height: '14px' }} />
              {isSaving ? 'Saving...' : 'Save Role'}
            </button>
            <button
              type="button"
              onClick={handleBack}
              style={{
                height: '36px',
                padding: '0 16px',
                background: 'transparent',
                border: '1px solid var(--border)',
                borderRadius: '6px',
                fontSize: '13px',
                fontWeight: 500,
                color: 'var(--text-secondary)',
                cursor: 'pointer',
                fontFamily: 'var(--font-sans)',
              }}
            >
              Cancel
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
