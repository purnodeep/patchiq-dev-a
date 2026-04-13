import { FeedConfigSettings } from './FeedConfigSettings';

export const FeedConfigSettingsPage = () => {
  return (
    <div style={{ padding: '28px 40px 80px', maxWidth: 900 }}>
      {/* Section header */}
      <div style={{ paddingBottom: 16, borderBottom: '1px solid var(--border)', marginBottom: 24 }}>
        <h2
          style={{
            fontSize: 18,
            fontWeight: 600,
            color: 'var(--text-emphasis)',
            letterSpacing: '-0.02em',
            margin: 0,
          }}
        >
          Feed Sources
        </h2>
        <p
          style={{
            fontSize: 12,
            color: 'var(--text-muted)',
            margin: '4px 0 0',
          }}
        >
          Vulnerability data sources, sync schedules, and feed health status.
        </p>
      </div>

      <FeedConfigSettings />
    </div>
  );
};
