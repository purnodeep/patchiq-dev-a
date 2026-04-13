import { useState, useEffect } from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  Button,
  useTheme,
} from '@patchiq/ui';

interface PatchDeploymentDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  patch?: { id: string; name: string; version: string; severity: string; os_family: string } | null;
}

const inputClass =
  'flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm focus:outline-none focus:ring-1 focus:ring-ring';

const selectClass =
  'flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm focus:outline-none focus:ring-1 focus:ring-ring cursor-pointer';

const textareaClass =
  'flex min-h-[70px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm resize-vertical focus:outline-none focus:ring-1 focus:ring-ring';

const labelClass = 'block text-xs font-semibold mb-1.5';

export const PatchDeploymentDialog = ({
  open,
  onOpenChange,
  patch,
}: PatchDeploymentDialogProps) => {
  const { resolvedMode } = useTheme();
  const [deploymentName, setDeploymentName] = useState('');
  const [description, setDescription] = useState('');
  const [configType, setConfigType] = useState<'install' | 'rollback'>('install');
  const [scope, setScope] = useState('');
  const [targetEndpoints, setTargetEndpoints] = useState('');
  const [schedule, setSchedule] = useState<'immediate' | 'scheduled'>('immediate');
  const [scheduledAt, setScheduledAt] = useState('');
  const [nameError, setNameError] = useState(false);

  // Auto-fill deployment name when patch changes
  useEffect(() => {
    if (patch) {
      setDeploymentName(`${patch.name} - Deployment`);
    }
  }, [patch]);

  // Reset form when dialog closes
  useEffect(() => {
    if (!open) {
      setDeploymentName('');
      setDescription('');
      setConfigType('install');
      setScope('');
      setTargetEndpoints('');
      setSchedule('immediate');
      setScheduledAt('');
      setNameError(false);
    }
  }, [open]);

  const handleClose = () => {
    onOpenChange(false);
  };

  const handleSaveDraft = () => {
    handleClose();
  };

  const handlePublish = () => {
    if (!deploymentName.trim()) {
      setNameError(true);
      return;
    }
    handleClose();
  };

  const severityColor = (severity: string) => {
    switch (severity.toLowerCase()) {
      case 'critical':
        return 'text-red-500';
      case 'high':
        return 'text-orange-500';
      case 'medium':
        return 'text-yellow-500';
      case 'low':
        return 'text-blue-400';
      default:
        return 'text-muted-foreground';
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg max-h-[90vh] flex flex-col overflow-hidden">
        <DialogHeader>
          <DialogTitle>Create Patch Deployment</DialogTitle>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto px-1 space-y-4 py-2">
          {/* Deployment Name */}
          <div>
            <label className={labelClass}>
              Deployment Name <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={deploymentName}
              onChange={(e) => {
                setDeploymentName(e.target.value);
                if (e.target.value.trim()) setNameError(false);
              }}
              placeholder="e.g., KB5034441 - Critical Patch"
              className={`${inputClass} ${nameError ? 'border-red-500' : ''}`}
            />
            {nameError && <p className="mt-1 text-xs text-red-500">Deployment Name is required.</p>}
          </div>

          {/* Description */}
          <div>
            <label className={labelClass}>Description</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Optional: Deployment notes, approval info..."
              className={textareaClass}
            />
          </div>

          {/* Configuration Type */}
          <div>
            <label className={labelClass}>
              Configuration Type <span className="text-red-500">*</span>
            </label>
            <div className="flex gap-4">
              <label className="flex items-center gap-1.5 cursor-pointer text-sm">
                <input
                  type="radio"
                  name="configType"
                  value="install"
                  checked={configType === 'install'}
                  onChange={() => setConfigType('install')}
                  className="w-4 h-4 cursor-pointer"
                />
                Install
              </label>
              <label className="flex items-center gap-1.5 cursor-pointer text-sm">
                <input
                  type="radio"
                  name="configType"
                  value="rollback"
                  checked={configType === 'rollback'}
                  onChange={() => setConfigType('rollback')}
                  className="w-4 h-4 cursor-pointer"
                />
                Rollback
              </label>
            </div>
          </div>

          {/* Scope & Target Endpoints */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className={labelClass}>
                Scope <span className="text-red-500">*</span>
              </label>
              <select
                value={scope}
                onChange={(e) => setScope(e.target.value)}
                className={selectClass}
                style={{ colorScheme: resolvedMode }}
              >
                <option value="">Select Scope</option>
                <option value="production">Production</option>
                <option value="staging">Staging</option>
                <option value="development">Development</option>
              </select>
            </div>
            <div>
              <label className={labelClass}>
                Target Endpoints <span className="text-red-500">*</span>
              </label>
              <input
                type="number"
                value={targetEndpoints}
                onChange={(e) => setTargetEndpoints(e.target.value)}
                placeholder="0"
                min={0}
                className={inputClass}
              />
            </div>
          </div>

          {/* Schedule */}
          <div>
            <label className={labelClass}>Schedule</label>
            <select
              value={schedule}
              onChange={(e) => setSchedule(e.target.value as 'immediate' | 'scheduled')}
              className={`${selectClass} mb-2`}
              style={{ colorScheme: resolvedMode }}
            >
              <option value="immediate">Immediate</option>
              <option value="scheduled">Scheduled</option>
            </select>
            {schedule === 'scheduled' && (
              <input
                type="datetime-local"
                value={scheduledAt}
                onChange={(e) => setScheduledAt(e.target.value)}
                className={inputClass}
                style={{ colorScheme: resolvedMode }}
              />
            )}
          </div>

          {/* Patches to Deploy */}
          <div>
            <label className={labelClass}>
              Patches to Deploy <span className="text-red-500">*</span>
            </label>
            <div className="rounded-md border border-border overflow-hidden text-xs">
              <table className="w-full border-collapse">
                <thead>
                  <tr className="border-b border-border bg-muted/40">
                    <th className="px-3 py-2 text-left text-muted-foreground font-semibold">
                      Name
                    </th>
                    <th className="px-3 py-2 text-left text-muted-foreground font-semibold">
                      Version
                    </th>
                    <th className="px-3 py-2 text-left text-muted-foreground font-semibold">
                      Severity
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {patch ? (
                    <tr className="border-b border-border last:border-0">
                      <td className="px-3 py-2 font-medium">{patch.name}</td>
                      <td className="px-3 py-2 text-muted-foreground">{patch.version}</td>
                      <td
                        className={`px-3 py-2 font-semibold capitalize ${severityColor(patch.severity)}`}
                      >
                        {patch.severity}
                      </td>
                    </tr>
                  ) : (
                    <tr>
                      <td colSpan={3} className="px-3 py-3 text-center text-muted-foreground">
                        No patch selected
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </div>

        <DialogFooter className="gap-2 pt-2">
          <Button variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button variant="ghost" onClick={handleSaveDraft}>
            Save as Draft
          </Button>
          <Button onClick={handlePublish} disabled={!deploymentName.trim()}>
            Publish
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
