package targeting

import (
	"strings"
	"testing"
)

func TestCompile(t *testing.T) {
	leaf := func(k, v string) Selector { return Selector{Op: OpEq, Key: k, Value: v} }

	tests := []struct {
		name         string
		in           Selector
		wantContains []string
		wantArgs     []any
	}{
		{
			"eq",
			leaf("env", "prod"),
			[]string{"EXISTS", "endpoint_tags", "e.id", "lower(t.key) = lower($1)", "lower(t.value) = lower($2)"},
			[]any{"env", "prod"},
		},
		{
			"in",
			Selector{Op: OpIn, Key: "os", Values: []string{"Ubuntu", "Debian"}},
			[]string{"lower(t.key) = lower($1)", "lower(t.value) = ANY($2)"},
			[]any{"os", []string{"ubuntu", "debian"}}, // values lowercased at compile time
		},
		{
			"exists",
			Selector{Op: OpExists, Key: "owner"},
			[]string{"lower(t.key) = lower($1)"},
			[]any{"owner"},
		},
		{
			"and composite",
			Selector{Op: OpAnd, Args: []Selector{leaf("a", "1"), leaf("b", "2")}},
			[]string{"(EXISTS", " AND EXISTS", "$1", "$2", "$3", "$4"},
			[]any{"a", "1", "b", "2"},
		},
		{
			"or composite",
			Selector{Op: OpOr, Args: []Selector{leaf("a", "1"), leaf("b", "2")}},
			[]string{" OR "},
			[]any{"a", "1", "b", "2"},
		},
		{
			"not composite",
			Selector{Op: OpNot, Arg: &Selector{Op: OpEq, Key: "a", Value: "1"}},
			[]string{"(NOT EXISTS"},
			[]any{"a", "1"},
		},
		{
			"nested and/or arg numbering stays consistent",
			Selector{Op: OpAnd, Args: []Selector{
				leaf("a", "1"),
				{Op: OpOr, Args: []Selector{leaf("b", "2"), leaf("c", "3")}},
			}},
			[]string{"$1", "$2", "$3", "$4", "$5", "$6"},
			[]any{"a", "1", "b", "2", "c", "3"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sql, args, err := compile(tc.in)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			// Safety rail: the SQL fragment must never contain raw user
			// input. All keys/values are positional placeholders.
			if tc.in.Op == OpEq {
				if strings.Contains(sql, tc.in.Key) || strings.Contains(sql, tc.in.Value) {
					t.Errorf("sql contains raw user input, expected placeholders only:\n  %s", sql)
				}
			}
			for _, frag := range tc.wantContains {
				if !strings.Contains(sql, frag) {
					t.Errorf("sql missing %q:\n  got: %s", frag, sql)
				}
			}
			if !argsEqual(args, tc.wantArgs) {
				t.Errorf("args mismatch:\n  got:  %#v\n  want: %#v", args, tc.wantArgs)
			}
		})
	}
}

func TestCompileEmptyCompositeErrors(t *testing.T) {
	// compile() now re-runs Validate defensively, so this produces the
	// Validate error ("at least one arg") rather than the emit-level error.
	_, _, err := compile(Selector{Op: OpAnd})
	if err == nil {
		t.Fatal("want error for empty and; got nil")
	}
}

// TestCompileRevalidatesMalformedInput pins the safety property that
// compile() is protected against direct callers that bypass buildQuery.
func TestCompileRevalidatesMalformedInput(t *testing.T) {
	// Selector{Op:""} would be caught by Validate as ErrEmpty.
	_, _, err := compile(Selector{})
	if err == nil {
		t.Fatal("want error for empty selector; got nil")
	}
}

// argsEqual compares two []any slices element-wise, handling []string
// embedded as one element (for the OpIn case).
func argsEqual(got, want []any) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		switch wv := want[i].(type) {
		case []string:
			gv, ok := got[i].([]string)
			if !ok || len(gv) != len(wv) {
				return false
			}
			for j := range gv {
				if gv[j] != wv[j] {
					return false
				}
			}
		default:
			if got[i] != want[i] {
				return false
			}
		}
	}
	return true
}
