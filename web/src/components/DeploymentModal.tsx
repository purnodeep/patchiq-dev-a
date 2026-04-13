import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@patchiq/ui';
import { usePatchDeploy } from '../api/hooks/usePatchDeploy';
import { SeverityBadge } from './SeverityBadge';
import type { PatchListItem } from '../types/patches';

const schema = z.object({
  name: z.string().min(1, 'Deployment name is required'),
  description: z.string().optional(),
  config_type: z.enum(['install', 'rollback']),
  scope: z.string().min(1, 'Scope is required'),
  target_endpoints: z.string().min(1, 'Target endpoints are required'),
  start_date: z.string().optional(),
  start_time: z.string().optional(),
});

type FormValues = z.infer<typeof schema>;

interface DeploymentModalProps {
  open: boolean;
  onClose: () => void;
  patch?: PatchListItem | null;
  onSuccess?: () => void;
}

const inputStyle: React.CSSProperties = {
  width: '100%',
  padding: '8px 12px',
  fontSize: 12,
  fontFamily: 'var(--font-sans)',
  borderRadius: 6,
  border: '1px solid var(--border)',
  background: 'var(--bg-inset)',
  color: 'var(--text-primary)',
  outline: 'none',
  boxSizing: 'border-box',
  transition: 'border-color 0.15s ease',
};

const labelStyle: React.CSSProperties = {
  display: 'block',
  fontSize: 12,
  fontWeight: 600,
  fontFamily: 'var(--font-sans)',
  color: 'var(--text-primary)',
  marginBottom: 6,
};

const errorStyle: React.CSSProperties = {
  marginTop: 4,
  fontSize: 11,
  fontFamily: 'var(--font-sans)',
  color: 'var(--signal-critical)',
};

export const DeploymentModal = ({ open, onClose, patch, onSuccess }: DeploymentModalProps) => {
  const deploy = usePatchDeploy();

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: patch ? `${patch.name} - Deployment` : '',
      config_type: 'install',
      scope: '',
      target_endpoints: '',
    },
  });

  const handleClose = () => {
    reset();
    onClose();
  };

  const onPublish = handleSubmit(async (values) => {
    if (!patch) return;
    let scheduled_at: string | undefined;
    if (values.start_date && values.start_time) {
      scheduled_at = `${values.start_date}T${values.start_time}:00Z`;
    }
    try {
      await deploy.mutateAsync({
        patchId: patch.id,
        name: values.name,
        description: values.description,
        config_type: values.config_type,
        scope: values.scope,
        target_endpoints: values.target_endpoints,
        scheduled_at,
      });
      handleClose();
      onSuccess?.();
    } catch {
      // error handled by mutation state
    }
  });

  const onSaveDraft = handleSubmit(() => {
    handleClose();
  });

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        if (!o) handleClose();
      }}
    >
      <DialogContent
        className="max-w-[680px] max-h-[85vh] overflow-y-auto p-0"
        style={{ background: 'var(--bg-card)', border: '1px solid var(--border)' }}
      >
        <DialogHeader className="px-5 py-4" style={{ borderBottom: '1px solid var(--border)' }}>
          <DialogTitle
            style={{
              fontFamily: 'var(--font-sans)',
              fontSize: 15,
              fontWeight: 700,
              color: 'var(--text-emphasis)',
            }}
          >
            Create Patch Deployment
          </DialogTitle>
        </DialogHeader>

        <div style={{ padding: '20px', display: 'flex', flexDirection: 'column', gap: 16 }}>
          {/* Name */}
          <div>
            <label style={labelStyle}>
              Deployment Name <span style={{ color: 'var(--signal-critical)' }}>*</span>
            </label>
            <input
              {...register('name')}
              style={inputStyle}
              placeholder="e.g., KB5034441 - Critical Patch"
            />
            {errors.name && <p style={errorStyle}>{errors.name.message}</p>}
          </div>

          {/* Description */}
          <div>
            <label style={labelStyle}>Description</label>
            <textarea
              {...register('description')}
              style={{ ...inputStyle, minHeight: 70, resize: 'vertical' }}
              placeholder="Optional: Deployment notes, approval info..."
            />
          </div>

          {/* Config Type */}
          <div>
            <label style={labelStyle}>
              Configuration Type <span style={{ color: 'var(--signal-critical)' }}>*</span>
            </label>
            <div style={{ display: 'flex', gap: 16 }}>
              <label
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                  cursor: 'pointer',
                  fontSize: 12,
                  fontFamily: 'var(--font-sans)',
                  color: 'var(--text-primary)',
                }}
              >
                <input
                  type="radio"
                  {...register('config_type')}
                  value="install"
                  style={{ width: 14, height: 14, cursor: 'pointer' }}
                />
                Install
              </label>
              <label
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                  cursor: 'pointer',
                  fontSize: 12,
                  fontFamily: 'var(--font-sans)',
                  color: 'var(--text-primary)',
                }}
              >
                <input
                  type="radio"
                  {...register('config_type')}
                  value="rollback"
                  style={{ width: 14, height: 14, cursor: 'pointer' }}
                />
                Rollback
              </label>
            </div>
          </div>

          {/* Scope + Endpoints */}
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
            <div>
              <label style={labelStyle}>
                Scope <span style={{ color: 'var(--signal-critical)' }}>*</span>
              </label>
              <select {...register('scope')} style={{ ...inputStyle, cursor: 'pointer' }}>
                <option value="">Select Scope</option>
                <option value="prod">Production</option>
                <option value="staging">Staging</option>
                <option value="dev">Development</option>
              </select>
              {errors.scope && <p style={errorStyle}>{errors.scope.message}</p>}
            </div>
            <div>
              <label style={labelStyle}>
                Target Endpoints <span style={{ color: 'var(--signal-critical)' }}>*</span>
              </label>
              <select
                {...register('target_endpoints')}
                style={{ ...inputStyle, cursor: 'pointer' }}
              >
                <option value="">Please Select Endpoints</option>
                <option value="all">All Endpoints</option>
                <option value="windows">Windows Only</option>
                <option value="linux">Linux Only</option>
              </select>
              {errors.target_endpoints && (
                <p style={errorStyle}>{errors.target_endpoints.message}</p>
              )}
            </div>
          </div>

          {/* Patches */}
          <div>
            <label style={labelStyle}>
              Patches to Deploy <span style={{ color: 'var(--signal-critical)' }}>*</span>
            </label>
            <div
              style={{
                borderRadius: 6,
                border: '1px solid var(--border)',
                overflow: 'hidden',
                marginBottom: 8,
              }}
            >
              <table style={{ width: '100%', fontSize: 11, borderCollapse: 'collapse' }}>
                <thead
                  style={{
                    background: 'var(--bg-inset)',
                    borderBottom: '1px solid var(--border)',
                  }}
                >
                  <tr>
                    {['ID', 'Version', 'Severity', 'Remove'].map((h) => (
                      <th
                        key={h}
                        style={{
                          padding: '6px 8px',
                          textAlign: h === 'Remove' ? 'center' : 'left',
                          fontFamily: 'var(--font-mono)',
                          fontSize: 10,
                          fontWeight: 500,
                          textTransform: 'uppercase',
                          letterSpacing: '0.04em',
                          color: 'var(--text-muted)',
                        }}
                      >
                        {h}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {patch ? (
                    <tr>
                      <td
                        style={{
                          padding: '8px',
                          fontFamily: 'var(--font-mono)',
                          fontWeight: 600,
                          color: 'var(--accent)',
                          borderBottom: '1px solid var(--border)',
                        }}
                      >
                        {patch.name}
                      </td>
                      <td
                        style={{
                          padding: '8px',
                          fontFamily: 'var(--font-mono)',
                          fontSize: 11,
                          color: 'var(--text-muted)',
                          borderBottom: '1px solid var(--border)',
                        }}
                      >
                        {patch.version}
                      </td>
                      <td style={{ padding: '8px', borderBottom: '1px solid var(--border)' }}>
                        <SeverityBadge severity={patch.severity} />
                      </td>
                      <td
                        style={{
                          padding: '8px',
                          textAlign: 'center',
                          borderBottom: '1px solid var(--border)',
                        }}
                      >
                        <button
                          type="button"
                          onClick={handleClose}
                          style={{
                            background: 'none',
                            border: 'none',
                            cursor: 'pointer',
                            color: 'var(--signal-critical)',
                            fontSize: 16,
                            lineHeight: 1,
                          }}
                        >
                          ×
                        </button>
                      </td>
                    </tr>
                  ) : (
                    <tr>
                      <td
                        colSpan={4}
                        style={{
                          padding: '12px 8px',
                          textAlign: 'center',
                          fontFamily: 'var(--font-sans)',
                          fontSize: 12,
                          color: 'var(--text-muted)',
                        }}
                      >
                        No patch selected
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
            <button
              type="button"
              disabled
              style={{
                padding: '6px 12px',
                border: '1px dashed var(--border)',
                borderRadius: 6,
                background: 'transparent',
                color: 'var(--accent)',
                fontSize: 12,
                fontFamily: 'var(--font-sans)',
                fontWeight: 500,
                cursor: 'not-allowed',
                opacity: 0.5,
              }}
            >
              + Add More Patches
            </button>
          </div>

          {/* Schedule */}
          <div>
            <label style={labelStyle}>Schedule Deployment (Optional)</label>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
              <div>
                <label
                  style={{
                    ...labelStyle,
                    fontSize: 10,
                    color: 'var(--text-muted)',
                    fontWeight: 400,
                  }}
                >
                  Start Date
                </label>
                <input type="date" {...register('start_date')} style={inputStyle} />
              </div>
              <div>
                <label
                  style={{
                    ...labelStyle,
                    fontSize: 10,
                    color: 'var(--text-muted)',
                    fontWeight: 400,
                  }}
                >
                  Start Time
                </label>
                <input type="time" {...register('start_time')} style={inputStyle} />
              </div>
            </div>
          </div>
        </div>

        <DialogFooter
          style={{ borderTop: '1px solid var(--border)', padding: '14px 20px' }}
          className="flex gap-2 justify-end"
        >
          <button
            type="button"
            onClick={handleClose}
            style={{
              padding: '6px 12px',
              fontSize: 12,
              fontFamily: 'var(--font-sans)',
              borderRadius: 6,
              border: '1px solid var(--border)',
              background: 'transparent',
              color: 'var(--text-primary)',
              cursor: 'pointer',
              transition: 'background 0.1s ease',
            }}
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={onSaveDraft}
            style={{
              padding: '6px 12px',
              fontSize: 12,
              fontFamily: 'var(--font-sans)',
              borderRadius: 6,
              border: '1px solid var(--border)',
              background: 'transparent',
              color: 'var(--text-secondary)',
              cursor: 'pointer',
            }}
          >
            Save as Draft
          </button>
          <button
            type="button"
            onClick={onPublish}
            disabled={deploy.isPending}
            style={{
              padding: '6px 14px',
              fontSize: 12,
              fontFamily: 'var(--font-sans)',
              fontWeight: 600,
              borderRadius: 6,
              border: 'none',
              background: 'var(--accent)',
              color: 'var(--text-on-color, #fff)',
              cursor: deploy.isPending ? 'not-allowed' : 'pointer',
              opacity: deploy.isPending ? 0.6 : 1,
              transition: 'opacity 0.1s ease',
            }}
          >
            {deploy.isPending ? 'Publishing...' : 'Publish'}
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
