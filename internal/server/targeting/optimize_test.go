package targeting

import (
	"encoding/json"
	"testing"
)

func TestOptimize(t *testing.T) {
	leaf := func(k, v string) Selector { return Selector{Op: OpEq, Key: k, Value: v} }
	notOf := func(s Selector) Selector { return Selector{Op: OpNot, Arg: &s} }

	tests := []struct {
		name string
		in   Selector
		want Selector
	}{
		{
			"single-arg and unwrapped",
			Selector{Op: OpAnd, Args: []Selector{leaf("a", "1")}},
			leaf("a", "1"),
		},
		{
			"single-arg or unwrapped",
			Selector{Op: OpOr, Args: []Selector{leaf("a", "1")}},
			leaf("a", "1"),
		},
		{
			"nested and flattened",
			Selector{Op: OpAnd, Args: []Selector{
				leaf("a", "1"),
				{Op: OpAnd, Args: []Selector{leaf("b", "2"), leaf("c", "3")}},
			}},
			Selector{Op: OpAnd, Args: []Selector{leaf("a", "1"), leaf("b", "2"), leaf("c", "3")}},
		},
		{
			"nested or not flattened into and",
			Selector{Op: OpAnd, Args: []Selector{
				leaf("a", "1"),
				{Op: OpOr, Args: []Selector{leaf("b", "2"), leaf("c", "3")}},
			}},
			Selector{Op: OpAnd, Args: []Selector{
				leaf("a", "1"),
				{Op: OpOr, Args: []Selector{leaf("b", "2"), leaf("c", "3")}},
			}},
		},
		{
			"double not collapsed",
			notOf(notOf(leaf("a", "1"))),
			leaf("a", "1"),
		},
		{
			"triple not left as single not",
			notOf(notOf(notOf(leaf("a", "1")))),
			notOf(leaf("a", "1")),
		},
		{
			"leaf unchanged",
			leaf("a", "1"),
			leaf("a", "1"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Optimize(tc.in)
			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(tc.want)
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("Optimize:\n  got:  %s\n  want: %s", gotJSON, wantJSON)
			}
		})
	}
}

// TestOptimizePreservesValidity pins the invariant that buildQuery relies
// on: a selector that passes Validate must still pass Validate after
// Optimize. If a new rewrite rule ever breaks this, the compiler will
// crash on a post-optimize tree that was not independently validated.
func TestOptimizePreservesValidity(t *testing.T) {
	leaf := func(k, v string) Selector { return Selector{Op: OpEq, Key: k, Value: v} }
	notOf := func(s Selector) Selector { return Selector{Op: OpNot, Arg: &s} }

	valid := []Selector{
		leaf("env", "prod"),
		{Op: OpIn, Key: "os", Values: []string{"ubuntu", "debian"}},
		{Op: OpExists, Key: "owner"},
		{Op: OpAnd, Args: []Selector{leaf("a", "1"), leaf("b", "2")}},
		{Op: OpOr, Args: []Selector{leaf("a", "1"), leaf("b", "2")}},
		notOf(leaf("a", "1")),
		notOf(notOf(leaf("a", "1"))),
		{Op: OpAnd, Args: []Selector{
			leaf("a", "1"),
			{Op: OpAnd, Args: []Selector{leaf("b", "2"), leaf("c", "3")}},
		}},
		{Op: OpAnd, Args: []Selector{leaf("only", "1")}},
	}
	for i, s := range valid {
		if err := Validate(s); err != nil {
			t.Fatalf("case %d: input not valid: %v", i, err)
		}
		opt := Optimize(s)
		if err := Validate(opt); err != nil {
			t.Errorf("case %d: Optimize produced invalid selector: %v\n  in: %+v\n  out: %+v", i, err, s, opt)
		}
	}
}
