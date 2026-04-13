import { useNavigate } from 'react-router';
import { SkeletonCard } from '@patchiq/ui';
import { useFeeds } from '../../api/hooks/useFeeds';
import { useDashboardStats } from '../../api/hooks/useDashboard';
import type { Feed } from '../../types/feed';

import './feed-pipeline.css';

export const FeedPipeline = () => {
  const navigate = useNavigate();
  const { data: feeds, isLoading: feedsLoading } = useFeeds();
  const { data: stats, isLoading: statsLoading } = useDashboardStats();

  if (feedsLoading || statsLoading) {
    return <SkeletonCard />;
  }

  const enabledFeeds = feeds?.filter((f: Feed) => f.enabled) ?? [];
  const errorFeed = enabledFeeds.find((f) => f.status === 'error');

  return (
    <div
      className="rounded-xl p-5"
      style={{
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        boxShadow: 'var(--shadow-sm)',
      }}
    >
      <div style={{ cursor: 'pointer' }} onClick={() => void navigate('/feeds')}>
        <h3 className="font-bold mb-1" style={{ color: 'var(--text-emphasis)' }}>
          Feed Pipeline{' '}
          <span style={{ fontSize: 10, color: 'var(--text-faint)', fontWeight: 400 }}>→</span>
        </h3>
        <p className="text-xs mb-3" style={{ color: 'var(--text-muted)' }}>
          Live data ingestion from vulnerability feeds to catalog
        </p>
      </div>

      <div className="flex items-stretch gap-0">
        {/* Left: Feed rows — each feed + its own pipe in one row */}
        <div className="flex flex-col gap-2 flex-1">
          {enabledFeeds.map((feed, i) => {
            const isError = feed.status === 'error';
            return (
              <div key={feed.id} className="flex items-center gap-0">
                {/* Feed label */}
                <div
                  className="flex-shrink-0 px-2 py-1.5 rounded-md text-center"
                  style={{
                    width: 90,
                    border: `1.5px solid ${isError ? 'var(--signal-critical)' : 'var(--border)'}`,
                    background: isError ? 'var(--signal-critical-subtle)' : 'var(--bg-card-hover)',
                    cursor: 'pointer',
                    transition: 'border-color 150ms',
                  }}
                  onClick={() => void navigate(`/feeds/${feed.id}`)}
                  onMouseEnter={(e) => {
                    e.currentTarget.style.borderColor = 'var(--text-faint)';
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.borderColor = isError
                      ? 'var(--signal-critical)'
                      : 'var(--border)';
                  }}
                >
                  <div
                    className="font-bold leading-tight"
                    style={{
                      fontSize: 10,
                      color: isError ? 'var(--signal-critical)' : 'var(--text-secondary)',
                      whiteSpace: 'nowrap',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                    }}
                  >
                    {feed.display_name}
                  </div>
                  <div
                    style={{
                      fontSize: 8,
                      color: isError ? 'var(--signal-critical)' : 'var(--text-muted)',
                      marginTop: 1,
                    }}
                  >
                    {isError ? 'Error' : `${feed.entries_ingested.toLocaleString()} entries`}
                  </div>
                </div>

                {/* Pipe — directly connected to this feed */}
                <div className="flex-1 flex items-center px-2">
                  <div
                    className="w-full h-1.5 rounded-full feed-pipe relative overflow-hidden"
                    style={{
                      background: isError
                        ? 'var(--signal-critical-subtle)'
                        : 'var(--accent-subtle)',
                    }}
                  >
                    {isError ? (
                      <div className="pipe-particle-red" />
                    ) : (
                      <>
                        <div
                          className="pipe-particle"
                          style={{ animationDelay: `${i * 0.3}s`, background: 'var(--accent)' }}
                        />
                        <div
                          className="pipe-particle"
                          style={{
                            animationDelay: `${i * 0.3 + 0.7}s`,
                            background: 'var(--accent)',
                            opacity: 0.6,
                          }}
                        />
                      </>
                    )}
                  </div>
                </div>
              </div>
            );
          })}
        </div>

        {/* Right: Catalog destination — all pipes converge here */}
        <div className="flex-shrink-0 flex items-center" style={{ width: 80 }}>
          <div
            className="w-full px-1 py-3 rounded-xl text-center"
            style={{
              border: '2px solid var(--accent-border)',
              background: 'var(--accent-subtle)',
              overflow: 'hidden',
              cursor: 'pointer',
            }}
            onClick={() => void navigate('/catalog')}
          >
            <p className="font-bold" style={{ fontSize: 9, color: 'var(--accent)' }}>
              Catalog
            </p>
            <p
              className="font-bold mt-0.5"
              style={{
                fontSize: 16,
                color: 'var(--accent)',
                fontFamily: 'var(--font-mono)',
                lineHeight: 1.1,
              }}
            >
              {(stats?.total_catalog_entries ?? 0).toLocaleString()}
            </p>
            <p style={{ fontSize: 8, color: 'var(--text-muted)' }}>entries</p>
          </div>
        </div>
      </div>

      {/* Error banner */}
      {errorFeed && (
        <div
          className="mt-3 p-2 rounded-lg text-xs flex items-center gap-2"
          style={{
            background: 'var(--signal-critical-subtle)',
            border: '1px solid var(--signal-critical)',
            color: 'var(--signal-critical)',
          }}
        >
          <svg
            className="w-4 h-4 flex-shrink-0"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
            />
          </svg>
          {errorFeed.display_name}: {errorFeed.last_error ?? 'Sync failed'}
        </div>
      )}
    </div>
  );
};
