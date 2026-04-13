import { useState } from 'react';
import { useFormContext } from 'react-hook-form';
import { Package, ScrollText, Terminal, X, Search } from 'lucide-react';
import { usePatches } from '../../api/hooks/usePatches';
import { usePolicies } from '../../api/hooks/usePolicies';
import { SeverityBadge } from '../SeverityBadge';
import type { DeploymentWizardValues } from '../../types/deployment-wizard';

const INPUT: React.CSSProperties = {
  width: '100%',
  background: 'var(--bg-inset)',
  border: '1px solid var(--border)',
  borderRadius: 6,
  padding: '7px 10px',
  fontSize: 12,
  color: 'var(--text-primary)',
  fontFamily: 'var(--font-sans)',
  outline: 'none',
  transition: 'border-color 0.15s',
  boxSizing: 'border-box',
};

const sourceOptions = [
  {
    value: 'catalog' as const,
    label: 'Patch Catalog',
    description: 'Select from available patches',
    Icon: Package,
  },
  {
    value: 'policy' as const,
    label: 'Policy',
    description: 'Deploy based on a policy',
    Icon: ScrollText,
  },
  {
    value: 'adhoc' as const,
    label: 'Ad-hoc Packages',
    description: 'Specify packages manually',
    Icon: Terminal,
  },
];

const LABEL_STYLE: React.CSSProperties = {
  fontSize: 10,
  fontWeight: 600,
  color: 'var(--text-muted)',
  textTransform: 'uppercase',
  letterSpacing: '0.06em',
  fontFamily: 'var(--font-mono)',
  marginBottom: 8,
  display: 'block',
};

export function SourceStep() {
  const { watch, setValue } = useFormContext<DeploymentWizardValues>();
  const sourceType = watch('sourceType');
  const selectedPatchIds = watch('patchIds') ?? [];
  const selectedPolicyId = watch('policyId');

  return (
    <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 18 }}>
      {/* Source type selector */}
      <div>
        <label style={LABEL_STYLE}>Deployment Source</label>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 8 }}>
          {sourceOptions.map(({ value, label, description, Icon }) => {
            const isActive = sourceType === value;
            return (
              <button
                key={value}
                type="button"
                onClick={() => setValue('sourceType', value)}
                style={{
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  gap: 6,
                  padding: '12px 8px',
                  borderRadius: 8,
                  border: `1px solid ${isActive ? 'var(--accent)' : 'var(--border)'}`,
                  background: isActive
                    ? 'color-mix(in srgb, var(--accent) 8%, transparent)'
                    : 'var(--bg-inset)',
                  cursor: 'pointer',
                  transition: 'all 0.15s',
                  textAlign: 'center',
                }}
                onMouseEnter={(e) => {
                  if (!isActive) e.currentTarget.style.borderColor = 'var(--border-hover)';
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.borderColor = isActive ? 'var(--accent)' : 'var(--border)';
                }}
              >
                <Icon
                  style={{
                    width: 18,
                    height: 18,
                    color: isActive ? 'var(--accent)' : 'var(--text-muted)',
                  }}
                />
                <span
                  style={{
                    fontSize: 11,
                    fontWeight: 600,
                    color: isActive ? 'var(--accent)' : 'var(--text-secondary)',
                  }}
                >
                  {label}
                </span>
                <span style={{ fontSize: 9, color: 'var(--text-muted)', lineHeight: 1.3 }}>
                  {description}
                </span>
              </button>
            );
          })}
        </div>
      </div>

      {/* Source content */}
      {sourceType === 'catalog' && (
        <CatalogSelector
          selectedIds={selectedPatchIds}
          onSelect={(ids) => setValue('patchIds', ids)}
        />
      )}
      {sourceType === 'policy' && (
        <PolicySelector selectedId={selectedPolicyId} onSelect={(id) => setValue('policyId', id)} />
      )}
      {sourceType === 'adhoc' && <AdhocPackages />}
    </div>
  );
}

// --- Catalog Selector ---

function CatalogSelector({
  selectedIds,
  onSelect,
}: {
  selectedIds: string[];
  onSelect: (ids: string[]) => void;
}) {
  const [search, setSearch] = useState('');
  const { data, isLoading } = usePatches({ search: search || undefined, limit: 10 });

  const togglePatch = (id: string) => {
    if (selectedIds.includes(id)) {
      onSelect(selectedIds.filter((p) => p !== id));
    } else {
      onSelect([...selectedIds, id]);
    }
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
      <label style={LABEL_STYLE}>Select Patches</label>

      {/* Selected tags */}
      {selectedIds.length > 0 && (
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 5 }}>
          {selectedIds.map((id) => (
            <span
              key={id}
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: 4,
                fontFamily: 'var(--font-mono)',
                fontSize: 10,
                color: 'var(--accent)',
                background: 'color-mix(in srgb, var(--accent) 8%, transparent)',
                border: '1px solid color-mix(in srgb, var(--accent) 20%, transparent)',
                borderRadius: 4,
                padding: '2px 7px',
              }}
            >
              {id.slice(0, 8)}
              <button
                type="button"
                onClick={() => togglePatch(id)}
                style={{
                  background: 'none',
                  border: 'none',
                  cursor: 'pointer',
                  padding: 0,
                  color: 'var(--text-muted)',
                  display: 'inline-flex',
                }}
              >
                <X style={{ width: 10, height: 10 }} />
              </button>
            </span>
          ))}
        </div>
      )}

      {/* Search input */}
      <div style={{ position: 'relative' }}>
        <Search
          style={{
            position: 'absolute',
            left: 10,
            top: '50%',
            transform: 'translateY(-50%)',
            width: 12,
            height: 12,
            color: 'var(--text-muted)',
          }}
        />
        <input
          type="text"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search patches (KB, USN, RHSA...)"
          style={{ ...INPUT, paddingLeft: 30 }}
          onFocus={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
          onBlur={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
        />
      </div>

      {/* Patch list */}
      <div
        style={{
          maxHeight: 240,
          overflowY: 'auto',
          background: 'var(--bg-inset)',
          border: '1px solid var(--border)',
          borderRadius: 6,
        }}
      >
        {isLoading ? (
          <div style={{ padding: 12, fontSize: 11, color: 'var(--text-muted)' }}>
            Loading patches...
          </div>
        ) : !data?.data?.length ? (
          <div style={{ padding: 12, fontSize: 11, color: 'var(--text-muted)' }}>
            No patches found.
          </div>
        ) : (
          data.data.map((patch, i) => {
            const isSelected = selectedIds.includes(patch.id);
            return (
              <button
                key={patch.id}
                type="button"
                onClick={() => togglePatch(patch.id)}
                style={{
                  display: 'flex',
                  width: '100%',
                  alignItems: 'center',
                  gap: 10,
                  padding: '8px 12px',
                  background: isSelected
                    ? 'color-mix(in srgb, var(--accent) 6%, transparent)'
                    : 'transparent',
                  border: 'none',
                  borderBottom:
                    i < (data.data?.length ?? 0) - 1 ? '1px solid var(--border)' : 'none',
                  cursor: 'pointer',
                  transition: 'background 0.1s',
                  textAlign: 'left',
                }}
                onMouseEnter={(e) => {
                  if (!isSelected)
                    e.currentTarget.style.background = 'color-mix(in srgb, white 2%, transparent)';
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.background = isSelected
                    ? 'color-mix(in srgb, var(--accent) 6%, transparent)'
                    : 'transparent';
                }}
              >
                {/* Checkbox */}
                <div
                  style={{
                    width: 14,
                    height: 14,
                    borderRadius: 3,
                    border: `1.5px solid ${isSelected ? 'var(--accent)' : 'var(--border)'}`,
                    background: isSelected ? 'var(--accent)' : 'transparent',
                    flexShrink: 0,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    fontSize: 8,
                    color: 'var(--btn-accent-text, #000)',
                    fontWeight: 700,
                    transition: 'all 0.15s',
                  }}
                >
                  {isSelected ? '✓' : ''}
                </div>

                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 1 }}>
                    <span
                      style={{
                        fontSize: 12,
                        fontWeight: 500,
                        color: 'var(--text-primary)',
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                        whiteSpace: 'nowrap',
                      }}
                    >
                      {patch.name}
                    </span>
                    <SeverityBadge severity={patch.severity} />
                  </div>
                  <span
                    style={{
                      fontSize: 10,
                      color: 'var(--text-muted)',
                      fontFamily: 'var(--font-mono)',
                    }}
                  >
                    {patch.os_family} · v{patch.version}
                  </span>
                </div>

                <span
                  style={{
                    fontSize: 10,
                    color: 'var(--text-muted)',
                    flexShrink: 0,
                    fontFamily: 'var(--font-mono)',
                  }}
                >
                  {patch.affected_endpoint_count} ep
                </span>
              </button>
            );
          })
        )}
      </div>

      {selectedIds.length > 0 && (
        <p style={{ fontSize: 11, color: 'var(--text-muted)', margin: 0 }}>
          {selectedIds.length} {selectedIds.length !== 1 ? 'patches' : 'patch'} selected
        </p>
      )}
    </div>
  );
}

// --- Policy Selector ---

function PolicySelector({
  selectedId,
  onSelect,
}: {
  selectedId?: string;
  onSelect: (id: string) => void;
}) {
  const { data, isLoading } = usePolicies({ limit: 50 });

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
      <label style={LABEL_STYLE}>Select Policy</label>
      <div
        style={{
          maxHeight: 300,
          overflowY: 'auto',
          background: 'var(--bg-inset)',
          border: '1px solid var(--border)',
          borderRadius: 6,
        }}
      >
        {isLoading ? (
          <div style={{ padding: 12, fontSize: 11, color: 'var(--text-muted)' }}>
            Loading policies...
          </div>
        ) : !data?.data?.length ? (
          <div style={{ padding: 12, fontSize: 11, color: 'var(--text-muted)' }}>
            No policies found.
          </div>
        ) : (
          data.data.map((policy, i) => {
            const isSelected = selectedId === policy.id;
            return (
              <button
                key={policy.id}
                type="button"
                onClick={() => onSelect(policy.id)}
                style={{
                  display: 'flex',
                  width: '100%',
                  alignItems: 'center',
                  gap: 10,
                  padding: '9px 12px',
                  background: isSelected
                    ? 'color-mix(in srgb, var(--accent) 6%, transparent)'
                    : 'transparent',
                  border: 'none',
                  borderBottom:
                    i < (data.data?.length ?? 0) - 1 ? '1px solid var(--border)' : 'none',
                  cursor: 'pointer',
                  transition: 'background 0.1s',
                  textAlign: 'left',
                }}
                onMouseEnter={(e) => {
                  if (!isSelected)
                    e.currentTarget.style.background = 'color-mix(in srgb, white 2%, transparent)';
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.background = isSelected
                    ? 'color-mix(in srgb, var(--accent) 6%, transparent)'
                    : 'transparent';
                }}
              >
                {/* Radio */}
                <div
                  style={{
                    width: 12,
                    height: 12,
                    borderRadius: '50%',
                    border: `1.5px solid ${isSelected ? 'var(--accent)' : 'var(--border)'}`,
                    background: isSelected ? 'var(--accent)' : 'transparent',
                    flexShrink: 0,
                    transition: 'all 0.15s',
                  }}
                />

                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 7, marginBottom: 1 }}>
                    <span style={{ fontSize: 12, fontWeight: 500, color: 'var(--text-primary)' }}>
                      {policy.name}
                    </span>
                    <span
                      style={{
                        fontFamily: 'var(--font-mono)',
                        fontSize: 9,
                        fontWeight: 600,
                        color: policy.enabled ? 'var(--accent)' : 'var(--text-muted)',
                        background: policy.enabled
                          ? 'color-mix(in srgb, var(--accent) 8%, transparent)'
                          : 'color-mix(in srgb, white 4%, transparent)',
                        border: `1px solid ${policy.enabled ? 'color-mix(in srgb, var(--accent) 20%, transparent)' : 'var(--border)'}`,
                        borderRadius: 3,
                        padding: '1px 5px',
                      }}
                    >
                      {policy.enabled ? 'Enabled' : 'Disabled'}
                    </span>
                  </div>
                  <span
                    style={{
                      fontSize: 10,
                      color: 'var(--text-muted)',
                      fontFamily: 'var(--font-mono)',
                    }}
                  >
                    {policy.selection_mode} · tag selector
                  </span>
                </div>
              </button>
            );
          })
        )}
      </div>
    </div>
  );
}

// --- Ad-hoc Packages ---

function AdhocPackages() {
  const { watch, setValue } = useFormContext<DeploymentWizardValues>();
  const packages = watch('adhocPackages') ?? [];

  const addPackage = () => {
    setValue('adhocPackages', [...packages, { name: '', version: '' }]);
  };

  const removePackage = (index: number) => {
    setValue(
      'adhocPackages',
      packages.filter((_, i) => i !== index),
    );
  };

  const updatePackage = (index: number, field: 'name' | 'version', value: string) => {
    const updated = packages.map((pkg, i) => (i === index ? { ...pkg, [field]: value } : pkg));
    setValue('adhocPackages', updated);
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
      <label style={LABEL_STYLE}>Packages</label>

      {packages.map((pkg, idx) => (
        <div key={idx} style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <input
            type="text"
            value={pkg.name}
            onChange={(e) => updatePackage(idx, 'name', e.target.value)}
            placeholder="Package name"
            style={{ ...INPUT, flex: 1 }}
            onFocus={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
            onBlur={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
          />
          <input
            type="text"
            value={pkg.version}
            onChange={(e) => updatePackage(idx, 'version', e.target.value)}
            placeholder="Version"
            style={{ ...INPUT, width: 100 }}
            onFocus={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
            onBlur={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
          />
          <button
            type="button"
            onClick={() => removePackage(idx)}
            style={{
              background: 'none',
              border: 'none',
              cursor: 'pointer',
              color: 'var(--text-muted)',
              padding: 4,
              display: 'flex',
            }}
          >
            <X style={{ width: 14, height: 14 }} />
          </button>
        </div>
      ))}

      <button
        type="button"
        onClick={addPackage}
        style={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: 5,
          padding: '6px 12px',
          borderRadius: 6,
          fontSize: 11,
          fontWeight: 500,
          cursor: 'pointer',
          border: '1px solid var(--border)',
          background: 'var(--bg-inset)',
          color: 'var(--text-secondary)',
          transition: 'border-color 0.15s',
          alignSelf: 'flex-start',
        }}
      >
        + Add Package
      </button>
    </div>
  );
}
