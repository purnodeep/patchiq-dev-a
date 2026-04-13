export type ExecutionStatus =
  | 'pending'
  | 'running'
  | 'paused'
  | 'completed'
  | 'failed'
  | 'cancelled';

export type NodeExecutionStatus = 'pending' | 'running' | 'completed' | 'failed' | 'skipped';

export type WorkflowNodeType =
  | 'trigger'
  | 'filter'
  | 'approval'
  | 'deployment_wave'
  | 'gate'
  | 'script'
  | 'notification'
  | 'rollback'
  | 'decision'
  | 'complete'
  | 'reboot'
  | 'scan'
  | 'tag_gate'
  | 'compliance_check';

export interface TriggerConfig {
  trigger_type: 'manual' | 'cron' | 'cve_severity' | 'policy_evaluation';
  cron_expression?: string;
  severity_threshold?: string;
}

export interface FilterConfig {
  os_types?: string[];
  // Tag predicates formatted as "key=value" strings. Multiple entries
  // compose with AND — an endpoint must carry every tag to pass. The
  // backend rejects malformed entries (missing "=") at save and runtime.
  tags?: string[];
  min_severity?: string;
  package_regex?: string;
}

export interface ApprovalConfig {
  approver_roles: string[];
  timeout_hours: number;
  escalation_role?: string;
  timeout_action?: 'reject' | 'escalate';
}

export interface DeploymentWaveConfig {
  percentage: number;
  max_parallel?: number;
  timeout_minutes?: number;
  success_threshold?: number;
}

export interface GateConfig {
  wait_minutes: number;
  failure_threshold: number;
  health_check?: boolean;
}

export interface ScriptConfig {
  script_body: string;
  script_type: 'shell' | 'powershell';
  timeout_minutes: number;
  failure_behavior: 'continue' | 'halt';
}

export interface NotificationConfig {
  channel: 'email' | 'slack' | 'webhook' | 'pagerduty';
  target: string;
  message_template?: string;
}

export interface RollbackConfig {
  strategy: 'snapshot_restore' | 'package_downgrade' | 'script';
  failure_threshold: number;
  rollback_script?: string;
  target_deployment?: string;
}

export interface DecisionConfig {
  field: string;
  operator: 'equals' | 'not_equals' | 'in' | 'gt' | 'lt';
  value: string;
  true_label?: string;
  false_label?: string;
}

export interface CompleteConfig {
  generate_report?: boolean;
  notify_on_complete?: boolean;
}

export interface RebootConfig {
  timeout_minutes: number;
  force_reboot?: boolean;
  failure_behavior: 'continue' | 'halt';
}

export interface ScanConfig {
  scan_type: 'inventory' | 'vulnerability' | 'compliance';
  timeout_minutes: number;
  failure_behavior: 'continue' | 'halt';
}

export interface TagGateConfig {
  required_tags: string[];
  match_mode: 'all' | 'any';
}

export interface ComplianceCheckConfig {
  framework: 'CIS' | 'PCI-DSS' | 'HIPAA' | 'NIST' | 'ISO27001' | 'SOC2';
  min_score: number;
  failure_behavior: 'continue' | 'halt';
}

export type NodeConfig =
  | TriggerConfig
  | FilterConfig
  | ApprovalConfig
  | DeploymentWaveConfig
  | GateConfig
  | ScriptConfig
  | NotificationConfig
  | RollbackConfig
  | DecisionConfig
  | CompleteConfig
  | RebootConfig
  | ScanConfig
  | TagGateConfig
  | ComplianceCheckConfig;

export interface WorkflowNodeRequest {
  id: string;
  node_type: WorkflowNodeType;
  label: string;
  position_x: number;
  position_y: number;
  config: NodeConfig;
}

export interface WorkflowEdgeRequest {
  source_node_id: string;
  target_node_id: string;
  label: string;
}

export interface WorkflowRequest {
  name: string;
  description?: string;
  nodes: WorkflowNodeRequest[];
  edges: WorkflowEdgeRequest[];
}

export interface WorkflowResponse {
  id: string;
  version_id: string;
  version: number;
  status: string;
}

export interface WorkflowListItem {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  created_at: string;
  updated_at: string;
  current_version: number;
  current_status: 'draft' | 'published' | 'archived';
  node_count: number;
  total_runs: number;
  last_run_status: string | null;
  last_run_at: string | null;
}

export interface WorkflowNode {
  id: string;
  version_id: string;
  tenant_id: string;
  node_type: WorkflowNodeType;
  label: string;
  position_x: number;
  position_y: number;
  config: NodeConfig;
  created_at: string;
}

export interface WorkflowEdge {
  id: string;
  version_id: string;
  tenant_id: string;
  source_node_id: string;
  target_node_id: string;
  label: string;
  created_at: string;
}

export interface WorkflowVersion {
  id: string;
  workflow_id: string;
  tenant_id: string;
  version: number;
  status: 'draft' | 'published' | 'archived';
  created_at: string;
}

export interface WorkflowDetail {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  created_at: string;
  updated_at: string;
  version?: WorkflowVersion;
  nodes: WorkflowNode[];
  edges: WorkflowEdge[];
}

export interface WorkflowTemplate {
  id: string;
  name: string;
  description: string;
  nodes: Array<{
    id: string;
    node_type: WorkflowNodeType;
    label: string;
    position_x: number;
    position_y: number;
    config: NodeConfig;
  }>;
  edges: Array<{
    source_node_id: string;
    target_node_id: string;
    label: string;
  }>;
}

export interface WorkflowExecution {
  id: string;
  workflow_id: string;
  version_id: string;
  tenant_id: string;
  status: ExecutionStatus;
  triggered_by: string;
  current_node_id: string | null;
  error_message: string;
  started_at: string | null;
  completed_at: string | null;
  created_at: string;
}

export interface NodeExecution {
  id: string;
  execution_id: string;
  node_id: string;
  tenant_id: string;
  status: NodeExecutionStatus;
  output: Record<string, unknown> | null;
  error_message: string;
  started_at: string | null;
  completed_at: string | null;
}

export interface ExecutionDetail {
  id: string;
  workflow_id: string;
  version_id: string;
  tenant_id: string;
  status: ExecutionStatus;
  triggered_by: string;
  current_node_id: string | null;
  error_message: string;
  started_at: string | null;
  completed_at: string | null;
  created_at: string;
  node_executions: NodeExecution[];
}

export interface PaginatedList<T> {
  data: T[];
  next_cursor: string | null;
  total_count: number;
}
