package workflow

import "encoding/json"

// WorkflowTemplate is a preset workflow that users can clone.
type WorkflowTemplate struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Nodes       []Node `json:"nodes"`
	Edges       []Edge `json:"edges"`
}

// mustMarshal marshals v to JSON or panics. Only use with types known to marshal successfully.
func mustMarshal(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

// AllTemplates returns all preset workflow templates.
func AllTemplates() []WorkflowTemplate {
	return []WorkflowTemplate{
		criticalFastTrack(),
		standardApprovalFlow(),
		canaryDeployment(),
	}
}

func criticalFastTrack() WorkflowTemplate {
	return WorkflowTemplate{
		ID:          "critical-fast-track",
		Name:        "Critical Patch Fast-Track",
		Description: "Immediately deploy critical CVE patches and notify the team",
		Nodes: []Node{
			{ID: "t1", NodeType: NodeTrigger, Label: "Critical CVE Detected", PositionX: 0, PositionY: 100,
				Config: mustMarshal(TriggerConfig{TriggerType: "cve_severity", SeverityThreshold: "critical"})},
			{ID: "w1", NodeType: NodeDeploymentWave, Label: "Deploy All", PositionX: 250, PositionY: 100,
				Config: mustMarshal(DeploymentWaveConfig{Percentage: 100, MaxParallel: 50, TimeoutMinutes: 60, SuccessThreshold: 95})},
			{ID: "n1", NodeType: NodeNotification, Label: "Notify Team", PositionX: 500, PositionY: 100,
				Config: mustMarshal(NotificationConfig{Channel: "slack", Target: "#security-ops", MessageTemplate: "Critical patch deployed"})},
			{ID: "c1", NodeType: NodeComplete, Label: "Done", PositionX: 750, PositionY: 100,
				Config: mustMarshal(CompleteConfig{GenerateReport: true})},
		},
		Edges: []Edge{
			{SourceNodeID: "t1", TargetNodeID: "w1"},
			{SourceNodeID: "w1", TargetNodeID: "n1"},
			{SourceNodeID: "n1", TargetNodeID: "c1"},
		},
	}
}

func standardApprovalFlow() WorkflowTemplate {
	return WorkflowTemplate{
		ID:          "standard-approval-flow",
		Name:        "Standard Approval Flow",
		Description: "Filter targets, get approval, deploy, and notify",
		Nodes: []Node{
			{ID: "t1", NodeType: NodeTrigger, Label: "Manual Trigger", PositionX: 0, PositionY: 100,
				Config: mustMarshal(TriggerConfig{TriggerType: "manual"})},
			{ID: "f1", NodeType: NodeFilter, Label: "Select Targets", PositionX: 200, PositionY: 100,
				Config: mustMarshal(FilterConfig{OSTypes: []string{"linux", "windows"}})},
			{ID: "a1", NodeType: NodeApproval, Label: "Manager Approval", PositionX: 400, PositionY: 100,
				Config: mustMarshal(ApprovalConfig{ApproverRoles: []string{"admin"}, TimeoutHours: 24})},
			{ID: "w1", NodeType: NodeDeploymentWave, Label: "Deploy All", PositionX: 600, PositionY: 100,
				Config: mustMarshal(DeploymentWaveConfig{Percentage: 100, MaxParallel: 20, TimeoutMinutes: 120, SuccessThreshold: 95})},
			{ID: "n1", NodeType: NodeNotification, Label: "Notify Stakeholders", PositionX: 800, PositionY: 100,
				Config: mustMarshal(NotificationConfig{Channel: "email", Target: "ops@example.com", MessageTemplate: "Deployment complete"})},
			{ID: "c1", NodeType: NodeComplete, Label: "Done", PositionX: 1000, PositionY: 100,
				Config: mustMarshal(CompleteConfig{GenerateReport: true, NotifyOnComplete: true})},
		},
		Edges: []Edge{
			{SourceNodeID: "t1", TargetNodeID: "f1"},
			{SourceNodeID: "f1", TargetNodeID: "a1"},
			{SourceNodeID: "a1", TargetNodeID: "w1"},
			{SourceNodeID: "w1", TargetNodeID: "n1"},
			{SourceNodeID: "n1", TargetNodeID: "c1"},
		},
	}
}

func canaryDeployment() WorkflowTemplate {
	return WorkflowTemplate{
		ID:          "canary-deployment",
		Name:        "Canary Deployment",
		Description: "Progressive rollout: 10% canary, then 50%, then 100% with health gates",
		Nodes: []Node{
			{ID: "t1", NodeType: NodeTrigger, Label: "Manual Trigger", PositionX: 0, PositionY: 100,
				Config: mustMarshal(TriggerConfig{TriggerType: "manual"})},
			{ID: "f1", NodeType: NodeFilter, Label: "Select Targets", PositionX: 200, PositionY: 100,
				Config: mustMarshal(FilterConfig{OSTypes: []string{"linux"}})},
			{ID: "w1", NodeType: NodeDeploymentWave, Label: "Canary 10%", PositionX: 400, PositionY: 100,
				Config: mustMarshal(DeploymentWaveConfig{Percentage: 10, MaxParallel: 5, TimeoutMinutes: 30, SuccessThreshold: 100})},
			{ID: "g1", NodeType: NodeGate, Label: "Health Check 4h", PositionX: 600, PositionY: 100,
				Config: mustMarshal(GateConfig{WaitMinutes: 240, FailureThreshold: 5, HealthCheck: true})},
			{ID: "w2", NodeType: NodeDeploymentWave, Label: "Expand 50%", PositionX: 800, PositionY: 100,
				Config: mustMarshal(DeploymentWaveConfig{Percentage: 50, MaxParallel: 20, TimeoutMinutes: 60, SuccessThreshold: 95})},
			{ID: "g2", NodeType: NodeGate, Label: "Health Check 2h", PositionX: 1000, PositionY: 100,
				Config: mustMarshal(GateConfig{WaitMinutes: 120, FailureThreshold: 10, HealthCheck: true})},
			{ID: "w3", NodeType: NodeDeploymentWave, Label: "Full Rollout 100%", PositionX: 1200, PositionY: 100,
				Config: mustMarshal(DeploymentWaveConfig{Percentage: 100, MaxParallel: 50, TimeoutMinutes: 120, SuccessThreshold: 90})},
			{ID: "c1", NodeType: NodeComplete, Label: "Done", PositionX: 1400, PositionY: 100,
				Config: mustMarshal(CompleteConfig{GenerateReport: true})},
		},
		Edges: []Edge{
			{SourceNodeID: "t1", TargetNodeID: "f1"},
			{SourceNodeID: "f1", TargetNodeID: "w1"},
			{SourceNodeID: "w1", TargetNodeID: "g1"},
			{SourceNodeID: "g1", TargetNodeID: "w2"},
			{SourceNodeID: "w2", TargetNodeID: "g2"},
			{SourceNodeID: "g2", TargetNodeID: "w3"},
			{SourceNodeID: "w3", TargetNodeID: "c1"},
		},
	}
}
