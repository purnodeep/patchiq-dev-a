package tags

import (
	"encoding/json"
	"testing"
)

func TestEvaluateSimpleTag(t *testing.T) {
	expr := &Expression{Tag: "env", Value: "production"}
	tags := map[string][]string{"env": {"production"}}

	if !Evaluate(expr, tags) {
		t.Error("expected match for env:production")
	}
}

func TestEvaluateSimpleTagNoMatch(t *testing.T) {
	expr := &Expression{Tag: "env", Value: "production"}
	tags := map[string][]string{"env": {"staging"}}

	if Evaluate(expr, tags) {
		t.Error("expected no match for env:staging against env:production")
	}
}

func TestEvaluateAND(t *testing.T) {
	expr := &Expression{
		Op: "AND",
		Conditions: []*Expression{
			{Tag: "env", Value: "production"},
			{Tag: "os", Value: "linux"},
		},
	}
	tags := map[string][]string{
		"env": {"production"},
		"os":  {"linux"},
	}

	if !Evaluate(expr, tags) {
		t.Error("expected AND match when both conditions met")
	}
}

func TestEvaluateANDPartialMatch(t *testing.T) {
	expr := &Expression{
		Op: "AND",
		Conditions: []*Expression{
			{Tag: "env", Value: "production"},
			{Tag: "os", Value: "windows"},
		},
	}
	tags := map[string][]string{
		"env": {"production"},
		"os":  {"linux"},
	}

	if Evaluate(expr, tags) {
		t.Error("expected no match when only one AND condition met")
	}
}

func TestEvaluateOR(t *testing.T) {
	expr := &Expression{
		Op: "OR",
		Conditions: []*Expression{
			{Tag: "env", Value: "production"},
			{Tag: "env", Value: "staging"},
		},
	}
	tags := map[string][]string{"env": {"staging"}}

	if !Evaluate(expr, tags) {
		t.Error("expected OR match when one condition met")
	}
}

func TestEvaluateNOT(t *testing.T) {
	expr := &Expression{
		Op: "NOT",
		Conditions: []*Expression{
			{Tag: "wave", Value: "canary"},
		},
	}
	tags := map[string][]string{"env": {"production"}}

	if !Evaluate(expr, tags) {
		t.Error("expected NOT match when tag absent")
	}
}

func TestEvaluateNOTNegative(t *testing.T) {
	expr := &Expression{
		Op: "NOT",
		Conditions: []*Expression{
			{Tag: "wave", Value: "canary"},
		},
	}
	tags := map[string][]string{"wave": {"canary"}}

	if Evaluate(expr, tags) {
		t.Error("expected NOT to reject when tag present")
	}
}

func TestEvaluateNested(t *testing.T) {
	// (env:production OR env:staging) AND NOT wave:canary
	expr := &Expression{
		Op: "AND",
		Conditions: []*Expression{
			{
				Op: "OR",
				Conditions: []*Expression{
					{Tag: "env", Value: "production"},
					{Tag: "env", Value: "staging"},
				},
			},
			{
				Op: "NOT",
				Conditions: []*Expression{
					{Tag: "wave", Value: "canary"},
				},
			},
		},
	}

	tests := []struct {
		name string
		tags map[string][]string
		want bool
	}{
		{
			name: "production without canary",
			tags: map[string][]string{"env": {"production"}},
			want: true,
		},
		{
			name: "staging without canary",
			tags: map[string][]string{"env": {"staging"}},
			want: true,
		},
		{
			name: "production with canary",
			tags: map[string][]string{"env": {"production"}, "wave": {"canary"}},
			want: false,
		},
		{
			name: "dev without canary",
			tags: map[string][]string{"env": {"dev"}},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Evaluate(expr, tt.tags)
			if got != tt.want {
				t.Errorf("Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvaluateNilExpression(t *testing.T) {
	tags := map[string][]string{"env": {"production"}}

	if !Evaluate(nil, tags) {
		t.Error("nil expression should match everything")
	}
}

func TestEvaluateEmptyEndpointTags(t *testing.T) {
	expr := &Expression{Tag: "env", Value: "production"}

	if Evaluate(expr, nil) {
		t.Error("expected no match with nil tags")
	}

	if Evaluate(expr, map[string][]string{}) {
		t.Error("expected no match with empty tags")
	}
}

func TestParseExpressionJSON(t *testing.T) {
	input := &Expression{
		Op: "AND",
		Conditions: []*Expression{
			{Tag: "env", Value: "production"},
			{Tag: "os", Value: "linux"},
		},
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	got, err := ParseExpression(data)
	if err != nil {
		t.Fatalf("ParseExpression: %v", err)
	}

	if got.Op != "AND" {
		t.Errorf("Op = %q, want AND", got.Op)
	}
	if len(got.Conditions) != 2 {
		t.Fatalf("Conditions len = %d, want 2", len(got.Conditions))
	}
	if got.Conditions[0].Tag != "env" || got.Conditions[0].Value != "production" {
		t.Errorf("first condition = %+v, want env:production", got.Conditions[0])
	}
	if got.Conditions[1].Tag != "os" || got.Conditions[1].Value != "linux" {
		t.Errorf("second condition = %+v, want os:linux", got.Conditions[1])
	}
}

func TestParseExpressionEmpty(t *testing.T) {
	got, err := ParseExpression(nil)
	if err != nil {
		t.Fatalf("unexpected error for nil: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for nil input, got %+v", got)
	}

	got, err = ParseExpression([]byte{})
	if err != nil {
		t.Fatalf("unexpected error for empty: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for empty input, got %+v", got)
	}
}

func TestEvaluateMultipleValuesForKey(t *testing.T) {
	expr := &Expression{Tag: "role", Value: "webserver"}
	tags := map[string][]string{
		"role": {"database", "webserver", "cache"},
	}

	if !Evaluate(expr, tags) {
		t.Error("expected match when value is among multiple values for key")
	}
}
