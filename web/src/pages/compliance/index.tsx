import { useState } from 'react';
import { toast } from 'sonner';
import { useQueryClient } from '@tanstack/react-query';
import { Button, Skeleton, EmptyState, ErrorState } from '@patchiq/ui';
import { Download, RefreshCw, ShieldCheck, Settings } from 'lucide-react';
import {
  useComplianceSummary,
  useOverallComplianceScore,
  useOverdueControls,
  useTriggerEvaluation,
  useTriggerFrameworkEvaluation,
} from '../../api/hooks/useCompliance';
import { useCan } from '../../app/auth/AuthContext';
import { OverallScoreCard } from './components/overall-score-card';
import { FrameworkCard } from './components/framework-card';
import { ComplianceTrend } from './components/compliance-trend';
import { OverdueControlsTable } from './components/overdue-controls-table';
import { FrameworkManager } from './components/framework-manager';

export const CompliancePage = () => {
  const can = useCan();
  const queryClient = useQueryClient();
  const summary = useComplianceSummary();
  const overallScore = useOverallComplianceScore();
  const { data: overdueControls } = useOverdueControls();
  const triggerEval = useTriggerEvaluation();
  const triggerFwEval = useTriggerFrameworkEvaluation();
  const [managerOpen, setManagerOpen] = useState(false);

  const frameworks = summary.data?.frameworks ?? [];

  if (summary.isError || overallScore.isError) {
    return (
      <div style={{ padding: 24, background: 'var(--bg-page)' }}>
        <ErrorState
          message="Failed to load compliance data"
          onRetry={() => {
            void summary.refetch();
            void overallScore.refetch();
          }}
        />
      </div>
    );
  }

  const isLoading = summary.isLoading || overallScore.isLoading;

  return (
    <div
      style={{
        background: 'var(--bg-page)',
        minHeight: '100%',
        padding: '24px',
        display: 'flex',
        flexDirection: 'column',
        gap: 16,
      }}
    >
      {/* Page header */}
      <div
        style={{
          display: 'flex',
          alignItems: 'flex-start',
          justifyContent: 'space-between',
          paddingBottom: 16,
          borderBottom: '1px solid var(--border)',
        }}
      >
        <div>
          <h1
            style={{
              fontFamily: 'var(--font-sans)',
              fontSize: 22,
              fontWeight: 600,
              color: 'var(--text-emphasis)',
              letterSpacing: '-0.02em',
              margin: 0,
            }}
          >
            Compliance
          </h1>
          <div
            style={{
              fontFamily: 'var(--font-sans)',
              fontSize: 13,
              color: 'var(--text-secondary)',
            }}
          >
            Security framework evaluation and tracking
          </div>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <Button
            variant="outline"
            size="sm"
            onClick={() => setManagerOpen(true)}
            style={{ fontFamily: 'var(--font-mono)', fontSize: 11 }}
          >
            <Settings style={{ width: 12, height: 12, marginRight: 6 }} />
            Manage Frameworks
          </Button>
          <Button
            size="sm"
            disabled={triggerEval.isPending || !can('compliance', 'create')}
            title={!can('compliance', 'create') ? "You don't have permission" : undefined}
            onClick={() =>
              triggerEval.mutate(undefined, {
                onSuccess: (data) => {
                  const d = data as
                    | { frameworks_evaluated?: number; total_evaluations?: number }
                    | undefined;
                  toast.success(
                    `Evaluation complete — ${d?.frameworks_evaluated ?? 0} frameworks, ${d?.total_evaluations ?? 0} controls evaluated`,
                  );
                  void queryClient.invalidateQueries({ queryKey: ['compliance'] });
                },
                onError: (err) =>
                  toast.error(err instanceof Error ? err.message : 'Failed to trigger evaluation'),
              })
            }
            style={{ fontFamily: 'var(--font-mono)', fontSize: 11 }}
          >
            <RefreshCw
              style={{
                width: 12,
                height: 12,
                marginRight: 6,
                animation: triggerEval.isPending ? 'spin 1s linear infinite' : undefined,
              }}
            />
            {triggerEval.isPending ? 'Evaluating\u2026' : 'Evaluate All'}
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled
            style={{ fontFamily: 'var(--font-mono)', fontSize: 11 }}
          >
            <Download style={{ width: 12, height: 12, marginRight: 6, opacity: 0.5 }} />
            Export Report
            <span
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 9,
                color: 'var(--text-muted)',
                marginLeft: 8,
              }}
            >
              Soon
            </span>
          </Button>
        </div>
      </div>

      {/* Loading state */}
      {isLoading && (
        <>
          <Skeleton className="min-h-[160px] rounded-lg" />
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))',
              gap: 14,
            }}
          >
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="min-h-[260px] rounded-lg" />
            ))}
          </div>
        </>
      )}

      {/* Empty state */}
      {!isLoading && frameworks.length === 0 && (
        <EmptyState
          icon={<ShieldCheck style={{ width: 40, height: 40 }} />}
          title="No frameworks enabled"
          description="Enable a compliance framework to start tracking your security posture."
          action={{ label: 'Manage Frameworks', onClick: () => setManagerOpen(true) }}
        />
      )}

      {/* Evaluation in-progress banner */}
      {triggerEval.isPending && (
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 10,
            padding: '10px 16px',
            background: 'color-mix(in srgb, var(--accent) 8%, var(--bg-card))',
            border: '1px solid color-mix(in srgb, var(--accent) 25%, var(--border))',
            borderRadius: 8,
          }}
        >
          <RefreshCw
            style={{
              width: 14,
              height: 14,
              color: 'var(--accent)',
              animation: 'spin 1s linear infinite',
              flexShrink: 0,
            }}
          />
          <span
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 12,
              color: 'var(--accent)',
              fontWeight: 600,
            }}
          >
            Evaluating all frameworks...
          </span>
          <span
            style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--text-muted)' }}
          >
            Scores will update when complete
          </span>
        </div>
      )}

      {/* Main content */}
      {!isLoading && frameworks.length > 0 && overallScore.data && (
        <>
          <OverallScoreCard
            score={overallScore.data}
            frameworks={frameworks}
            overdueControlsCount={overdueControls?.length ?? 0}
          />

          <div
            style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))',
              gap: 14,
            }}
          >
            {frameworks.map((fw) => (
              <FrameworkCard
                key={fw.framework_id}
                framework={fw}
                onEvaluate={(frameworkId) =>
                  triggerFwEval.mutate(frameworkId, {
                    onSuccess: (data) => {
                      const d = data as { total_evaluations?: number } | undefined;
                      toast.success(
                        `Evaluation complete — ${d?.total_evaluations ?? 0} controls evaluated`,
                      );
                    },
                    onError: (err) =>
                      toast.error(
                        err instanceof Error ? err.message : 'Failed to trigger evaluation',
                      ),
                  })
                }
              />
            ))}
          </div>

          <ComplianceTrend frameworks={frameworks} />

          <OverdueControlsTable />
        </>
      )}

      <FrameworkManager open={managerOpen} onOpenChange={setManagerOpen} />
    </div>
  );
};
