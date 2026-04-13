import { useRef, useEffect, useCallback } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import {
  Button,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@patchiq/ui';
import { EditorView, basicSetup } from 'codemirror';
import { EditorState } from '@codemirror/state';
import type { ScriptConfig } from '../types';

const schema = z.object({
  script_body: z.string().min(1, 'Script body is required'),
  script_type: z.enum(['shell', 'powershell']),
  timeout_minutes: z.number().min(1),
  failure_behavior: z.enum(['continue', 'halt']),
});

interface ScriptPanelProps {
  config: ScriptConfig;
  onSave: (config: ScriptConfig) => void;
}

export function ScriptPanel({ config, onSave }: ScriptPanelProps) {
  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors },
  } = useForm<ScriptConfig>({
    resolver: zodResolver(schema),
    defaultValues: config,
  });

  const editorRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);

  const handleDocChange = useCallback(
    (doc: string) => {
      setValue('script_body', doc, { shouldValidate: true });
    },
    [setValue],
  );

  useEffect(() => {
    if (!editorRef.current || viewRef.current) return;

    const state = EditorState.create({
      doc: config.script_body ?? '',
      extensions: [
        basicSetup,
        EditorView.updateListener.of((update) => {
          if (update.docChanged) {
            handleDocChange(update.state.doc.toString());
          }
        }),
      ],
    });

    viewRef.current = new EditorView({
      state,
      parent: editorRef.current,
    });

    return () => {
      viewRef.current?.destroy();
      viewRef.current = null;
    };
  }, [config.script_body, handleDocChange]);

  return (
    <form onSubmit={handleSubmit(onSave)} className="space-y-4">
      <div>
        <label htmlFor="script_type" className="block text-sm font-medium mb-1">
          Script Type
        </label>
        <Select
          value={watch('script_type')}
          onValueChange={(v) => setValue('script_type', v as ScriptConfig['script_type'])}
        >
          <SelectTrigger id="script_type">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="shell">Shell</SelectItem>
            <SelectItem value="powershell">PowerShell</SelectItem>
          </SelectContent>
        </Select>
        <p className="text-xs text-muted-foreground mt-1">
          Shell for Linux/macOS agents, PowerShell for Windows agents.
        </p>
      </div>

      <div>
        <label className="block text-sm font-medium mb-1">Script Body</label>
        <div
          ref={editorRef}
          data-testid="script-editor"
          className="min-h-[120px] rounded-md border border-input overflow-hidden"
        />
        <p className="text-xs text-muted-foreground mt-1">
          The script to execute on target endpoints. Runs with agent-level permissions.
        </p>
        {errors.script_body && (
          <p className="text-sm text-destructive">{errors.script_body.message}</p>
        )}
      </div>

      <div>
        <label htmlFor="timeout_minutes" className="block text-sm font-medium mb-1">
          Timeout (minutes)
        </label>
        <Input
          id="timeout_minutes"
          type="number"
          {...register('timeout_minutes', { valueAsNumber: true })}
        />
        <p className="text-xs text-muted-foreground mt-1">
          Max execution time before the script is forcibly terminated.
        </p>
        {errors.timeout_minutes && (
          <p className="text-sm text-destructive">{errors.timeout_minutes.message}</p>
        )}
      </div>

      <div>
        <label htmlFor="failure_behavior" className="block text-sm font-medium mb-1">
          On Failure
        </label>
        <Select
          value={watch('failure_behavior')}
          onValueChange={(v) => setValue('failure_behavior', v as ScriptConfig['failure_behavior'])}
        >
          <SelectTrigger id="failure_behavior">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="continue">Continue</SelectItem>
            <SelectItem value="halt">Halt</SelectItem>
          </SelectContent>
        </Select>
        <p className="text-xs text-muted-foreground mt-1">
          Continue proceeds to the next node on failure. Halt stops the workflow.
        </p>
      </div>

      <Button type="submit">Save</Button>
    </form>
  );
}
