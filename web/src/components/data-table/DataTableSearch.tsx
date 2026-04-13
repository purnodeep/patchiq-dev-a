import { Search } from 'lucide-react';

interface DataTableSearchProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
}

export const DataTableSearch = ({
  value,
  onChange,
  placeholder = 'Search...',
}: DataTableSearchProps) => (
  <div style={{ position: 'relative', maxWidth: 320 }}>
    <Search
      style={{
        position: 'absolute',
        left: 10,
        top: '50%',
        transform: 'translateY(-50%)',
        width: 14,
        height: 14,
        color: 'var(--text-muted)',
        pointerEvents: 'none',
      }}
    />
    <input
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder={placeholder}
      style={{
        display: 'block',
        width: '100%',
        height: 36,
        paddingLeft: 32,
        paddingRight: 12,
        fontFamily: 'var(--font-sans)',
        fontSize: 13,
        color: 'var(--text-primary)',
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: 6,
        outline: 'none',
        boxSizing: 'border-box',
        transition: 'border-color 0.15s ease',
      }}
    />
  </div>
);
