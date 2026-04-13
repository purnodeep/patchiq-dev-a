import { useState } from 'react';
import { PreferencesTab } from './PreferencesTab';
import { HistoryTab } from './HistoryTab';

type TabValue = 'preferences' | 'history';

const TABS: { value: TabValue; label: string }[] = [
  { value: 'preferences', label: 'Preferences' },
  { value: 'history', label: 'History' },
];

export function NotificationsPage() {
  const [activeTab, setActiveTab] = useState<TabValue>('preferences');

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        height: '100%',
        background: 'var(--bg-page)',
      }}
    >
      {/* Page header */}
      <div
        style={{
          borderBottom: '1px solid var(--border)',
          padding: '20px 24px 0',
        }}
      >
        <h1
          style={{
            fontFamily: 'var(--font-sans)',
            fontSize: 22,
            fontWeight: 600,
            color: 'var(--text-emphasis)',
            letterSpacing: '-0.02em',
            marginBottom: 16,
          }}
        >
          Notifications
        </h1>

        {/* Tab bar */}
        <div style={{ display: 'flex', gap: 0 }}>
          {TABS.map((tab) => {
            const isActive = activeTab === tab.value;
            return (
              <button
                key={tab.value}
                role="tab"
                aria-selected={isActive}
                onClick={() => setActiveTab(tab.value)}
                style={{
                  fontFamily: 'var(--font-sans)',
                  fontSize: 13,
                  fontWeight: isActive ? 600 : 400,
                  color: isActive ? 'var(--text-emphasis)' : 'var(--text-muted)',
                  padding: '8px 18px',
                  background: 'none',
                  border: 'none',
                  borderBottom: isActive ? '2px solid var(--accent)' : '2px solid transparent',
                  cursor: 'pointer',
                  transition: 'color 0.15s, border-color 0.15s',
                  marginBottom: -1,
                }}
                onMouseEnter={(e) => {
                  if (!isActive)
                    (e.currentTarget as HTMLButtonElement).style.color = 'var(--text-secondary)';
                }}
                onMouseLeave={(e) => {
                  if (!isActive)
                    (e.currentTarget as HTMLButtonElement).style.color = 'var(--text-muted)';
                }}
              >
                {tab.label}
              </button>
            );
          })}
        </div>
      </div>

      {/* Tab content */}
      <div style={{ flex: 1, overflowY: 'auto' }}>
        {activeTab === 'preferences' && <PreferencesTab />}
        {activeTab === 'history' && <HistoryTab />}
      </div>
    </div>
  );
}
