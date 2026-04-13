import { useEffect } from 'react';

interface KeyboardHelpProps {
  open: boolean;
  onClose: () => void;
}

const shortcuts = [
  { keys: ['Ctrl+Z', 'Cmd+Z'], description: 'Undo' },
  { keys: ['Ctrl+Shift+Z', 'Cmd+Y'], description: 'Redo' },
  { keys: ['Ctrl+C', 'Cmd+C'], description: 'Copy selected nodes' },
  { keys: ['Ctrl+V', 'Cmd+V'], description: 'Paste nodes' },
  { keys: ['Delete', 'Backspace'], description: 'Delete selected' },
  { keys: ['?'], description: 'Show keyboard shortcuts' },
];

const backdropStyle: React.CSSProperties = {
  position: 'fixed',
  inset: 0,
  background: 'rgba(0, 0, 0, 0.6)',
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  zIndex: 9999,
};

const cardStyle: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: 12,
  padding: '24px',
  maxWidth: 400,
  width: '100%',
  boxShadow: '0 16px 48px rgba(0, 0, 0, 0.3)',
};

const titleStyle: React.CSSProperties = {
  margin: '0 0 16px 0',
  fontSize: 16,
  fontWeight: 600,
  color: 'var(--text-primary)',
};

const rowStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'space-between',
  padding: '8px 0',
  borderBottom: '1px solid var(--border)',
};

const descStyle: React.CSSProperties = {
  color: 'var(--text-secondary)',
  fontSize: 13,
};

const kbdStyle: React.CSSProperties = {
  fontFamily: 'var(--font-mono)',
  fontSize: 11,
  fontWeight: 500,
  padding: '2px 6px',
  borderRadius: 4,
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  color: 'var(--text-muted)',
  boxShadow: '0 1px 0 var(--border)',
};

const keyGroupStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: 4,
  flexShrink: 0,
};

const separatorStyle: React.CSSProperties = {
  color: 'var(--text-muted)',
  fontSize: 11,
};

export function KeyboardHelp({ open, onClose }: KeyboardHelpProps) {
  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div style={backdropStyle} onClick={onClose}>
      <div style={cardStyle} onClick={(e) => e.stopPropagation()}>
        <h3 style={titleStyle}>Keyboard Shortcuts</h3>
        {shortcuts.map((s, i) => (
          <div
            key={i}
            style={i === shortcuts.length - 1 ? { ...rowStyle, borderBottom: 'none' } : rowStyle}
          >
            <div style={keyGroupStyle}>
              {s.keys.map((k, j) => (
                <span key={j} style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
                  {j > 0 && <span style={separatorStyle}>/</span>}
                  <kbd style={kbdStyle}>{k}</kbd>
                </span>
              ))}
            </div>
            <span style={descStyle}>{s.description}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
