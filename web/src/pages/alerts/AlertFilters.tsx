import { Search } from 'lucide-react';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue, Input } from '@patchiq/ui';

export interface AlertFiltersProps {
  status: string;
  onStatusChange: (v: string) => void;
  category: string;
  onCategoryChange: (v: string) => void;
  search: string;
  onSearchChange: (v: string) => void;
  dateRange: string;
  onDateRangeChange: (v: string) => void;
  fromDate: string;
  onFromDateChange: (v: string) => void;
  toDate: string;
  onToDateChange: (v: string) => void;
}

const STATUS_OPTIONS = [
  { value: 'active', label: 'Active' },
  { value: 'acknowledged', label: 'Acknowledged' },
  { value: 'dismissed', label: 'Dismissed' },
  { value: 'all', label: 'All' },
];

const CATEGORY_OPTIONS = [
  { value: 'all', label: 'All' },
  { value: 'deployments', label: 'Deployments' },
  { value: 'agents', label: 'Agents' },
  { value: 'cves', label: 'CVEs' },
  { value: 'compliance', label: 'Compliance' },
  { value: 'system', label: 'System' },
];

const DATE_RANGES = [
  { value: '24h', label: 'Last 24h' },
  { value: '7d', label: 'Last 7 days' },
  { value: '30d', label: 'Last 30 days' },
  { value: 'custom', label: 'Custom Range' },
];

function PillButton({
  label,
  isActive,
  onClick,
}: {
  label: string;
  isActive: boolean;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 5,
        padding: '5px 12px',
        fontSize: 12,
        fontWeight: isActive ? 600 : 400,
        fontFamily: 'var(--font-sans)',
        color: isActive ? 'var(--text-emphasis)' : 'var(--text-muted)',
        background: isActive ? 'var(--bg-elevated)' : 'transparent',
        border: `1px solid ${isActive ? 'var(--border-hover)' : 'var(--border)'}`,
        borderRadius: 'var(--radius-full, 9999px)',
        cursor: 'pointer',
        transition: 'all 0.12s ease',
        whiteSpace: 'nowrap',
      }}
    >
      {label}
    </button>
  );
}

function Divider() {
  return (
    <div
      style={{
        width: 1,
        height: 20,
        background: 'var(--border)',
        flexShrink: 0,
        alignSelf: 'center',
      }}
    />
  );
}

export function AlertFilters({
  status,
  onStatusChange,
  category,
  onCategoryChange,
  search,
  onSearchChange,
  dateRange,
  onDateRangeChange,
  fromDate,
  onFromDateChange,
  toDate,
  onToDateChange,
}: AlertFiltersProps) {
  return (
    <div
      style={{
        display: 'flex',
        flexWrap: 'wrap',
        alignItems: 'center',
        gap: 8,
        padding: '10px 14px',
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: 8,
        boxShadow: 'var(--shadow-sm)',
      }}
    >
      {/* Status pills */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
        {STATUS_OPTIONS.map((opt) => (
          <PillButton
            key={opt.value}
            label={opt.label}
            isActive={(status || 'active') === opt.value}
            onClick={() => onStatusChange(opt.value)}
          />
        ))}
      </div>

      <Divider />

      {/* Category pills */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
        {CATEGORY_OPTIONS.map((opt) => (
          <PillButton
            key={opt.value}
            label={opt.label}
            isActive={(category || 'all') === opt.value}
            onClick={() => onCategoryChange(opt.value)}
          />
        ))}
      </div>

      <Divider />

      {/* Search input */}
      <div style={{ position: 'relative', display: 'flex', alignItems: 'center' }}>
        <Search
          style={{
            position: 'absolute',
            left: 8,
            width: 13,
            height: 13,
            color: 'var(--text-muted)',
            pointerEvents: 'none',
          }}
        />
        <Input
          type="text"
          value={search}
          onChange={(e) => onSearchChange(e.target.value)}
          placeholder="Search alerts..."
          className="w-[180px] h-8 text-xs"
          style={{ paddingLeft: 26 }}
          aria-label="Search alerts"
        />
      </div>

      {/* Date range */}
      <Select value={dateRange || '24h'} onValueChange={onDateRangeChange}>
        <SelectTrigger className="w-[130px]" size="sm">
          <SelectValue placeholder="Last 24h" />
        </SelectTrigger>
        <SelectContent>
          {DATE_RANGES.map((t) => (
            <SelectItem key={t.value} value={t.value}>
              {t.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      {dateRange === 'custom' && (
        <>
          <Input
            type="date"
            value={fromDate}
            onChange={(e) => onFromDateChange(e.target.value)}
            className="w-[140px] h-8 text-xs"
            aria-label="From date"
          />
          <Input
            type="date"
            value={toDate}
            onChange={(e) => onToDateChange(e.target.value)}
            className="w-[140px] h-8 text-xs"
            aria-label="To date"
          />
        </>
      )}
    </div>
  );
}
