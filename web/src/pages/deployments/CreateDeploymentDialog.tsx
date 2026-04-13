import { useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  Button,
  useTheme,
} from '@patchiq/ui';
import { useCan } from '../../app/auth/AuthContext';
import { usePolicies } from '../../api/hooks/usePolicies';
import { useCreateDeployment } from '../../api/hooks/useDeployments';
import type { WaveConfig } from '../../api/hooks/useDeployments';
import type { components } from '../../api/types';
import { PolicyModeBadge } from '../../components/PolicyModeBadge';

type CreateDeploymentBody = components['schemas']['CreateDeploymentRequest'];

interface CreateDeploymentDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated?: (id: string) => void;
}

const defaultWaveRow: WaveConfig = {
  percentage: 10,
  success_threshold: 95,
  error_rate_max: 5,
  delay_minutes: 30,
};

export const CreateDeploymentDialog = ({
  open,
  onOpenChange,
  onCreated,
}: CreateDeploymentDialogProps) => {
  const [policyId, setPolicyId] = useState('');
  const [description, setDescription] = useState('');
  const [targetEndpoints, setTargetEndpoints] = useState<number | undefined>(undefined);
  const [numWaves, setNumWaves] = useState<number>(1);
  const [confirming, setConfirming] = useState(false);
  const [showWaveConfig, setShowWaveConfig] = useState(false);
  const [waveRows, setWaveRows] = useState<WaveConfig[]>([]);
  const [maxConcurrent, setMaxConcurrent] = useState<number | undefined>(undefined);
  const [scheduledAt, setScheduledAt] = useState('');
  const [deployError, setDeployError] = useState<string | null>(null);
  const can = useCan();
  const { resolvedMode } = useTheme();
  const policies = usePolicies();
  const createDeployment = useCreateDeployment();

  const selectedPolicy = policies.data?.data?.find((p) => p.id === policyId);

  const resetForm = () => {
    setPolicyId('');
    setDescription('');
    setTargetEndpoints(undefined);
    setNumWaves(1);
    setConfirming(false);
    setShowWaveConfig(false);
    setWaveRows([]);
    setMaxConcurrent(undefined);
    setScheduledAt('');
    setDeployError(null);
  };

  const addWaveRow = () => {
    setWaveRows((prev) => [...prev, { ...defaultWaveRow }]);
  };

  const removeWaveRow = (index: number) => {
    setWaveRows((prev) => prev.filter((_, i) => i !== index));
  };

  const updateWaveRow = (index: number, field: keyof WaveConfig, value: number) => {
    setWaveRows((prev) => prev.map((row, i) => (i === index ? { ...row, [field]: value } : row)));
  };

  const handleDeploy = async () => {
    if (!confirming) {
      setConfirming(true);
      return;
    }
    const body: CreateDeploymentBody = { policy_id: policyId };
    if (showWaveConfig && waveRows.length > 0) {
      body.wave_config = waveRows;
    }
    if (maxConcurrent !== undefined && maxConcurrent > 0) {
      body.max_concurrent = maxConcurrent;
    }
    if (scheduledAt) {
      body.scheduled_at = new Date(scheduledAt).toISOString();
    }
    if (description.trim()) body.description = description.trim();
    try {
      setDeployError(null);
      const result = await createDeployment.mutateAsync(body);
      resetForm();
      onOpenChange(false);
      onCreated?.(result?.id ?? '');
    } catch (err) {
      setDeployError(err instanceof Error ? err.message : 'Deployment failed');
      setConfirming(false);
    }
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        onOpenChange(o);
        if (!o) resetForm();
      }}
    >
      <DialogContent className="max-h-[85vh] flex flex-col">
        <DialogHeader>
          <DialogTitle style={{ fontFamily: 'var(--font-display)' }}>New Deployment</DialogTitle>
        </DialogHeader>

        <div className="space-y-4 overflow-y-auto flex-1">
          <div>
            <label htmlFor="policy" className="text-sm font-medium">
              Policy
            </label>
            <select
              id="policy"
              value={policyId}
              onChange={(e) => {
                setPolicyId(e.target.value);
                setConfirming(false);
              }}
              className="mt-1 flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm"
              style={{ colorScheme: resolvedMode }}
            >
              <option value="">Select a policy...</option>
              {policies.data?.data?.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </select>
          </div>

          {selectedPolicy && (
            <div className="rounded-lg border bg-muted/30 p-4">
              <div className="flex items-center gap-2">
                <span className="font-medium">{selectedPolicy.name}</span>
                <PolicyModeBadge mode={selectedPolicy.selection_mode} />
              </div>
              <p className="mt-1 text-sm text-muted-foreground">
                Will deploy patches to endpoints matching this policy's tag selector.
              </p>
            </div>
          )}

          {/* Description */}
          <div>
            <label htmlFor="description" className="text-sm font-medium">
              Description (optional)
            </label>
            <textarea
              id="description"
              placeholder="Add notes or reason for this deployment..."
              value={description}
              onChange={(e) => {
                setDescription(e.target.value);
                setConfirming(false);
              }}
              className="mt-1 flex min-h-[60px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm resize-vertical"
            />
          </div>

          {/* Schedule */}
          <div>
            <label htmlFor="scheduled-at" className="text-sm font-medium">
              Schedule (optional)
            </label>
            <input
              id="scheduled-at"
              type="datetime-local"
              value={scheduledAt}
              onChange={(e) => {
                setScheduledAt(e.target.value);
                setConfirming(false);
              }}
              style={{ colorScheme: resolvedMode }}
              className="mt-1 flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm"
            />
          </div>

          {/* Max concurrent */}
          <div>
            <label htmlFor="max-concurrent" className="text-sm font-medium">
              Max Concurrent Endpoints (optional)
            </label>
            <input
              id="max-concurrent"
              type="number"
              min={1}
              placeholder="Unlimited"
              value={maxConcurrent ?? ''}
              onChange={(e) => {
                const val = e.target.value ? parseInt(e.target.value, 10) : undefined;
                setMaxConcurrent(val);
                setConfirming(false);
              }}
              className="mt-1 flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm"
            />
          </div>

          {/* Target Endpoints */}
          <div>
            <label htmlFor="target-endpoints" className="text-sm font-medium">
              Target Endpoints (optional)
            </label>
            <input
              id="target-endpoints"
              type="number"
              min={1}
              placeholder="All eligible endpoints"
              value={targetEndpoints ?? ''}
              onChange={(e) => {
                const val = e.target.value ? parseInt(e.target.value, 10) : undefined;
                setTargetEndpoints(val);
                setConfirming(false);
              }}
              className="mt-1 flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm"
            />
          </div>

          {/* Number of Waves */}
          <div>
            <label htmlFor="num-waves" className="text-sm font-medium">
              Number of Waves
            </label>
            <input
              id="num-waves"
              type="number"
              min={1}
              max={10}
              value={numWaves}
              onChange={(e) => {
                const val = Math.max(1, Math.min(10, parseInt(e.target.value, 10) || 1));
                setNumWaves(val);
                setConfirming(false);
              }}
              className="mt-1 flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm"
            />
          </div>

          {/* Wave config toggle */}
          <div>
            <label className="flex items-center gap-2 text-sm font-medium">
              <input
                type="checkbox"
                checked={showWaveConfig}
                onChange={(e) => {
                  setShowWaveConfig(e.target.checked);
                  if (e.target.checked && waveRows.length === 0) {
                    setWaveRows([{ ...defaultWaveRow }]);
                  }
                  if (!e.target.checked) setWaveRows([]);
                  setConfirming(false);
                }}
                className="rounded border-input"
              />
              Configure deployment waves
            </label>
            <p className="mt-1 text-xs text-muted-foreground">
              Without waves, all endpoints receive patches at once.
            </p>
          </div>

          {/* Wave rows */}
          {showWaveConfig && (
            <div className="space-y-3 rounded-lg border bg-muted/30 p-3">
              {waveRows.map((row, idx) => (
                <div
                  key={idx}
                  className="flex flex-wrap items-end gap-2 border-b pb-2 last:border-b-0 last:pb-0"
                >
                  <div className="w-16">
                    <label className="text-xs text-muted-foreground">%</label>
                    <input
                      type="number"
                      min={1}
                      max={100}
                      value={row.percentage}
                      onChange={(e) =>
                        updateWaveRow(idx, 'percentage', parseInt(e.target.value, 10) || 0)
                      }
                      className="flex h-8 w-full rounded-md border border-input bg-background px-2 text-sm"
                    />
                  </div>
                  <div className="w-20">
                    <label className="text-xs text-muted-foreground">Threshold%</label>
                    <input
                      type="number"
                      min={0}
                      max={100}
                      value={row.success_threshold}
                      onChange={(e) =>
                        updateWaveRow(idx, 'success_threshold', parseInt(e.target.value, 10) || 0)
                      }
                      className="flex h-8 w-full rounded-md border border-input bg-background px-2 text-sm"
                    />
                  </div>
                  <div className="w-20">
                    <label className="text-xs text-muted-foreground">Max Err%</label>
                    <input
                      type="number"
                      min={0}
                      max={100}
                      value={row.error_rate_max}
                      onChange={(e) =>
                        updateWaveRow(idx, 'error_rate_max', parseInt(e.target.value, 10) || 0)
                      }
                      className="flex h-8 w-full rounded-md border border-input bg-background px-2 text-sm"
                    />
                  </div>
                  <div className="w-20">
                    <label className="text-xs text-muted-foreground">Delay (min)</label>
                    <input
                      type="number"
                      min={0}
                      value={row.delay_minutes}
                      onChange={(e) =>
                        updateWaveRow(idx, 'delay_minutes', parseInt(e.target.value, 10) || 0)
                      }
                      className="flex h-8 w-full rounded-md border border-input bg-background px-2 text-sm"
                    />
                  </div>
                  <button
                    type="button"
                    onClick={() => removeWaveRow(idx)}
                    className="mb-0.5 text-xs text-red-500 hover:text-red-700"
                  >
                    Remove
                  </button>
                </div>
              ))}
              <button
                type="button"
                onClick={addWaveRow}
                className="text-xs font-medium text-blue-600 hover:text-blue-800 dark:text-blue-400"
              >
                + Add wave
              </button>
            </div>
          )}

          {confirming && (
            <div className="rounded-lg border border-amber-500/50 bg-amber-500/10 p-3 text-sm text-amber-700 dark:text-amber-300">
              {scheduledAt
                ? 'Are you sure? This deployment will be scheduled for the selected time.'
                : 'Are you sure? This will begin patch installation immediately.'}
            </div>
          )}
        </div>

        {deployError && <p className="text-sm text-destructive">{deployError}</p>}

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={handleDeploy}
            disabled={!policyId || createDeployment.isPending || !can('deployments', 'create')}
            title={!can('deployments', 'create') ? "You don't have permission" : undefined}
            style={!can('deployments', 'create') ? { opacity: 0.5 } : undefined}
          >
            {createDeployment.isPending
              ? 'Creating...'
              : confirming
                ? scheduledAt
                  ? 'Confirm Schedule'
                  : 'Confirm Deploy'
                : scheduledAt
                  ? 'Schedule'
                  : 'Deploy Now'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
