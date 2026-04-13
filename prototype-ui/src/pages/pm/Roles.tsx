import { useState } from 'react';
import { motion } from 'framer-motion';
import { Plus, Edit2, Trash2, X } from 'lucide-react';
import { useHotkeys } from '@/hooks/useHotkeys';
import { GlassCard } from '@/components/shared/GlassCard';
import { SectionHeader } from '@/components/shared/SectionHeader';

// ── Framer Motion variants ────────────────────────────────────────────────────
const stagger = {
  hidden: {},
  show: { transition: { staggerChildren: 0.06 } },
};

const fadeUp = {
  hidden: { opacity: 0, y: 12 },
  show: { opacity: 1, y: 0, transition: { duration: 0.4, ease: 'easeOut' } },
};

// ── Types ─────────────────────────────────────────────────────────────────────
type RoleType = 'system' | 'custom';

interface AssignedUser {
  id: string;
  name: string;
  color: string;
}

interface Role {
  name: string;
  icon: string;
  iconColor: string;
  description: string;
  users: number;
  permissions: number;
  type: RoleType;
  assignedUsers: AssignedUser[];
}

type Permission = Record<string, string[]>;

// ── Initial Data ──────────────────────────────────────────────────────────────
const INITIAL_ROLES: Role[] = [
  {
    name: 'Administrator',
    icon: '🔑',
    iconColor: 'linear-gradient(135deg, var(--color-danger), #dc2626)',
    description: 'Full access to all resources',
    users: 2,
    permissions: 48,
    type: 'system',
    assignedUsers: [
      { id: 'JD', name: 'admin@acme.com', color: 'linear-gradient(135deg,#3b82f6,#1d4ed8)' },
      { id: 'SW', name: 'sandy@acme.com', color: 'linear-gradient(135deg,#10b981,#059669)' },
    ],
  },
  {
    name: 'Operator',
    icon: '⚙️',
    iconColor: 'linear-gradient(135deg, var(--color-primary), #1d4ed8)',
    description: 'Deploy and manage patches/endpoints',
    users: 5,
    permissions: 32,
    type: 'system',
    assignedUsers: [
      { id: 'RK', name: 'rishab@acme.com', color: 'linear-gradient(135deg,#f59e0b,#d97706)' },
      { id: 'DN', name: 'danish@acme.com', color: 'linear-gradient(135deg,#6366f1,#4338ca)' },
      { id: 'M1', name: 'marco@acme.com', color: 'linear-gradient(135deg,#8b5cf6,#6d28d9)' },
      { id: 'L1', name: 'leena@acme.com', color: 'linear-gradient(135deg,#ec4899,#be185d)' },
      { id: 'K1', name: 'kiran@acme.com', color: 'linear-gradient(135deg,#14b8a6,#0f766e)' },
    ],
  },
  {
    name: 'Analyst',
    icon: '📊',
    iconColor: 'linear-gradient(135deg, var(--color-success), #059669)',
    description: 'View dashboards and run evaluations',
    users: 8,
    permissions: 18,
    type: 'system',
    assignedUsers: [
      { id: 'A1', name: 'alice@acme.com', color: 'linear-gradient(135deg,#3b82f6,#1d4ed8)' },
      { id: 'B1', name: 'bob@acme.com', color: 'linear-gradient(135deg,#f59e0b,#d97706)' },
      { id: 'C1', name: 'charlie@acme.com', color: 'linear-gradient(135deg,#10b981,#059669)' },
    ],
  },
  {
    name: 'Read Only',
    icon: '👁',
    iconColor: 'linear-gradient(135deg, var(--color-muted), #475569)',
    description: 'View-only access to all resources',
    users: 12,
    permissions: 12,
    type: 'system',
    assignedUsers: [
      { id: 'D1', name: 'dave@acme.com', color: 'linear-gradient(135deg,#8b5cf6,#6d28d9)' },
      { id: 'E1', name: 'eve@acme.com', color: 'linear-gradient(135deg,#ec4899,#be185d)' },
    ],
  },
  {
    name: 'Compliance Auditor',
    icon: '🛡',
    iconColor: 'linear-gradient(135deg, var(--color-warning), #d97706)',
    description: 'Compliance-focused: evaluate, report, export',
    users: 3,
    permissions: 20,
    type: 'custom',
    assignedUsers: [
      { id: 'F1', name: 'frank@acme.com', color: 'linear-gradient(135deg,#6366f1,#4338ca)' },
      { id: 'G1', name: 'grace@acme.com', color: 'linear-gradient(135deg,#14b8a6,#0f766e)' },
      { id: 'H1', name: 'henry@acme.com', color: 'linear-gradient(135deg,#f97316,#c2410c)' },
    ],
  },
];

// ── Permission matrix data ─────────────────────────────────────────────────────
const RESOURCES = [
  'Endpoints',
  'Patches',
  'CVEs',
  'Policies',
  'Deployments',
  'Workflows',
  'Compliance',
  'Audit',
  'Settings',
  'Roles',
];

const ACTIONS = ['View', 'Create', 'Update', 'Delete', 'Deploy', 'Export'];

const INITIAL_ROLE_PERMISSIONS: Record<string, Permission> = {
  Operator: {
    Endpoints: ['View', 'Create', 'Update', 'Deploy'],
    Patches: ['View', 'Create', 'Update'],
    CVEs: ['View', 'Create', 'Update'],
    Policies: ['View', 'Create', 'Update'],
    Deployments: ['View', 'Create', 'Update', 'Deploy'],
    Workflows: ['View', 'Create', 'Update'],
    Compliance: ['View', 'Export'],
    Audit: ['View'],
    Settings: ['View'],
    Roles: ['View'],
  },
  Administrator: {
    Endpoints: ['View', 'Create', 'Update', 'Delete', 'Deploy', 'Export'],
    Patches: ['View', 'Create', 'Update', 'Delete', 'Deploy', 'Export'],
    CVEs: ['View', 'Create', 'Update', 'Delete', 'Export'],
    Policies: ['View', 'Create', 'Update', 'Delete', 'Export'],
    Deployments: ['View', 'Create', 'Update', 'Delete', 'Deploy', 'Export'],
    Workflows: ['View', 'Create', 'Update', 'Delete', 'Export'],
    Compliance: ['View', 'Create', 'Update', 'Delete', 'Export'],
    Audit: ['View', 'Export'],
    Settings: ['View', 'Create', 'Update', 'Delete'],
    Roles: ['View', 'Create', 'Update', 'Delete', 'Export'],
  },
  Analyst: {
    Endpoints: ['View'],
    Patches: ['View'],
    CVEs: ['View', 'Export'],
    Policies: ['View'],
    Deployments: ['View'],
    Workflows: ['View'],
    Compliance: ['View', 'Export'],
    Audit: ['View', 'Export'],
    Settings: ['View'],
    Roles: ['View'],
  },
  'Read Only': {
    Endpoints: ['View'],
    Patches: ['View'],
    CVEs: ['View'],
    Policies: ['View'],
    Deployments: ['View'],
    Workflows: ['View'],
    Compliance: ['View'],
    Audit: ['View'],
    Settings: ['View'],
    Roles: ['View'],
  },
  'Compliance Auditor': {
    Endpoints: ['View'],
    Patches: ['View'],
    CVEs: ['View', 'Export'],
    Policies: ['View'],
    Deployments: ['View'],
    Workflows: ['View'],
    Compliance: ['View', 'Create', 'Update', 'Export'],
    Audit: ['View', 'Export'],
    Settings: ['View'],
    Roles: ['View'],
  },
};

// Helper: count permissions from a Permission record
function countPermissions(perms: Permission): number {
  return Object.values(perms).reduce((total, actions) => total + actions.length, 0);
}

// ── Sub-components ────────────────────────────────────────────────────────────
function TypeBadge({ type }: { type: 'system' | 'custom' }) {
  const isCustom = type === 'custom';
  return (
    <span
      style={{
        fontSize: 10,
        fontWeight: 700,
        padding: '2px 8px',
        borderRadius: 20,
        border: `1px solid ${isCustom ? 'var(--color-cyan)' : 'var(--color-separator)'}`,
        color: isCustom ? 'var(--color-cyan)' : 'var(--color-muted)',
        textTransform: 'uppercase',
        letterSpacing: '0.06em',
      }}
    >
      {type}
    </span>
  );
}

// ── Role modal (Create / Edit) ────────────────────────────────────────────────
interface RoleModalProps {
  mode: 'create' | 'edit';
  initialName?: string;
  initialDesc?: string;
  initialPermissions?: string[];
  onSave: (name: string, desc: string, permissions: string[]) => void;
  onClose: () => void;
}

function RoleModal({
  mode,
  initialName = '',
  initialDesc = '',
  initialPermissions = [],
  onSave,
  onClose,
}: RoleModalProps) {
  const [roleName, setRoleName] = useState(initialName);
  const [roleDesc, setRoleDesc] = useState(initialDesc);
  const [selectedPermissions, setSelectedPermissions] = useState<string[]>(initialPermissions);
  const [availableChecked, setAvailableChecked] = useState<string[]>([]);

  const totalAvailable = RESOURCES.length * ACTIONS.length;
  const availableCount = totalAvailable - selectedPermissions.length;

  function toggleAvailableCheck(permKey: string) {
    setAvailableChecked((prev) =>
      prev.includes(permKey) ? prev.filter((p) => p !== permKey) : [...prev, permKey],
    );
  }

  function toggleResourceCheck(resource: string) {
    const resourcePerms = ACTIONS.map((a) => `${resource}:${a}`).filter(
      (p) => !selectedPermissions.includes(p),
    );
    const allChecked = resourcePerms.every((p) => availableChecked.includes(p));
    if (allChecked) {
      setAvailableChecked((prev) => prev.filter((p) => !resourcePerms.includes(p)));
    } else {
      setAvailableChecked((prev) => [...new Set([...prev, ...resourcePerms])]);
    }
  }

  function moveToSelected() {
    const toAdd = availableChecked.filter((p) => !selectedPermissions.includes(p));
    setSelectedPermissions((prev) => [...prev, ...toAdd]);
    setAvailableChecked([]);
  }

  function removeFromSelected() {
    setSelectedPermissions((prev) => prev.filter((p) => !availableChecked.includes(p)));
    setAvailableChecked([]);
  }

  function removePermPill(permKey: string) {
    setSelectedPermissions((prev) => prev.filter((p) => p !== permKey));
  }

  function handleReset() {
    setRoleName(mode === 'create' ? '' : initialName);
    setRoleDesc(mode === 'create' ? '' : initialDesc);
    setSelectedPermissions(mode === 'create' ? [] : initialPermissions);
    setAvailableChecked([]);
  }

  function handleSave() {
    if (!roleName.trim()) return;
    onSave(roleName.trim(), roleDesc.trim(), selectedPermissions);
  }

  const inputStyle: React.CSSProperties = {
    width: '100%',
    background: 'color-mix(in srgb, var(--color-foreground) 5%, transparent)',
    border: '1px solid var(--color-separator)',
    borderRadius: 6,
    padding: '8px 12px',
    fontSize: 12,
    color: 'var(--color-foreground)',
    outline: 'none',
    boxSizing: 'border-box',
  };

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0,0,0,0.6)',
        backdropFilter: 'blur(4px)',
        zIndex: 50,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        overflow: 'auto',
        padding: 20,
      }}
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div
        style={{
          background: 'var(--color-card)',
          border: '1px solid var(--color-separator)',
          borderRadius: 12,
          padding: 24,
          width: 800,
          maxWidth: '95vw',
          maxHeight: '80vh',
          overflowY: 'auto',
          margin: 'auto',
        }}
      >
        {/* Modal header */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            marginBottom: 20,
          }}
        >
          <div style={{ fontSize: 18, fontWeight: 700, color: 'var(--color-foreground)' }}>
            {mode === 'create' ? 'Create Role' : 'Edit Role'}
          </div>
          <button
            onClick={onClose}
            style={{
              background: 'none',
              border: 'none',
              color: 'var(--color-muted)',
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
            }}
          >
            <X size={18} />
          </button>
        </div>

        {/* Name */}
        <div style={{ marginBottom: 14 }}>
          <label
            style={{
              fontSize: 12,
              fontWeight: 600,
              color: 'var(--color-foreground)',
              display: 'block',
              marginBottom: 6,
            }}
          >
            <span style={{ color: 'var(--color-danger)' }}>*</span> Name
          </label>
          <input
            style={inputStyle}
            type="text"
            placeholder="Role name"
            value={roleName}
            onChange={(e) => setRoleName(e.target.value)}
          />
        </div>

        {/* Description */}
        <div style={{ marginBottom: 16 }}>
          <label
            style={{
              fontSize: 12,
              fontWeight: 600,
              color: 'var(--color-foreground)',
              display: 'block',
              marginBottom: 6,
            }}
          >
            Description
          </label>
          <textarea
            style={{ ...inputStyle, resize: 'vertical', minHeight: 70 }}
            placeholder="Optional description"
            value={roleDesc}
            onChange={(e) => setRoleDesc(e.target.value)}
          />
        </div>

        {/* Permissions dual-pane */}
        <div style={{ marginBottom: 16 }}>
          <label
            style={{
              fontSize: 12,
              fontWeight: 600,
              color: 'var(--color-foreground)',
              display: 'block',
              marginBottom: 10,
            }}
          >
            Permissions
          </label>
          <div style={{ display: 'flex', gap: 8, alignItems: 'stretch' }}>
            {/* Left pane: Available */}
            <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
              <div
                style={{
                  fontSize: 11,
                  fontWeight: 600,
                  color: 'var(--color-muted)',
                  textTransform: 'uppercase',
                  letterSpacing: '0.06em',
                  marginBottom: 6,
                }}
              >
                Available <span style={{ color: 'var(--color-primary)' }}>({availableCount})</span>
              </div>
              <div
                style={{
                  height: 240,
                  overflowY: 'auto',
                  border: '1px solid var(--color-separator)',
                  borderRadius: 8,
                  padding: 8,
                }}
              >
                {RESOURCES.map((resource) => {
                  const resourcePerms = ACTIONS.map((a) => `${resource}:${a}`).filter(
                    (p) => !selectedPermissions.includes(p),
                  );
                  if (resourcePerms.length === 0) return null;
                  const allChecked = resourcePerms.every((p) => availableChecked.includes(p));
                  const someChecked = resourcePerms.some((p) => availableChecked.includes(p));
                  return (
                    <div key={resource} style={{ marginBottom: 6 }}>
                      <label
                        style={{
                          display: 'flex',
                          alignItems: 'center',
                          gap: 8,
                          padding: '5px 4px',
                          borderRadius: 4,
                          cursor: 'pointer',
                          fontSize: 11,
                          fontWeight: 600,
                          color: 'var(--color-foreground)',
                          background: 'color-mix(in srgb, var(--color-foreground) 4%, transparent)',
                        }}
                      >
                        <input
                          type="checkbox"
                          checked={allChecked}
                          ref={(el) => {
                            if (el) el.indeterminate = someChecked && !allChecked;
                          }}
                          onChange={() => toggleResourceCheck(resource)}
                          style={{ cursor: 'pointer', accentColor: 'var(--color-primary)' }}
                        />
                        {resource}
                      </label>
                      <div
                        style={{
                          paddingLeft: 20,
                          display: 'flex',
                          flexDirection: 'column',
                          gap: 1,
                        }}
                      >
                        {ACTIONS.map((action) => {
                          const permKey = `${resource}:${action}`;
                          if (selectedPermissions.includes(permKey)) return null;
                          return (
                            <label
                              key={action}
                              style={{
                                display: 'flex',
                                alignItems: 'center',
                                gap: 8,
                                padding: '3px 4px',
                                borderRadius: 4,
                                cursor: 'pointer',
                                fontSize: 10,
                                color: 'var(--color-muted)',
                              }}
                            >
                              <input
                                type="checkbox"
                                checked={availableChecked.includes(permKey)}
                                onChange={() => toggleAvailableCheck(permKey)}
                                style={{
                                  cursor: 'pointer',
                                  accentColor: 'var(--color-primary)',
                                  width: 12,
                                  height: 12,
                                }}
                              />
                              {action}
                            </label>
                          );
                        })}
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>

            {/* Center arrows */}
            <div
              style={{
                width: 60,
                display: 'flex',
                flexDirection: 'column',
                gap: 8,
                alignItems: 'center',
                justifyContent: 'center',
                flexShrink: 0,
              }}
            >
              <button
                onClick={moveToSelected}
                disabled={availableChecked.length === 0}
                style={{
                  width: 36,
                  height: 36,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  background:
                    availableChecked.length > 0
                      ? 'var(--color-primary)'
                      : 'color-mix(in srgb, var(--color-foreground) 8%, transparent)',
                  border: '1px solid var(--color-separator)',
                  borderRadius: 8,
                  cursor: availableChecked.length > 0 ? 'pointer' : 'not-allowed',
                  color: availableChecked.length > 0 ? '#fff' : 'var(--color-muted)',
                  fontSize: 16,
                  fontWeight: 700,
                  transition: 'background 0.15s',
                }}
              >
                ›
              </button>
              <button
                onClick={removeFromSelected}
                disabled={
                  availableChecked.filter((p) => selectedPermissions.includes(p)).length === 0
                }
                style={{
                  width: 36,
                  height: 36,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  background: 'color-mix(in srgb, var(--color-foreground) 8%, transparent)',
                  border: '1px solid var(--color-separator)',
                  borderRadius: 8,
                  cursor: 'pointer',
                  color: 'var(--color-muted)',
                  fontSize: 16,
                  fontWeight: 700,
                  transition: 'background 0.15s',
                }}
              >
                ‹
              </button>
            </div>

            {/* Right pane: Selected */}
            <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
              <div
                style={{
                  fontSize: 11,
                  fontWeight: 600,
                  color: 'var(--color-muted)',
                  textTransform: 'uppercase',
                  letterSpacing: '0.06em',
                  marginBottom: 6,
                }}
              >
                Selected{' '}
                <span style={{ color: 'var(--color-primary)' }}>
                  ({selectedPermissions.length})
                </span>
              </div>
              <div
                style={{
                  height: 240,
                  overflowY: 'auto',
                  border: '1px solid var(--color-separator)',
                  borderRadius: 8,
                  padding: 8,
                  display: 'flex',
                  flexDirection: 'column',
                  gap: 4,
                }}
              >
                {selectedPermissions.length === 0 ? (
                  <div
                    style={{
                      fontSize: 11,
                      color: 'var(--color-muted)',
                      textAlign: 'center',
                      marginTop: 20,
                    }}
                  >
                    No permissions selected
                  </div>
                ) : (
                  selectedPermissions.map((permKey) => {
                    const [resource, action] = permKey.split(':');
                    return (
                      <div
                        key={permKey}
                        style={{
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'space-between',
                          gap: 6,
                        }}
                      >
                        <span
                          style={{
                            background: 'color-mix(in srgb, var(--color-primary) 15%, transparent)',
                            border:
                              '1px solid color-mix(in srgb, var(--color-primary) 30%, transparent)',
                            borderRadius: 12,
                            padding: '2px 8px',
                            fontSize: 11,
                            color: 'var(--color-foreground)',
                            flex: 1,
                          }}
                        >
                          {resource} • {action}
                        </span>
                        <button
                          onClick={() => removePermPill(permKey)}
                          style={{
                            background: 'none',
                            border: 'none',
                            color: 'var(--color-muted)',
                            cursor: 'pointer',
                            display: 'flex',
                            alignItems: 'center',
                            padding: 2,
                            flexShrink: 0,
                          }}
                        >
                          <X size={12} />
                        </button>
                      </div>
                    );
                  })
                )}
              </div>
            </div>
          </div>
        </div>

        {/* Footer */}
        <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
          <button
            onClick={onClose}
            style={{
              background: 'none',
              border: '1px solid var(--color-separator)',
              borderRadius: 8,
              padding: '7px 14px',
              fontSize: 12,
              fontWeight: 600,
              color: 'var(--color-muted)',
              cursor: 'pointer',
            }}
          >
            Cancel
          </button>
          <button
            onClick={handleReset}
            style={{
              background: 'none',
              border: '1px solid var(--color-separator)',
              borderRadius: 8,
              padding: '7px 14px',
              fontSize: 12,
              fontWeight: 600,
              color: 'var(--color-muted)',
              cursor: 'pointer',
            }}
          >
            Reset
          </button>
          <button
            onClick={handleSave}
            disabled={!roleName.trim()}
            style={{
              background: roleName.trim()
                ? 'var(--color-primary)'
                : 'color-mix(in srgb, var(--color-primary) 40%, transparent)',
              color: '#fff',
              border: 'none',
              borderRadius: 8,
              padding: '7px 14px',
              fontSize: 12,
              fontWeight: 600,
              cursor: roleName.trim() ? 'pointer' : 'not-allowed',
            }}
          >
            {mode === 'create' ? 'Create Role' : 'Save Changes'}
          </button>
        </div>
      </div>
    </div>
  );
}

// ── Roles page ────────────────────────────────────────────────────────────────
export default function Roles() {
  useHotkeys();

  // State: role list and permissions
  const [roles, setRoles] = useState<Role[]>(INITIAL_ROLES);
  const [rolePermissions, setRolePermissions] =
    useState<Record<string, Permission>>(INITIAL_ROLE_PERMISSIONS);

  // Selection
  const [selectedRole, setSelectedRole] = useState('Operator');

  // Modal state
  const [showModal, setShowModal] = useState(false);
  const [modalMode, setModalMode] = useState<'create' | 'edit'>('create');

  // Delete confirm
  const [confirmDelete, setConfirmDelete] = useState(false);

  const permissions = rolePermissions[selectedRole] ?? {};
  const role = roles.find((r) => r.name === selectedRole);

  // Convert Permission record → array of "Resource:Action" strings
  function permissionsToStringArray(perms: Permission): string[] {
    const result: string[] = [];
    for (const [resource, actions] of Object.entries(perms)) {
      for (const action of actions) {
        result.push(`${resource}:${action}`);
      }
    }
    return result;
  }

  // Convert array of "Resource:Action" strings → Permission record
  function stringArrayToPermissions(arr: string[]): Permission {
    const result: Permission = {};
    for (const item of arr) {
      const [resource, action] = item.split(':');
      if (!result[resource]) result[resource] = [];
      if (!result[resource].includes(action)) {
        result[resource].push(action);
      }
    }
    return result;
  }

  function handleOpenCreate() {
    setModalMode('create');
    setShowModal(true);
    setConfirmDelete(false);
  }

  function handleOpenEdit() {
    setModalMode('edit');
    setShowModal(true);
    setConfirmDelete(false);
  }

  function handleModalSave(name: string, desc: string, selectedPerms: string[]) {
    const newPerms = stringArrayToPermissions(selectedPerms);
    const permCount = countPermissions(newPerms);

    if (modalMode === 'create') {
      const newRole: Role = {
        name,
        icon: '🔒',
        iconColor: 'linear-gradient(135deg, var(--color-cyan), #0891b2)',
        description: desc || 'Custom role',
        users: 0,
        permissions: permCount,
        type: 'custom',
        assignedUsers: [],
      };
      setRoles((prev) => [...prev, newRole]);
      setRolePermissions((prev) => ({ ...prev, [name]: newPerms }));
      setSelectedRole(name);
    } else {
      // Edit mode
      setRoles((prev) =>
        prev.map((r) =>
          r.name === selectedRole ? { ...r, name, description: desc, permissions: permCount } : r,
        ),
      );
      setRolePermissions((prev) => {
        const updated = { ...prev };
        if (name !== selectedRole) {
          updated[name] = newPerms;
          delete updated[selectedRole];
        } else {
          updated[selectedRole] = newPerms;
        }
        return updated;
      });
      setSelectedRole(name);
    }
    setShowModal(false);
  }

  function handleConfirmDelete() {
    setRoles((prev) => prev.filter((r) => r.name !== selectedRole));
    setRolePermissions((prev) => {
      const updated = { ...prev };
      delete updated[selectedRole];
      return updated;
    });
    const remaining = roles.filter((r) => r.name !== selectedRole);
    setSelectedRole(remaining[0]?.name ?? '');
    setConfirmDelete(false);
  }

  function togglePermCell(resource: string, action: string) {
    if (role?.type === 'system') return;
    setRolePermissions((prev) => {
      const current = prev[selectedRole] ?? {};
      const currentActions = current[resource] ?? [];
      const newActions = currentActions.includes(action)
        ? currentActions.filter((a) => a !== action)
        : [...currentActions, action];
      return {
        ...prev,
        [selectedRole]: {
          ...current,
          [resource]: newActions,
        },
      };
    });
  }

  const isSystemRole = role?.type === 'system';

  // For the full comparison matrix at the bottom
  const ACTION_ABBRS: Record<string, string> = {
    View: 'V',
    Create: 'C',
    Update: 'U',
    Delete: 'D',
    Deploy: 'Dp',
    Export: 'Ex',
  };

  return (
    <>
      {showModal && (
        <RoleModal
          mode={modalMode}
          initialName={modalMode === 'edit' ? selectedRole : ''}
          initialDesc={modalMode === 'edit' ? (role?.description ?? '') : ''}
          initialPermissions={modalMode === 'edit' ? permissionsToStringArray(permissions) : []}
          onSave={handleModalSave}
          onClose={() => setShowModal(false)}
        />
      )}

      <motion.div
        variants={stagger}
        initial="hidden"
        animate="show"
        style={{
          display: 'flex',
          flexDirection: 'column',
          gap: 16,
          padding: '20px 24px',
          overflowY: 'auto',
          height: '100%',
        }}
      >
        {/* Page header */}
        <motion.div
          variants={fadeUp}
          style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}
        >
          <div>
            <h1 style={{ fontSize: 20, fontWeight: 700, margin: 0 }}>Roles & Permissions</h1>
            <p style={{ fontSize: 12, color: 'var(--color-muted)', margin: '4px 0 0' }}>
              {roles.length} roles
            </p>
          </div>
          <button
            onClick={handleOpenCreate}
            style={{
              background: 'var(--color-primary)',
              color: '#fff',
              border: 'none',
              borderRadius: 8,
              padding: '8px 14px',
              fontSize: 13,
              fontWeight: 600,
              cursor: 'pointer',
              display: 'inline-flex',
              alignItems: 'center',
              gap: 6,
            }}
          >
            <Plus size={14} />
            Create Role
          </button>
        </motion.div>

        {/* Two-section layout */}
        <motion.div
          variants={fadeUp}
          style={{
            display: 'grid',
            gridTemplateColumns: '1fr 1fr',
            gap: 16,
            alignItems: 'start',
          }}
        >
          {/* LEFT: Roles table */}
          <GlassCard className="p-5" hover={false}>
            <SectionHeader title="Roles" />
            <div style={{ marginTop: 12 }}>
              {/* Table header */}
              <div
                style={{
                  display: 'grid',
                  gridTemplateColumns: '1fr auto auto auto',
                  padding: '6px 8px',
                  borderBottom: '1px solid var(--color-separator)',
                  marginBottom: 4,
                }}
              >
                {['Role', 'Users', 'Perms', 'Type'].map((col) => (
                  <span
                    key={col}
                    style={{
                      fontSize: 10,
                      fontWeight: 700,
                      color: 'var(--color-muted)',
                      textTransform: 'uppercase',
                      letterSpacing: '0.05em',
                      textAlign: col === 'Role' ? 'left' : 'center',
                    }}
                  >
                    {col}
                  </span>
                ))}
              </div>

              {/* Table rows */}
              {roles.map((r) => {
                const isSelected = r.name === selectedRole;
                return (
                  <div
                    key={r.name}
                    onClick={() => {
                      setSelectedRole(r.name);
                      setConfirmDelete(false);
                    }}
                    style={{
                      display: 'grid',
                      gridTemplateColumns: '1fr auto auto auto',
                      alignItems: 'center',
                      padding: '10px 8px',
                      borderRadius: 8,
                      cursor: 'pointer',
                      background: isSelected
                        ? 'color-mix(in srgb, var(--color-primary) 10%, transparent)'
                        : 'transparent',
                      border: isSelected
                        ? '1px solid color-mix(in srgb, var(--color-primary) 30%, transparent)'
                        : '1px solid transparent',
                      marginBottom: 4,
                      transition: 'background 0.15s ease',
                    }}
                  >
                    {/* Role name + description */}
                    <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                      <span style={{ fontSize: 16 }}>{r.icon}</span>
                      <div>
                        <div
                          style={{
                            fontSize: 13,
                            fontWeight: 600,
                            color: isSelected ? 'var(--color-primary)' : 'var(--color-foreground)',
                          }}
                        >
                          {r.name}
                        </div>
                        <div
                          style={{
                            fontSize: 11,
                            color: 'var(--color-muted)',
                            marginTop: 1,
                          }}
                        >
                          {r.description}
                        </div>
                      </div>
                    </div>

                    {/* Users */}
                    <div
                      style={{
                        fontSize: 13,
                        fontWeight: 600,
                        color: 'var(--color-foreground)',
                        textAlign: 'center',
                        minWidth: 40,
                      }}
                    >
                      {r.users}
                    </div>

                    {/* Permissions count */}
                    <div
                      style={{
                        fontSize: 13,
                        fontWeight: 600,
                        color: 'var(--color-cyan)',
                        textAlign: 'center',
                        minWidth: 40,
                      }}
                    >
                      {r.permissions}
                    </div>

                    {/* Type badge */}
                    <div style={{ minWidth: 70, display: 'flex', justifyContent: 'center' }}>
                      <TypeBadge type={r.type} />
                    </div>
                  </div>
                );
              })}
            </div>
          </GlassCard>

          {/* RIGHT: Permission detail */}
          <GlassCard className="p-5" hover={false}>
            {/* Detail header: icon + name + actions */}
            <div
              style={{
                display: 'flex',
                alignItems: 'flex-start',
                gap: 12,
                marginBottom: 14,
                paddingBottom: 14,
                borderBottom: '1px solid var(--color-separator)',
              }}
            >
              <div
                style={{
                  width: 40,
                  height: 40,
                  borderRadius: 10,
                  background: role?.iconColor ?? 'var(--color-separator)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: 18,
                  flexShrink: 0,
                }}
              >
                {role?.icon}
              </div>
              <div style={{ flex: 1 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <div style={{ fontSize: 14, fontWeight: 700, color: 'var(--color-foreground)' }}>
                    {selectedRole}
                  </div>
                  <TypeBadge type={role?.type ?? 'system'} />
                </div>
                <div style={{ fontSize: 12, color: 'var(--color-muted)', marginTop: 2 }}>
                  {role?.description}
                </div>
              </div>
              {/* Edit / Delete buttons */}
              <div style={{ display: 'flex', gap: 6, flexShrink: 0 }}>
                <button
                  onClick={handleOpenEdit}
                  style={{
                    display: 'inline-flex',
                    alignItems: 'center',
                    gap: 4,
                    background: 'color-mix(in srgb, var(--color-primary) 12%, transparent)',
                    border: '1px solid color-mix(in srgb, var(--color-primary) 30%, transparent)',
                    borderRadius: 6,
                    padding: '5px 10px',
                    fontSize: 11,
                    fontWeight: 600,
                    color: 'var(--color-primary)',
                    cursor: 'pointer',
                  }}
                >
                  <Edit2 size={12} />
                  Edit
                </button>
                {role?.type === 'custom' && (
                  <button
                    onClick={() => setConfirmDelete((v) => !v)}
                    style={{
                      display: 'inline-flex',
                      alignItems: 'center',
                      gap: 4,
                      background: 'color-mix(in srgb, var(--color-danger) 12%, transparent)',
                      border: '1px solid color-mix(in srgb, var(--color-danger) 30%, transparent)',
                      borderRadius: 6,
                      padding: '5px 10px',
                      fontSize: 11,
                      fontWeight: 600,
                      color: 'var(--color-danger)',
                      cursor: 'pointer',
                    }}
                  >
                    <Trash2 size={12} />
                    Delete
                  </button>
                )}
              </div>
            </div>

            {/* Delete confirm section */}
            {confirmDelete && role?.type === 'custom' && (
              <div
                style={{
                  border: '1px solid var(--color-danger)',
                  borderRadius: 8,
                  padding: '12px 14px',
                  marginBottom: 14,
                  background: 'color-mix(in srgb, var(--color-danger) 8%, transparent)',
                }}
              >
                <div
                  style={{
                    fontSize: 12,
                    fontWeight: 600,
                    color: 'var(--color-danger)',
                    marginBottom: 6,
                  }}
                >
                  Delete "{selectedRole}"?
                </div>
                <div style={{ fontSize: 11, color: 'var(--color-muted)', marginBottom: 12 }}>
                  Are you sure you want to delete this role? This will affect {role.users} user
                  {role.users !== 1 ? 's' : ''}.
                </div>
                <div style={{ display: 'flex', gap: 8 }}>
                  <button
                    onClick={handleConfirmDelete}
                    style={{
                      background: 'var(--color-danger)',
                      color: '#fff',
                      border: 'none',
                      borderRadius: 6,
                      padding: '5px 12px',
                      fontSize: 11,
                      fontWeight: 600,
                      cursor: 'pointer',
                    }}
                  >
                    Confirm Delete
                  </button>
                  <button
                    onClick={() => setConfirmDelete(false)}
                    style={{
                      background: 'none',
                      border: '1px solid var(--color-separator)',
                      borderRadius: 6,
                      padding: '5px 12px',
                      fontSize: 11,
                      fontWeight: 600,
                      color: 'var(--color-muted)',
                      cursor: 'pointer',
                    }}
                  >
                    Cancel
                  </button>
                </div>
              </div>
            )}

            {/* Assigned Users */}
            {role && role.assignedUsers.length > 0 && (
              <div style={{ marginBottom: 16 }}>
                <div
                  style={{
                    fontSize: 11,
                    fontWeight: 600,
                    color: 'var(--color-muted)',
                    textTransform: 'uppercase',
                    letterSpacing: '0.06em',
                    marginBottom: 8,
                  }}
                >
                  Assigned Users
                </div>
                <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
                  {role.assignedUsers.map((u) => (
                    <div
                      key={u.id}
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: 6,
                        padding: '4px 10px',
                        background: 'color-mix(in srgb, var(--color-foreground) 5%, transparent)',
                        border: '1px solid var(--color-separator)',
                        borderRadius: 20,
                        fontSize: 11,
                        color: 'var(--color-foreground)',
                      }}
                    >
                      <div
                        style={{
                          width: 20,
                          height: 20,
                          borderRadius: '50%',
                          background: u.color,
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          fontSize: 9,
                          fontWeight: 700,
                          color: '#fff',
                          flexShrink: 0,
                        }}
                      >
                        {u.id}
                      </div>
                      {u.name}
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Permission Matrix */}
            <div style={{ overflowX: 'auto' }}>
              <div
                style={{
                  fontSize: 11,
                  fontWeight: 600,
                  color: 'var(--color-foreground)',
                  marginBottom: 8,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                }}
              >
                Permissions
                {!isSystemRole && (
                  <span style={{ fontSize: 10, color: 'var(--color-muted)', fontWeight: 400 }}>
                    Click cells to toggle
                  </span>
                )}
              </div>
              <table style={{ borderCollapse: 'collapse', width: '100%', minWidth: 420 }}>
                <thead>
                  <tr>
                    <th
                      style={{
                        textAlign: 'left',
                        fontSize: 10,
                        fontWeight: 700,
                        color: 'var(--color-muted)',
                        textTransform: 'uppercase',
                        letterSpacing: '0.05em',
                        padding: '4px 8px 8px',
                        borderBottom: '1px solid var(--color-separator)',
                      }}
                    >
                      Resource
                    </th>
                    {ACTIONS.map((action) => (
                      <th
                        key={action}
                        style={{
                          width: 32,
                          textAlign: 'center',
                          fontSize: 10,
                          fontWeight: 700,
                          color: 'var(--color-muted)',
                          textTransform: 'uppercase',
                          letterSpacing: '0.05em',
                          padding: '4px 0 8px',
                          borderBottom: '1px solid var(--color-separator)',
                        }}
                      >
                        {action.slice(0, 2)}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {RESOURCES.map((resource, i) => {
                    const allowed = permissions[resource] ?? [];
                    return (
                      <tr
                        key={resource}
                        style={{
                          background:
                            i % 2 === 0
                              ? 'transparent'
                              : 'color-mix(in srgb, var(--color-separator) 20%, transparent)',
                        }}
                      >
                        <td
                          style={{
                            fontSize: 12,
                            fontWeight: 500,
                            color: 'var(--color-foreground)',
                            padding: '2px 8px',
                          }}
                        >
                          {resource}
                        </td>
                        {ACTIONS.map((action) => {
                          const isAllowed = allowed.includes(action);
                          return (
                            <td key={action} style={{ padding: 0, textAlign: 'center' }}>
                              <div
                                onClick={() => togglePermCell(resource, action)}
                                style={{
                                  width: 32,
                                  height: 32,
                                  display: 'flex',
                                  alignItems: 'center',
                                  justifyContent: 'center',
                                  fontSize: 13,
                                  color: isAllowed ? 'var(--color-success)' : 'var(--color-muted)',
                                  cursor: isSystemRole ? 'default' : 'pointer',
                                  borderRadius: 4,
                                  transition: 'background 0.1s',
                                }}
                                onMouseEnter={(e) => {
                                  if (!isSystemRole) {
                                    (e.currentTarget as HTMLDivElement).style.background =
                                      'color-mix(in srgb, var(--color-primary) 12%, transparent)';
                                  }
                                }}
                                onMouseLeave={(e) => {
                                  (e.currentTarget as HTMLDivElement).style.background =
                                    'transparent';
                                }}
                              >
                                {isAllowed ? '✓' : '—'}
                              </div>
                            </td>
                          );
                        })}
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>

            {/* Legend */}
            <div
              style={{
                marginTop: 12,
                display: 'flex',
                gap: 16,
                fontSize: 11,
                color: 'var(--color-muted)',
              }}
            >
              <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                <span style={{ color: 'var(--color-success)', fontWeight: 700 }}>✓</span> Allowed
              </span>
              <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                <span style={{ fontWeight: 700 }}>—</span> Denied
              </span>
              <span style={{ color: 'var(--color-muted)', marginLeft: 'auto' }}>
                Vw=View · Cr=Create · Up=Update · De=Delete · Dp=Deploy · Ex=Export
              </span>
            </div>
          </GlassCard>
        </motion.div>

        {/* Full Permission Overview Matrix */}
        <motion.div variants={fadeUp}>
          <GlassCard className="p-5" hover={false}>
            <div
              style={{
                fontSize: 14,
                fontWeight: 700,
                color: 'var(--color-foreground)',
                marginBottom: 14,
              }}
            >
              Permission Overview
            </div>
            <div style={{ overflowX: 'auto' }}>
              <table
                style={{
                  width: '100%',
                  borderCollapse: 'separate',
                  borderSpacing: 3,
                  minWidth: 700,
                }}
              >
                <thead>
                  <tr>
                    <th
                      style={{
                        fontSize: 10,
                        fontWeight: 600,
                        color: 'var(--color-muted)',
                        textAlign: 'left',
                        padding: '6px 8px',
                        minWidth: 110,
                      }}
                    >
                      Resource
                    </th>
                    {roles.map((r) => (
                      <th
                        key={r.name}
                        style={{
                          fontSize: 10,
                          fontWeight: 600,
                          color:
                            r.name === selectedRole ? 'var(--color-primary)' : 'var(--color-muted)',
                          textAlign: 'center',
                          padding: '6px 8px',
                          whiteSpace: 'nowrap',
                        }}
                      >
                        {r.icon} {r.name.length > 12 ? r.name.slice(0, 10) + '…' : r.name}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {RESOURCES.map((resource, i) => (
                    <tr
                      key={resource}
                      style={{
                        background:
                          i % 2 === 0
                            ? 'transparent'
                            : 'color-mix(in srgb, var(--color-separator) 15%, transparent)',
                      }}
                    >
                      <td
                        style={{
                          fontSize: 11,
                          fontWeight: 500,
                          color: 'var(--color-foreground)',
                          padding: '5px 8px',
                          whiteSpace: 'nowrap',
                        }}
                      >
                        {resource}
                      </td>
                      {roles.map((r) => {
                        const rPerms = (rolePermissions[r.name] ?? {})[resource] ?? [];
                        return (
                          <td
                            key={r.name}
                            style={{
                              textAlign: 'center',
                              padding: '4px 6px',
                            }}
                          >
                            {rPerms.length === 0 ? (
                              <span style={{ fontSize: 11, color: 'var(--color-muted)' }}>—</span>
                            ) : (
                              <div
                                style={{
                                  display: 'flex',
                                  flexWrap: 'wrap',
                                  gap: 2,
                                  justifyContent: 'center',
                                }}
                              >
                                {ACTIONS.filter((a) => rPerms.includes(a)).map((a) => (
                                  <span
                                    key={a}
                                    title={a}
                                    style={{
                                      fontSize: 9,
                                      fontWeight: 700,
                                      padding: '1px 4px',
                                      borderRadius: 3,
                                      background:
                                        a === 'Delete'
                                          ? 'color-mix(in srgb, var(--color-danger) 20%, transparent)'
                                          : a === 'Deploy'
                                            ? 'color-mix(in srgb, var(--color-warning) 20%, transparent)'
                                            : 'color-mix(in srgb, var(--color-success) 20%, transparent)',
                                      color:
                                        a === 'Delete'
                                          ? 'var(--color-danger)'
                                          : a === 'Deploy'
                                            ? 'var(--color-warning)'
                                            : 'var(--color-success)',
                                    }}
                                  >
                                    {ACTION_ABBRS[a]}
                                  </span>
                                ))}
                              </div>
                            )}
                          </td>
                        );
                      })}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {/* Legend */}
            <div
              style={{
                marginTop: 12,
                display: 'flex',
                gap: 12,
                flexWrap: 'wrap',
                fontSize: 10,
                color: 'var(--color-muted)',
              }}
            >
              <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                <span
                  style={{
                    background: 'color-mix(in srgb, var(--color-success) 20%, transparent)',
                    color: 'var(--color-success)',
                    padding: '1px 4px',
                    borderRadius: 3,
                    fontWeight: 700,
                  }}
                >
                  V/C/U/Ex
                </span>
                View / Create / Update / Export
              </span>
              <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                <span
                  style={{
                    background: 'color-mix(in srgb, var(--color-warning) 20%, transparent)',
                    color: 'var(--color-warning)',
                    padding: '1px 4px',
                    borderRadius: 3,
                    fontWeight: 700,
                  }}
                >
                  Dp
                </span>
                Deploy
              </span>
              <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                <span
                  style={{
                    background: 'color-mix(in srgb, var(--color-danger) 20%, transparent)',
                    color: 'var(--color-danger)',
                    padding: '1px 4px',
                    borderRadius: 3,
                    fontWeight: 700,
                  }}
                >
                  D
                </span>
                Delete
              </span>
            </div>
          </GlassCard>
        </motion.div>
      </motion.div>
    </>
  );
}
