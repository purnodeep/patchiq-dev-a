import { Controller, useFormContext } from 'react-hook-form';
import { Switch } from '@patchiq/ui';
import { TagSelectorBuilder } from '../targeting/TagSelectorBuilder';
import type { Selector } from '../../types/targeting';
import type { PolicyWizardValues } from './types';
import { LABEL_STYLE, TOGGLE_CARD } from './types';

export function TargetsStep() {
  const { watch, setValue, control } = useFormContext<PolicyWizardValues>();
  const respectMW = watch('respect_maintenance_window');
  const onlineOnly = watch('online_only');

  return (
    <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 18 }}>
      <div>
        <label style={LABEL_STYLE}>Tag selector</label>
        <Controller
          name="target_selector"
          control={control}
          render={({ field }) => (
            <TagSelectorBuilder
              value={field.value as Selector | null}
              onChange={(next) => field.onChange(next)}
            />
          )}
        />
      </div>

      <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        <div style={TOGGLE_CARD}>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              gap: 12,
            }}
          >
            <div>
              <div
                style={{
                  fontSize: 12,
                  fontWeight: 500,
                  color: 'var(--text-primary)',
                  marginBottom: 2,
                }}
              >
                Respect Maintenance Windows
              </div>
              <div style={{ fontSize: 10, color: 'var(--text-muted)' }}>
                Only deploy during configured maintenance windows
              </div>
            </div>
            <Switch
              checked={respectMW}
              onCheckedChange={(checked) => setValue('respect_maintenance_window', checked)}
            />
          </div>
        </div>

        <div style={TOGGLE_CARD}>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              gap: 12,
            }}
          >
            <div>
              <div
                style={{
                  fontSize: 12,
                  fontWeight: 500,
                  color: 'var(--text-primary)',
                  marginBottom: 2,
                }}
              >
                Online Endpoints Only
              </div>
              <div style={{ fontSize: 10, color: 'var(--text-muted)' }}>
                Skip endpoints that are currently offline
              </div>
            </div>
            <Switch
              checked={onlineOnly}
              onCheckedChange={(checked) => setValue('online_only', checked)}
            />
          </div>
        </div>
      </div>
    </div>
  );
}
