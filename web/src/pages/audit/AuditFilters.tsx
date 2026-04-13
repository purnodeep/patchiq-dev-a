export interface AuditFiltersProps {
  eventType: string;
  onEventTypeChange: (value: string) => void;
  actorSearch: string;
  onActorSearchChange: (value: string) => void;
  resource: string;
  onResourceChange: (value: string) => void;
  dateRange: string;
  onDateRangeChange: (value: string) => void;
  fromDate: string;
  onFromDateChange: (value: string) => void;
  toDate: string;
  onToDateChange: (value: string) => void;
}

const EVENT_TYPES = [
  { value: '__all__', label: 'All Event Types' },
  { value: 'endpoint', label: 'Endpoint' },
  { value: 'patch', label: 'Patch' },
  { value: 'deployment', label: 'Deployment' },
  { value: 'policy', label: 'Policy' },
  { value: 'compliance', label: 'Compliance' },
  { value: 'auth', label: 'Auth' },
  { value: 'system', label: 'System' },
];

const RESOURCE_TYPES = [
  { value: '__all__', label: 'All Resources' },
  { value: 'endpoint', label: 'Endpoints' },
  { value: 'deployment', label: 'Deployments' },
  { value: 'policy', label: 'Policies' },
  { value: 'settings', label: 'Settings' },
];

const DATE_RANGES = [
  { value: '24h', label: 'Last 24h' },
  { value: '7d', label: 'Last 7 days' },
  { value: '30d', label: 'Last 30 days' },
  { value: 'custom', label: 'Custom Range' },
];

const selectStyle: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: 6,
  padding: '5px 10px',
  fontSize: 11.5,
  color: 'var(--text-secondary)',
  outline: 'none',
  cursor: 'pointer',
};

function focusBorder(e: React.FocusEvent<HTMLSelectElement | HTMLInputElement>) {
  e.currentTarget.style.borderColor = 'var(--accent)';
}
function blurBorder(e: React.FocusEvent<HTMLSelectElement | HTMLInputElement>) {
  e.currentTarget.style.borderColor = 'var(--border)';
}

export function AuditFilters({
  eventType,
  onEventTypeChange,
  actorSearch,
  onActorSearchChange,
  resource,
  onResourceChange,
  dateRange,
  onDateRangeChange,
  fromDate,
  onFromDateChange,
  toDate,
  onToDateChange,
}: AuditFiltersProps) {
  const hasFilters =
    (eventType && eventType !== '__all__') ||
    actorSearch !== '' ||
    (resource && resource !== '__all__') ||
    dateRange !== '30d';

  const clearFilters = () => {
    onEventTypeChange('__all__');
    onActorSearchChange('');
    onResourceChange('__all__');
    onDateRangeChange('30d');
    onFromDateChange('');
    onToDateChange('');
  };

  return (
    <div
      style={{
        flex: 1,
        display: 'flex',
        alignItems: 'center',
        gap: 8,
        flexWrap: 'wrap',
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: 8,
        padding: '10px 14px',
        boxShadow: 'var(--shadow-sm)',
      }}
    >
      {/* Search */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          padding: '5px 10px',
          border: '1px solid var(--border)',
          borderRadius: 6,
          background: 'var(--bg-inset)',
          flex: 1,
          maxWidth: 280,
          transition: 'border-color 0.15s',
        }}
        onFocusCapture={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
        onBlurCapture={(e) => (e.currentTarget.style.borderColor = 'var(--border)')}
      >
        <svg
          width="12"
          height="12"
          viewBox="0 0 24 24"
          fill="none"
          stroke="var(--text-muted)"
          strokeWidth="2.5"
          aria-hidden="true"
        >
          <circle cx="11" cy="11" r="8" />
          <path d="M21 21l-4.35-4.35" />
        </svg>
        <input
          type="text"
          value={actorSearch}
          onChange={(e) => onActorSearchChange(e.target.value)}
          placeholder="Search by actor..."
          aria-label="Search by actor"
          style={{
            background: 'transparent',
            border: 'none',
            outline: 'none',
            fontSize: 12,
            color: 'var(--text-primary)',
            width: '100%',
          }}
        />
        {actorSearch && (
          <button
            type="button"
            aria-label="Clear search"
            onClick={() => onActorSearchChange('')}
            style={{
              width: 16,
              height: 16,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              background: 'transparent',
              border: 'none',
              cursor: 'pointer',
              color: 'var(--text-muted)',
              padding: 0,
            }}
          >
            <svg
              width="10"
              height="10"
              viewBox="0 0 10 10"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.5"
            >
              <path d="M2 2l6 6M8 2l-6 6" />
            </svg>
          </button>
        )}
      </div>

      <select
        aria-label="Filter by event type"
        value={eventType || '__all__'}
        onChange={(e) => onEventTypeChange(e.target.value)}
        style={{
          ...selectStyle,
          color:
            eventType && eventType !== '__all__' ? 'var(--text-primary)' : 'var(--text-secondary)',
        }}
        onFocus={focusBorder}
        onBlur={blurBorder}
      >
        {EVENT_TYPES.map((t) => (
          <option key={t.value} value={t.value}>
            {t.label}
          </option>
        ))}
      </select>

      <select
        aria-label="Filter by resource"
        value={resource || '__all__'}
        onChange={(e) => onResourceChange(e.target.value)}
        style={{
          ...selectStyle,
          color:
            resource && resource !== '__all__' ? 'var(--text-primary)' : 'var(--text-secondary)',
        }}
        onFocus={focusBorder}
        onBlur={blurBorder}
      >
        {RESOURCE_TYPES.map((t) => (
          <option key={t.value} value={t.value}>
            {t.label}
          </option>
        ))}
      </select>

      <select
        aria-label="Filter by date range"
        value={dateRange || '30d'}
        onChange={(e) => onDateRangeChange(e.target.value)}
        style={{
          ...selectStyle,
          color: dateRange && dateRange !== '30d' ? 'var(--text-primary)' : 'var(--text-secondary)',
        }}
        onFocus={focusBorder}
        onBlur={blurBorder}
      >
        {DATE_RANGES.map((t) => (
          <option key={t.value} value={t.value}>
            {t.label}
          </option>
        ))}
      </select>

      {dateRange === 'custom' && (
        <>
          <input
            type="date"
            value={fromDate}
            onChange={(e) => onFromDateChange(e.target.value)}
            aria-label="From date"
            style={{ ...selectStyle, fontSize: 11 }}
            onFocus={focusBorder}
            onBlur={blurBorder}
          />
          <input
            type="date"
            value={toDate}
            onChange={(e) => onToDateChange(e.target.value)}
            aria-label="To date"
            style={{ ...selectStyle, fontSize: 11 }}
            onFocus={focusBorder}
            onBlur={blurBorder}
          />
        </>
      )}

      {hasFilters && (
        <button
          type="button"
          onClick={clearFilters}
          style={{
            padding: '5px 10px',
            fontSize: 11,
            borderRadius: 6,
            border: '1px solid var(--border)',
            background: 'transparent',
            color: 'var(--text-secondary)',
            cursor: 'pointer',
            display: 'flex',
            alignItems: 'center',
            gap: 4,
          }}
        >
          <svg
            width="10"
            height="10"
            viewBox="0 0 10 10"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.5"
          >
            <path d="M2 2l6 6M8 2l-6 6" />
          </svg>
          Clear filters
        </button>
      )}
    </div>
  );
}
