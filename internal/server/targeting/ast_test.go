package targeting

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSelectorUnmarshal(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantOp  Op
		wantErr string
	}{
		{"eq", `{"op":"eq","key":"env","value":"prod"}`, OpEq, ""},
		{"in", `{"op":"in","key":"os","values":["ubuntu","debian"]}`, OpIn, ""},
		{"exists", `{"op":"exists","key":"owner"}`, OpExists, ""},
		{"and", `{"op":"and","args":[{"op":"eq","key":"a","value":"1"}]}`, OpAnd, ""},
		{"or", `{"op":"or","args":[{"op":"eq","key":"a","value":"1"}]}`, OpOr, ""},
		{"not", `{"op":"not","arg":{"op":"eq","key":"a","value":"1"}}`, OpNot, ""},
		{"unknown op", `{"op":"bogus","key":"x"}`, "", "unknown op"},
		{"malformed json", `{"op":"eq"`, "", "unexpected end"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var s Selector
			err := json.Unmarshal([]byte(tc.input), &s)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("want error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if s.Op != tc.wantOp {
				t.Errorf("Op = %q, want %q", s.Op, tc.wantOp)
			}
		})
	}
}

func TestSelectorRoundTrip(t *testing.T) {
	// Build a tree, marshal, unmarshal, ensure structural equivalence via
	// re-marshal + byte equality (safer than deep Equal across pointer fields).
	inner := Selector{Op: OpEq, Key: "env", Value: "prod"}
	s := Selector{
		Op: OpAnd,
		Args: []Selector{
			{Op: OpIn, Key: "os", Values: []string{"ubuntu", "debian"}},
			{Op: OpNot, Arg: &inner},
		},
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var s2 Selector
	if err := json.Unmarshal(data, &s2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	data2, err := json.Marshal(s2)
	if err != nil {
		t.Fatalf("remarshal: %v", err)
	}
	if string(data) != string(data2) {
		t.Errorf("round trip mismatch:\n  first: %s\n  again: %s", data, data2)
	}
}
