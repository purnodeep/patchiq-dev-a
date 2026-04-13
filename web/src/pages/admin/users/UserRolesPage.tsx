import { useState, useMemo } from 'react';
import { useCan } from '../../../app/auth/AuthContext';
import { Plus, Trash2, Users } from 'lucide-react';
import {
  useRoles,
  useUserRoles,
  useAssignUserRole,
  useRevokeUserRole,
  type Role,
} from '../../../api/hooks/useRoles';

const inputStyle: React.CSSProperties = {
  width: '100%',
  height: '40px',
  padding: '0 12px',
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
  const { style, ...rest } = props;
  return (
    <input
      style={{
        ...inputStyle,
        ...(focused ? inputFocusStyle : {}),
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
  fontSize: '12px',
  fontWeight: 500,
  color: 'var(--text-secondary)',
  marginBottom: '6px',
  fontFamily: 'var(--font-sans)',
};

const sectionTitleStyle: React.CSSProperties = {
  fontSize: '13px',
  fontWeight: 600,
  color: 'var(--text-emphasis)',
  fontFamily: 'var(--font-sans)',
  marginBottom: '10px',
};

export function UserRolesPage() {
  const can = useCan();
  const [userId, setUserId] = useState('');
  const [selectedRoleId, setSelectedRoleId] = useState('');
  const [selectFocused, setSelectFocused] = useState(false);

  const trimmedUserId = userId.trim();

  const { data: allRolesData, isLoading: rolesLoading } = useRoles({ limit: 100 });
  const {
    data: userRoles,
    isLoading: userRolesLoading,
    isError: userRolesError,
    refetch: refetchUserRoles,
  } = useUserRoles(trimmedUserId);
  const assignRole = useAssignUserRole();
  const revokeRole = useRevokeUserRole();

  const assignedRoleIds = useMemo(
    () => new Set((userRoles ?? []).map((r: Role) => r.id)),
    [userRoles],
  );

  const availableRoles = useMemo(
    () => (allRolesData?.data ?? []).filter((r) => !assignedRoleIds.has(r.id)),
    [allRolesData?.data, assignedRoleIds],
  );

  const handleAssign = () => {
    if (!trimmedUserId || !selectedRoleId) return;
    assignRole.mutate(
      { userId: trimmedUserId, roleId: selectedRoleId },
      { onSuccess: () => setSelectedRoleId('') },
    );
  };

  const handleRevoke = (roleId: string) => {
    if (!trimmedUserId) return;
    revokeRole.mutate({ userId: trimmedUserId, roleId });
  };

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
      <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            width: '34px',
            height: '34px',
            borderRadius: '8px',
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
          }}
        >
          <Users style={{ width: '16px', height: '16px', color: 'var(--text-muted)' }} />
        </div>
        <h1
          style={{
            fontSize: '22px',
            fontWeight: 600,
            color: 'var(--text-emphasis)',
            letterSpacing: '-0.02em',
            fontFamily: 'var(--font-sans)',
          }}
        >
          User Roles
        </h1>
      </div>

      {/* User ID input */}
      <div
        style={{
          background: 'var(--bg-card)',
          border: '1px solid var(--border)',
          borderRadius: '8px',
          boxShadow: 'var(--shadow-sm)',
          padding: '20px',
          maxWidth: '480px',
        }}
      >
        <h2 style={sectionTitleStyle}>Look up user</h2>
        <div>
          <label htmlFor="user-id-input" style={labelStyle}>
            User ID
          </label>
          <FocusInput
            id="user-id-input"
            placeholder="Enter user ID (UUID)"
            value={userId}
            onChange={(e) => setUserId(e.target.value)}
            style={{ fontFamily: 'var(--font-mono)', fontSize: '12px' }}
          />
          <p
            style={{
              fontSize: '11px',
              color: 'var(--text-faint)',
              marginTop: '6px',
              fontFamily: 'var(--font-sans)',
            }}
          >
            Enter the user ID to manage their role assignments.
          </p>
        </div>
      </div>

      {trimmedUserId && (
        <>
          {/* Role assignment */}
          <div
            style={{
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              borderRadius: '8px',
              boxShadow: 'var(--shadow-sm)',
              padding: '20px',
              maxWidth: '480px',
              display: 'flex',
              flexDirection: 'column',
              gap: '14px',
            }}
          >
            <h2 style={sectionTitleStyle}>Assign role</h2>
            <div style={{ display: 'flex', gap: '8px', alignItems: 'flex-start' }}>
              <div style={{ flex: 1 }}>
                <select
                  value={selectedRoleId}
                  onChange={(e) => setSelectedRoleId(e.target.value)}
                  onFocus={() => setSelectFocused(true)}
                  onBlur={() => setSelectFocused(false)}
                  style={{
                    width: '100%',
                    height: '40px',
                    padding: '0 10px',
                    background: 'var(--bg-card)',
                    border: '1px solid var(--border)',
                    borderRadius: '6px',
                    fontSize: '13px',
                    color: selectedRoleId ? 'var(--text-primary)' : 'var(--text-muted)',
                    fontFamily: 'var(--font-sans)',
                    outline: 'none',
                    cursor: 'pointer',
                    ...(selectFocused
                      ? {
                          borderColor: 'var(--accent)',
                          boxShadow: '0 0 0 2px color-mix(in srgb, var(--accent) 15%, transparent)',
                        }
                      : {}),
                  }}
                >
                  <option value="" disabled>
                    {rolesLoading ? 'Loading roles...' : 'Select a role...'}
                  </option>
                  {availableRoles.map((role) => (
                    <option key={role.id} value={role.id}>
                      {role.name}
                    </option>
                  ))}
                  {!rolesLoading && availableRoles.length === 0 && (
                    <option value="_empty" disabled>
                      No roles available
                    </option>
                  )}
                </select>
              </div>
              <button
                type="button"
                onClick={handleAssign}
                disabled={!selectedRoleId || assignRole.isPending || !can('roles', 'update')}
                title={!can('roles', 'update') ? "You don't have permission" : undefined}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '6px',
                  height: '40px',
                  padding: '0 14px',
                  background:
                    !selectedRoleId || assignRole.isPending || !can('roles', 'update')
                      ? 'color-mix(in srgb, var(--accent) 40%, transparent)'
                      : 'var(--accent)',
                  border: 'none',
                  borderRadius: '6px',
                  fontSize: '13px',
                  fontWeight: 600,
                  color: 'var(--text-on-color, #fff)',
                  cursor:
                    !selectedRoleId || assignRole.isPending || !can('roles', 'update')
                      ? 'not-allowed'
                      : 'pointer',
                  fontFamily: 'var(--font-sans)',
                  whiteSpace: 'nowrap',
                }}
              >
                <Plus style={{ width: '14px', height: '14px' }} />
                Assign
              </button>
            </div>
            {assignRole.isError && (
              <p
                style={{
                  fontSize: '12px',
                  color: 'var(--signal-critical)',
                  fontFamily: 'var(--font-sans)',
                }}
              >
                Failed to assign role: {assignRole.error.message}
              </p>
            )}
          </div>

          {/* Assigned roles */}
          <div style={{ maxWidth: '480px', display: 'flex', flexDirection: 'column', gap: '10px' }}>
            <h2 style={{ ...sectionTitleStyle, marginBottom: 0 }}>
              Assigned roles
              {userRoles && userRoles.length > 0 && (
                <span
                  style={{
                    marginLeft: '8px',
                    fontSize: '12px',
                    fontFamily: 'var(--font-mono)',
                    color: 'var(--text-muted)',
                    fontWeight: 400,
                  }}
                >
                  {userRoles.length}
                </span>
              )}
            </h2>

            {userRolesError ? (
              <div
                style={{
                  borderRadius: '8px',
                  border: '1px solid color-mix(in srgb, var(--signal-critical) 10%, transparent)',
                  background: 'color-mix(in srgb, var(--signal-critical) 10%, transparent)',
                  padding: '12px 14px',
                  fontSize: '13px',
                  color: 'var(--signal-critical)',
                  fontFamily: 'var(--font-sans)',
                }}
              >
                Failed to load user roles.{' '}
                <button
                  onClick={() => refetchUserRoles()}
                  style={{
                    background: 'none',
                    border: 'none',
                    cursor: 'pointer',
                    color: 'var(--signal-critical)',
                    textDecoration: 'underline',
                    fontFamily: 'var(--font-sans)',
                    fontSize: '13px',
                  }}
                >
                  Retry
                </button>
              </div>
            ) : userRolesLoading ? (
              <p
                style={{
                  fontSize: '13px',
                  color: 'var(--text-muted)',
                  fontFamily: 'var(--font-sans)',
                }}
              >
                Loading assigned roles...
              </p>
            ) : !userRoles || userRoles.length === 0 ? (
              <p
                style={{
                  fontSize: '13px',
                  color: 'var(--text-muted)',
                  fontFamily: 'var(--font-sans)',
                }}
              >
                No roles assigned to this user.
              </p>
            ) : (
              <div
                style={{
                  background: 'var(--bg-card)',
                  border: '1px solid var(--border)',
                  borderRadius: '8px',
                  overflow: 'hidden',
                }}
              >
                {userRoles.map((role: Role, idx: number) => (
                  <div
                    key={role.id}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'space-between',
                      padding: '12px 16px',
                      borderBottom: idx < userRoles.length - 1 ? '1px solid var(--border)' : 'none',
                    }}
                  >
                    <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                      <div
                        style={{
                          width: '28px',
                          height: '28px',
                          borderRadius: '6px',
                          background: 'var(--bg-inset)',
                          border: '1px solid var(--border)',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                        }}
                      >
                        <Users
                          style={{ width: '12px', height: '12px', color: 'var(--text-muted)' }}
                        />
                      </div>
                      <div style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
                        <span
                          style={{
                            fontSize: '13px',
                            fontWeight: 500,
                            color: 'var(--text-primary)',
                            fontFamily: 'var(--font-sans)',
                          }}
                        >
                          {role.name}
                        </span>
                        {role.is_system && (
                          <span
                            style={{
                              fontSize: '10px',
                              color: 'var(--text-faint)',
                              fontFamily: 'var(--font-sans)',
                              textTransform: 'uppercase',
                              letterSpacing: '0.05em',
                            }}
                          >
                            System
                          </span>
                        )}
                      </div>
                    </div>
                    <button
                      type="button"
                      onClick={() => handleRevoke(role.id)}
                      disabled={revokeRole.isPending || !can('roles', 'update')}
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        width: '30px',
                        height: '30px',
                        background: 'transparent',
                        border: '1px solid transparent',
                        borderRadius: '6px',
                        cursor:
                          revokeRole.isPending || !can('roles', 'update')
                            ? 'not-allowed'
                            : 'pointer',
                        color: 'var(--signal-critical)',
                        opacity: revokeRole.isPending || !can('roles', 'update') ? 0.5 : 1,
                        transition: 'border-color 0.1s',
                      }}
                      title={!can('roles', 'update') ? "You don't have permission" : 'Revoke role'}
                    >
                      <Trash2 style={{ width: '13px', height: '13px' }} />
                    </button>
                  </div>
                ))}
              </div>
            )}

            {revokeRole.isError && (
              <p
                style={{
                  fontSize: '12px',
                  color: 'var(--signal-critical)',
                  fontFamily: 'var(--font-sans)',
                }}
              >
                Failed to revoke role: {revokeRole.error.message}
              </p>
            )}
          </div>
        </>
      )}
    </div>
  );
}
