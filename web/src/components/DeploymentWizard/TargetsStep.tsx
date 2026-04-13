import { useState, useMemo } from 'react';
import { useFormContext } from 'react-hook-form';
import { Switch } from '@patchiq/ui';
import { Search } from 'lucide-react';
import { TagExpressionBuilder } from '../TagExpressionBuilder';
import { useEndpoints, type Endpoint } from '../../api/hooks/useEndpoints';
import type { DeploymentWizardValues, TagExpression } from '../../types/deployment-wizard';

type TargetMode = 'all' | 'tags' | 'select';

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

const TOGGLE_CARD: React.CSSProperties = {
  background: 'var(--bg-inset)',
  border: '1px solid var(--border)',
  borderRadius: 8,
  padding: '12px 14px',
};

const MODE_CARD_BASE: React.CSSProperties = {
  flex: 1,
  border: '1px solid var(--border)',
  borderRadius: 8,
  padding: '10px 12px',
  cursor: 'pointer',
  transition: 'all 0.15s ease',
  background: 'var(--bg-card)',
  textAlign: 'center',
};

function ModeCard({
  active,
  label,
  description,
  onClick,
}: {
  active: boolean;
  label: string;
  description: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      style={{
        ...MODE_CARD_BASE,
        borderColor: active ? 'var(--accent)' : 'var(--border)',
        background: active ? 'color-mix(in srgb, var(--accent) 6%, transparent)' : 'var(--bg-card)',
      }}
    >
      <div style={{ fontSize: 12, fontWeight: 600, color: 'var(--text-primary)', marginBottom: 2 }}>
        {label}
      </div>
      <div style={{ fontSize: 10, color: 'var(--text-muted)' }}>{description}</div>
    </button>
  );
}

function EndpointSelector({
  selectedIds,
  onSelectionChange,
}: {
  selectedIds: string[];
  onSelectionChange: (ids: string[]) => void;
}) {
  const [search, setSearch] = useState('');
  const { data, isLoading } = useEndpoints({ limit: 200, search: search || undefined });
  const endpoints: Endpoint[] = data?.data ?? [];
  const totalCount = data?.total_count ?? 0;

  const selectedSet = useMemo(() => new Set(selectedIds), [selectedIds]);

  const toggleEndpoint = (id: string) => {
    if (selectedSet.has(id)) {
      onSelectionChange(selectedIds.filter((eid) => eid !== id));
    } else {
      onSelectionChange([...selectedIds, id]);
    }
  };

  const toggleAll = () => {
    if (endpoints.length > 0 && endpoints.every((ep) => selectedSet.has(ep.id))) {
      // Deselect all visible
      const visibleIds = new Set(endpoints.map((ep) => ep.id));
      onSelectionChange(selectedIds.filter((id) => !visibleIds.has(id)));
    } else {
      // Select all visible
      const merged = new Set([...selectedIds, ...endpoints.map((ep) => ep.id)]);
      onSelectionChange([...merged]);
    }
  };

  const allVisibleSelected =
    endpoints.length > 0 && endpoints.every((ep) => selectedSet.has(ep.id));

  const statusDot = (status: string) => {
    const colors: Record<string, string> = {
      online: 'var(--signal-healthy)',
      offline: 'var(--text-muted)',
      pending: 'var(--signal-warning)',
      stale: 'var(--signal-critical)',
    };
    return (
      <span
        style={{
          width: 6,
          height: 6,
          borderRadius: '50%',
          background: colors[status] ?? 'var(--text-muted)',
          display: 'inline-block',
          flexShrink: 0,
        }}
      />
    );
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      {/* Search bar */}
      <div style={{ position: 'relative' }}>
        <Search
          style={{
            position: 'absolute',
            left: 8,
            top: '50%',
            transform: 'translateY(-50%)',
            width: 13,
            height: 13,
            color: 'var(--text-muted)',
            pointerEvents: 'none',
          }}
        />
        <input
          type="text"
          placeholder="Search endpoints..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          style={{
            width: '100%',
            height: 32,
            borderRadius: 6,
            border: '1px solid var(--border)',
            background: 'var(--bg-card)',
            padding: '0 8px 0 28px',
            fontSize: 12,
            color: 'var(--text-primary)',
            outline: 'none',
            boxSizing: 'border-box',
          }}
        />
      </div>

      {/* Selection count */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          fontSize: 10,
          color: 'var(--text-muted)',
          fontFamily: 'var(--font-mono)',
        }}
      >
        <span>
          {selectedIds.length} of {totalCount} selected
        </span>
        <button
          type="button"
          onClick={toggleAll}
          style={{
            background: 'none',
            border: 'none',
            color: 'var(--accent)',
            cursor: 'pointer',
            fontSize: 10,
            fontFamily: 'var(--font-mono)',
            padding: 0,
          }}
        >
          {allVisibleSelected ? 'Deselect visible' : 'Select visible'}
        </button>
      </div>

      {/* Endpoint list */}
      <div
        style={{
          border: '1px solid var(--border)',
          borderRadius: 8,
          overflow: 'hidden',
          maxHeight: 240,
          overflowY: 'auto',
        }}
      >
        {isLoading ? (
          <div
            style={{
              padding: 16,
              textAlign: 'center',
              fontSize: 11,
              color: 'var(--text-muted)',
            }}
          >
            Loading endpoints...
          </div>
        ) : endpoints.length === 0 ? (
          <div
            style={{
              padding: 16,
              textAlign: 'center',
              fontSize: 11,
              color: 'var(--text-muted)',
            }}
          >
            No endpoints found
          </div>
        ) : (
          endpoints.map((ep, idx) => (
            <label
              key={ep.id}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 10,
                padding: '8px 12px',
                cursor: 'pointer',
                borderBottom: idx < endpoints.length - 1 ? '1px solid var(--border)' : undefined,
                background: selectedSet.has(ep.id)
                  ? 'color-mix(in srgb, var(--accent) 4%, transparent)'
                  : 'transparent',
                transition: 'background 0.1s',
              }}
            >
              <input
                type="checkbox"
                checked={selectedSet.has(ep.id)}
                onChange={() => toggleEndpoint(ep.id)}
                style={{
                  accentColor: 'var(--accent)',
                  width: 14,
                  height: 14,
                  cursor: 'pointer',
                  flexShrink: 0,
                }}
              />
              {statusDot(ep.status)}
              <div style={{ flex: 1, minWidth: 0 }}>
                <div
                  style={{
                    fontSize: 12,
                    fontWeight: 500,
                    color: 'var(--text-primary)',
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {ep.hostname}
                </div>
                <div style={{ fontSize: 10, color: 'var(--text-muted)' }}>
                  {ep.os_family} {ep.os_version}
                  {ep.ip_address ? ` · ${ep.ip_address}` : ''}
                </div>
              </div>
              <span
                style={{
                  fontSize: 9,
                  fontFamily: 'var(--font-mono)',
                  color: 'var(--text-muted)',
                  flexShrink: 0,
                }}
              >
                {ep.status}
              </span>
            </label>
          ))
        )}
      </div>
    </div>
  );
}

export function TargetsStep() {
  const { watch, setValue } = useFormContext<DeploymentWizardValues>();
  const targetMode = watch('targetMode');
  const targetExpression = watch('targetExpression');
  const endpointIds = watch('endpointIds') ?? [];
  const respectMaintenanceWindow = watch('respectMaintenanceWindow');
  const excludePendingDeployments = watch('excludePendingDeployments');

  const { data: totalData } = useEndpoints({ limit: 1 });
  const totalCount = totalData?.total_count ?? 0;

  const firstTag = getFirstTag(targetExpression);
  const hasConditions = !!targetExpression;
  const { data: filteredData, isLoading: endpointsLoading } = useEndpoints(
    hasConditions && firstTag ? { tag_id: firstTag, limit: 1 } : { limit: 1 },
  );
  const tagEndpointCount = hasConditions ? (filteredData?.total_count ?? 0) : totalCount;

  const setMode = (mode: TargetMode) => {
    setValue('targetMode', mode);
    // Clear other mode's data when switching
    if (mode !== 'tags') setValue('targetExpression', undefined);
    if (mode !== 'select') setValue('endpointIds', []);
  };

  // Endpoint count for display based on mode
  const effectiveCount =
    targetMode === 'all'
      ? totalCount
      : targetMode === 'tags'
        ? tagEndpointCount
        : endpointIds.length;

  return (
    <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 18 }}>
      {/* Mode selector */}
      <div>
        <label style={LABEL_STYLE}>Target Mode</label>
        <div style={{ display: 'flex', gap: 8 }}>
          <ModeCard
            active={targetMode === 'all'}
            label="All Endpoints"
            description={`${totalCount} total`}
            onClick={() => setMode('all')}
          />
          <ModeCard
            active={targetMode === 'tags'}
            label="Filter by Tags"
            description="Tag expressions"
            onClick={() => setMode('tags')}
          />
          <ModeCard
            active={targetMode === 'select'}
            label="Select Endpoints"
            description="Pick individually"
            onClick={() => setMode('select')}
          />
        </div>
      </div>

      {/* Tag expression builder — only in tags mode */}
      {targetMode === 'tags' && (
        <div>
          <label style={LABEL_STYLE}>Tag Expression</label>
          <p style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 12, marginTop: 0 }}>
            Define which endpoints receive this deployment using tag expressions.
          </p>
          <TagExpressionBuilder
            value={targetExpression}
            onChange={(expr) => setValue('targetExpression', expr)}
            endpointCount={tagEndpointCount}
            endpointCountLoading={endpointsLoading && !!targetExpression}
          />
        </div>
      )}

      {/* Endpoint selector — only in select mode */}
      {targetMode === 'select' && (
        <div>
          <label style={LABEL_STYLE}>Select Endpoints</label>
          <EndpointSelector
            selectedIds={endpointIds}
            onSelectionChange={(ids) => setValue('endpointIds', ids)}
          />
        </div>
      )}

      {/* All endpoints summary — only in all mode */}
      {targetMode === 'all' && (
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            borderRadius: 6,
            background: 'var(--bg-inset)',
            border: '1px solid var(--border)',
            padding: '10px 14px',
          }}
        >
          <div
            style={{
              width: 8,
              height: 8,
              borderRadius: '50%',
              background: 'var(--signal-healthy)',
              flexShrink: 0,
            }}
          />
          <span style={{ fontSize: 12, color: 'var(--text-secondary)' }}>
            All {totalCount} endpoints will be targeted
          </span>
        </div>
      )}

      {/* Endpoint count bar (for tags and select modes) */}
      {targetMode !== 'all' && (
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            borderRadius: 6,
            background: 'var(--bg-inset)',
            border: '1px solid var(--border)',
            padding: '8px 12px',
          }}
        >
          <div
            style={{
              width: 8,
              height: 8,
              borderRadius: '50%',
              background: effectiveCount > 0 ? 'var(--signal-healthy)' : 'var(--text-muted)',
              flexShrink: 0,
            }}
          />
          <span style={{ fontSize: 11, color: 'var(--text-secondary)' }}>
            {effectiveCount} endpoint{effectiveCount !== 1 ? 's' : ''}{' '}
            {targetMode === 'tags' ? 'match this criteria' : 'selected'}
          </span>
        </div>
      )}

      {/* Options */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        <div style={TOGGLE_CARD}>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              gap: 12,
            }}
          >
            <div>
              <div
                style={{
                  fontSize: 12,
                  fontWeight: 500,
                  color: 'var(--text-primary)',
                  marginBottom: 2,
                }}
              >
                Respect Maintenance Windows
              </div>
              <div style={{ fontSize: 10, color: 'var(--text-muted)' }}>
                Only deploy during configured maintenance windows
              </div>
            </div>
            <Switch
              checked={respectMaintenanceWindow}
              onCheckedChange={(checked) => setValue('respectMaintenanceWindow', checked)}
            />
          </div>
        </div>

        <div style={TOGGLE_CARD}>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              gap: 12,
            }}
          >
            <div>
              <div
                style={{
                  fontSize: 12,
                  fontWeight: 500,
                  color: 'var(--text-primary)',
                  marginBottom: 2,
                }}
              >
                Exclude Pending Deployments
              </div>
              <div style={{ fontSize: 10, color: 'var(--text-muted)' }}>
                Skip endpoints with active deployments
              </div>
            </div>
            <Switch
              checked={excludePendingDeployments}
              onCheckedChange={(checked) => setValue('excludePendingDeployments', checked)}
            />
          </div>
        </div>
      </div>
    </div>
  );
}

function getFirstTag(expr: TagExpression | undefined): string | undefined {
  if (!expr) return undefined;
  if (expr.tag) return expr.tag;
  if (expr.conditions?.length) return getFirstTag(expr.conditions[0]);
  return undefined;
}
