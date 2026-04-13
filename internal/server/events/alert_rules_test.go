package events

import "testing"

func TestRenderTemplate(t *testing.T) {
	tests := []struct {
		name     string
		tmpl     string
		data     map[string]any
		fallback string
		want     string
	}{
		{
			name:     "simple substitution",
			tmpl:     "Deployment failed: {{.name}}",
			data:     map[string]any{"name": "Q1-Rollout"},
			fallback: "deployment.failed",
			want:     "Deployment failed: Q1-Rollout",
		},
		{
			name:     "missing field renders as empty string",
			tmpl:     "Hello {{.name}}",
			data:     map[string]any{},
			fallback: "deployment.failed",
			want:     "Hello ",
		},
		{
			name:     "nil data renders missing fields as empty",
			tmpl:     "Hello {{.name}}",
			data:     nil,
			fallback: "deployment.failed",
			want:     "Hello ",
		},
		{
			name:     "invalid template uses fallback",
			tmpl:     "{{.name",
			data:     map[string]any{"name": "test"},
			fallback: "deployment.failed",
			want:     "deployment.failed",
		},
		{
			name:     "multiple fields",
			tmpl:     "{{.name}} has {{.count}} patches",
			data:     map[string]any{"name": "server-01", "count": 5},
			fallback: "deployment.failed",
			want:     "server-01 has 5 patches",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := renderTemplate(tc.tmpl, tc.data, tc.fallback)
			if got != tc.want {
				t.Errorf("renderTemplate(%q, %v, %q) = %q; want %q",
					tc.tmpl, tc.data, tc.fallback, got, tc.want)
			}
		})
	}
}

func TestIsAlertEventType(t *testing.T) {
	tests := []struct {
		eventType string
		want      bool
	}{
		{"alert.created", true},
		{"alert.status_updated", true},
		{"alert_rule.created", true},
		{"alert_rule.updated", true},
		{"alert_rule.deleted", true},
		{"deployment.failed", false},
		{"endpoint.enrolled", false},
	}

	for _, tc := range tests {
		t.Run(tc.eventType, func(t *testing.T) {
			got := isAlertEventType(tc.eventType)
			if got != tc.want {
				t.Errorf("isAlertEventType(%q) = %v; want %v", tc.eventType, got, tc.want)
			}
		})
	}
}
