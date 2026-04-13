import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@patchiq/ui';
import { NODE_TYPE_REGISTRY } from '../node-types';
import type {
  WorkflowNodeType,
  NodeConfig,
  TriggerConfig,
  FilterConfig,
  ApprovalConfig,
  DeploymentWaveConfig,
  GateConfig,
  ScriptConfig,
  NotificationConfig,
  RollbackConfig,
  DecisionConfig,
  CompleteConfig,
  RebootConfig,
  ScanConfig,
  TagGateConfig,
  ComplianceCheckConfig,
} from '../types';
import { TriggerPanel } from './trigger-panel';
import { FilterPanel } from './filter-panel';
import { ApprovalPanel } from './approval-panel';
import { WavePanel } from './wave-panel';
import { GatePanel } from './gate-panel';
import { ScriptPanel } from './script-panel';
import { NotificationPanel } from './notification-panel';
import { RollbackPanel } from './rollback-panel';
import { DecisionPanel } from './decision-panel';
import { CompletePanel } from './complete-panel';
import { RebootPanel } from './reboot-panel';
import { ScanPanel } from './scan-panel';
import { TagGatePanel } from './tag-gate-panel';
import { ComplianceCheckPanel } from './compliance-check-panel';

interface ConfigPanelProps {
  nodeType: WorkflowNodeType;
  nodeLabel: string;
  config: NodeConfig;
  open: boolean;
  onClose: () => void;
  onSave: (config: NodeConfig) => void;
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any -- panel configs are narrowed per node type
type AnySave = (c: any) => void;

function renderPanel(
  nodeType: WorkflowNodeType,
  config: NodeConfig,
  onSave: (config: NodeConfig) => void,
) {
  const save = onSave as AnySave;
  switch (nodeType) {
    case 'trigger':
      return <TriggerPanel config={config as TriggerConfig} onSave={save} />;
    case 'filter':
      return <FilterPanel config={config as FilterConfig} onSave={save} />;
    case 'approval':
      return <ApprovalPanel config={config as ApprovalConfig} onSave={save} />;
    case 'deployment_wave':
      return <WavePanel config={config as DeploymentWaveConfig} onSave={save} />;
    case 'gate':
      return <GatePanel config={config as GateConfig} onSave={save} />;
    case 'script':
      return <ScriptPanel config={config as ScriptConfig} onSave={save} />;
    case 'notification':
      return <NotificationPanel config={config as NotificationConfig} onSave={save} />;
    case 'rollback':
      return <RollbackPanel config={config as RollbackConfig} onSave={save} />;
    case 'decision':
      return <DecisionPanel config={config as DecisionConfig} onSave={save} />;
    case 'complete':
      return <CompletePanel config={config as CompleteConfig} onSave={save} />;
    case 'reboot':
      return <RebootPanel config={config as RebootConfig} onSave={save} />;
    case 'scan':
      return <ScanPanel config={config as ScanConfig} onSave={save} />;
    case 'tag_gate':
      return <TagGatePanel config={config as TagGateConfig} onSave={save} />;
    case 'compliance_check':
      return <ComplianceCheckPanel config={config as ComplianceCheckConfig} onSave={save} />;
    default:
      return <div className="text-sm text-destructive">Unknown node type: {String(nodeType)}</div>;
  }
}

export function ConfigPanel({
  nodeType,
  nodeLabel,
  config,
  open,
  onClose,
  onSave,
}: ConfigPanelProps) {
  const info = NODE_TYPE_REGISTRY[nodeType];
  const Icon = info.icon;

  return (
    <Sheet
      open={open}
      onOpenChange={(isOpen) => {
        if (!isOpen) onClose();
      }}
    >
      <SheetContent className="w-[400px] sm:w-[540px] overflow-y-auto">
        <SheetHeader>
          <SheetTitle className="flex items-center gap-2">
            <Icon className={`h-5 w-5 ${info.color}`} />
            {nodeLabel}
          </SheetTitle>
          {info.description && (
            <p className="text-xs text-muted-foreground leading-relaxed mt-1">{info.description}</p>
          )}
        </SheetHeader>
        <div className="mt-4 px-4">{renderPanel(nodeType, config, onSave)}</div>
      </SheetContent>
    </Sheet>
  );
}
