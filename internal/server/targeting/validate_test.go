package targeting

import (
	"errors"
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	// Build a MaxDepth+2 deep chain to trigger the depth limit.
	deep := Selector{Op: OpEq, Key: "k", Value: "v"}
	for i := 0; i < MaxDepth+1; i++ {
		d := deep
		deep = Selector{Op: OpNot, Arg: &d}
	}
	// Build a tree that is exactly at MaxDepth so a future off-by-one in
	// `depth > MaxDepth` (e.g. changed to >=) is caught — the positive
	// boundary case the upstream review flagged as missing.
	atLimit := Selector{Op: OpEq, Key: "k", Value: "v"}
	for i := 0; i < MaxDepth; i++ {
		d := atLimit
		atLimit = Selector{Op: OpNot, Arg: &d}
	}

	tests := []struct {
		name    string
		in      Selector
		wantErr string
	}{
		{"eq ok", Selector{Op: OpEq, Key: "env", Value: "prod"}, ""},
		{"eq missing key", Selector{Op: OpEq, Value: "prod"}, "invalid key"},
		{"eq missing value", Selector{Op: OpEq, Key: "env"}, "invalid value"},
		{"eq with stray values", Selector{Op: OpEq, Key: "env", Value: "prod", Values: []string{"x"}}, "must set only"},
		{"in ok", Selector{Op: OpIn, Key: "os", Values: []string{"ubuntu"}}, ""},
		{"in empty values", Selector{Op: OpIn, Key: "os"}, "non-empty values"},
		{"in bad value", Selector{Op: OpIn, Key: "os", Values: []string{""}}, "invalid value"},
		{"exists ok", Selector{Op: OpExists, Key: "owner"}, ""},
		{"exists with value", Selector{Op: OpExists, Key: "owner", Value: "x"}, "must set only key"},
		{"and ok", Selector{Op: OpAnd, Args: []Selector{{Op: OpEq, Key: "a", Value: "1"}}}, ""},
		{"and empty", Selector{Op: OpAnd}, "at least one arg"},
		{"or empty", Selector{Op: OpOr}, "at least one arg"},
		{
			"not ok",
			Selector{Op: OpNot, Arg: &Selector{Op: OpEq, Key: "a", Value: "1"}},
			"",
		},
		{"not nil arg", Selector{Op: OpNot}, "requires arg"},
		{"empty op", Selector{}, "empty"},
		{"too deep", deep, "depth exceeds"},
		{"at max depth ok", atLimit, ""},
		{
			"unsupported schema version",
			Selector{V: 99, Op: OpEq, Key: "k", Value: "v"},
			"unsupported schema version",
		},
		{"current schema version ok", Selector{V: SchemaVersion, Op: OpEq, Key: "k", Value: "v"}, ""},
		{
			"key too long",
			Selector{Op: OpEq, Key: strings.Repeat("k", MaxKeyLen+1), Value: "v"},
			"invalid key",
		},
		{
			"value non-printable",
			Selector{Op: OpEq, Key: "k", Value: "ab\x01cd"},
			"invalid value",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(tc.in)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("want nil, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("want error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestValidateDepthErrIs(t *testing.T) {
	deep := Selector{Op: OpEq, Key: "k", Value: "v"}
	for i := 0; i < MaxDepth+1; i++ {
		d := deep
		deep = Selector{Op: OpNot, Arg: &d}
	}
	err := Validate(deep)
	if !errors.Is(err, ErrDepthExceeded) {
		t.Errorf("want errors.Is(err, ErrDepthExceeded); got %v", err)
	}
	// The depth sentinel chains into ErrMalformedSelector so downstream
	// handlers can route any validation failure to a single 400 branch.
	if !errors.Is(err, ErrMalformedSelector) {
		t.Errorf("want errors.Is(err, ErrMalformedSelector); got %v", err)
	}
}

// TestValidate_MalformedSelectorSentinel pins that every structural
// rejection path — op-shape violation, charset, depth, version, empty —
// chains into ErrMalformedSelector. Phase 2 HTTP handlers rely on this
// to distinguish client input errors (400) from infra failures (500)
// without string-matching on error messages.
func TestValidate_MalformedSelectorSentinel(t *testing.T) {
	cases := []struct {
		name string
		in   Selector
	}{
		{"empty", Selector{}},
		{"eq missing key", Selector{Op: OpEq, Value: "v"}},
		{"eq stray values", Selector{Op: OpEq, Key: "k", Value: "v", Values: []string{"x"}}},
		{"in empty values", Selector{Op: OpIn, Key: "os"}},
		{"in bad value", Selector{Op: OpIn, Key: "os", Values: []string{""}}},
		{"exists stray", Selector{Op: OpExists, Key: "k", Value: "v"}},
		{"and empty args", Selector{Op: OpAnd}},
		{"or stray key", Selector{Op: OpOr, Key: "k", Args: []Selector{{Op: OpEq, Key: "a", Value: "1"}}}},
		{"not nil arg", Selector{Op: OpNot}},
		{"unsupported version", Selector{V: 42, Op: OpEq, Key: "k", Value: "v"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(tc.in)
			if err == nil {
				t.Fatal("want error, got nil")
			}
			if !errors.Is(err, ErrMalformedSelector) {
				t.Errorf("want errors.Is(err, ErrMalformedSelector); got %v", err)
			}
		})
	}
}
