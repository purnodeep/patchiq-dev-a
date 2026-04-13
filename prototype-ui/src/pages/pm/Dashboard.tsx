import { useState, useEffect } from 'react';
import { useDashboardLayout } from '@/hooks/useDashboardLayout';
import { useHotkeys } from '@/hooks/useHotkeys';
import { DashboardHeader } from './DashboardHeader';
import { DashboardGrid } from './DashboardGrid';

// ── Edit FAB ──────────────────────────────────────────────────────────────────

interface EditFabProps {
  isEditMode: boolean;
  onToggle: () => void;
  onReset: () => void;
}

function EditFab({ isEditMode, onToggle, onReset }: EditFabProps) {
  const [justReset, setJustReset] = useState(false);

  useEffect(() => {
    if (!justReset) return;
    const t = setTimeout(() => setJustReset(false), 2000);
    return () => clearTimeout(t);
  }, [justReset]);

  function handleReset() {
    onReset();
    setJustReset(true);
  }

  return (
    <div
      style={{
        position: 'fixed',
        bottom: 28,
        right: 28,
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        gap: 10,
        zIndex: 100,
      }}
    >
      {/* Reset pill — only visible in edit mode */}
      {isEditMode && (
        <button
          onClick={handleReset}
          style={{
            padding: '6px 14px',
            borderRadius: 100,
            fontSize: 11,
            fontWeight: 600,
            border: '1px solid var(--color-glass-border)',
            background: 'var(--color-glass-card)',
            backdropFilter: 'blur(12px)',
            WebkitBackdropFilter: 'blur(12px)',
            color: justReset ? 'var(--color-success)' : 'var(--color-muted)',
            cursor: 'pointer',
            transition: 'color 0.2s',
            boxShadow: '0 4px 16px rgba(0,0,0,0.15)',
            whiteSpace: 'nowrap',
          }}
        >
          {justReset ? '✓ Reset' : 'Reset layout'}
        </button>
      )}

      {/* Main FAB */}
      <button
        onClick={onToggle}
        title={isEditMode ? 'Done editing' : 'Edit layout'}
        style={{
          width: 52,
          height: 52,
          borderRadius: '50%',
          border: 'none',
          background: isEditMode ? 'var(--color-success)' : 'var(--color-primary)',
          color: '#fff',
          cursor: 'pointer',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          boxShadow: isEditMode
            ? '0 4px 20px rgba(34,197,94,0.4), 0 2px 8px rgba(0,0,0,0.2)'
            : '0 4px 20px rgba(59,130,246,0.4), 0 2px 8px rgba(0,0,0,0.2)',
          transition: 'background 0.2s, box-shadow 0.2s, transform 0.15s',
        }}
        onMouseEnter={(e) => {
          (e.currentTarget as HTMLButtonElement).style.transform = 'scale(1.08)';
        }}
        onMouseLeave={(e) => {
          (e.currentTarget as HTMLButtonElement).style.transform = 'scale(1)';
        }}
      >
        {isEditMode ? (
          // Checkmark
          <svg width={22} height={22} viewBox="0 0 22 22" fill="none">
            <path
              d="M5 11.5L9.5 16L17 7"
              stroke="white"
              strokeWidth={2.2}
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
        ) : (
          // Pencil
          <svg width={20} height={20} viewBox="0 0 20 20" fill="none">
            <path
              d="M13.5 3.5a2.121 2.121 0 0 1 3 3L7 16l-4 1 1-4 9.5-9.5z"
              stroke="white"
              strokeWidth={1.8}
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
        )}
      </button>
    </div>
  );
}

// ── Page ──────────────────────────────────────────────────────────────────────

export default function Dashboard() {
  useHotkeys();

  const {
    layout,
    isEditMode,
    alertDismissed,
    toggleEditMode,
    onLayoutChange,
    resetLayout,
    dismissAlert,
  } = useDashboardLayout();

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        gap: 12,
        padding: '20px 24px',
        overflowY: 'auto',
        height: '100%',
      }}
    >
      <DashboardHeader />
      <DashboardGrid
        layout={layout}
        isEditMode={isEditMode}
        alertDismissed={alertDismissed}
        onLayoutChange={onLayoutChange}
        onAlertDismiss={dismissAlert}
      />
      <EditFab isEditMode={isEditMode} onToggle={toggleEditMode} onReset={resetLayout} />
    </div>
  );
}
