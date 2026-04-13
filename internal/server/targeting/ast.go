package targeting

import (
	"encoding/json"
	"fmt"
)

// Op is a selector operator. The set is intentionally small.
type Op string

const (
	OpEq     Op = "eq"
	OpIn     Op = "in"
	OpExists Op = "exists"
	OpAnd    Op = "and"
	OpOr     Op = "or"
	OpNot    Op = "not"
)

// MaxDepth bounds AST nesting so a pathological expression cannot exhaust
// stack or blow up the compiler. Real selectors rarely exceed 3 or 4 levels.
const MaxDepth = 8

// SchemaVersion is the current Selector JSON envelope version. It is
// stored alongside the AST in policy_tag_selectors.expression so a future
// Phase 3 shape change (new op, nested struct, etc.) can ship without
// silently mis-reading rows persisted by an older binary.
//
// On read, UnmarshalJSON accepts selectors without a "v" field (treated
// as v1 for backward compatibility with rows stored before this field
// landed) and rejects any non-zero version other than SchemaVersion.
const SchemaVersion = 1

// MaxKeyLen and MaxValueLen cap individual string lengths. The DB columns are
// TEXT so there is no hard limit; these guard UI sanity and query size.
const (
	MaxKeyLen   = 128
	MaxValueLen = 128
)

// Selector is the JSON/AST shape of a tag selector expression.
//
// Exactly one shape per Op:
//
//	{"op":"eq",     "key":"env", "value":"prod"}
//	{"op":"in",     "key":"os",  "values":["ubuntu","debian"]}
//	{"op":"exists", "key":"owner"}
//	{"op":"and",    "args":[...]}
//	{"op":"or",     "args":[...]}
//	{"op":"not",    "arg":{...}}
type Selector struct {
	// V is the envelope schema version. Omitted on the wire for v1 to keep
	// existing rows backward compatible; omitempty means marshalled output
	// for a default-constructed Selector stays identical to pre-v1 shape.
	V      int        `json:"v,omitempty"`
	Op     Op         `json:"op"`
	Key    string     `json:"key,omitempty"`
	Value  string     `json:"value,omitempty"`
	Values []string   `json:"values,omitempty"`
	Args   []Selector `json:"args,omitempty"`
	Arg    *Selector  `json:"arg,omitempty"`
}

// UnmarshalJSON decodes a Selector and rejects unknown operators and
// unrecognised schema versions at the parsing boundary so callers don't
// need to re-check Op validity after a successful Unmarshal.
//
// A missing "v" field is treated as SchemaVersion for backward compat
// with rows stored before the envelope field landed; any explicit
// version other than SchemaVersion is a hard error so an older binary
// reading a newer row fails loudly instead of silently mis-executing.
func (s *Selector) UnmarshalJSON(data []byte) error {
	type alias Selector
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return fmt.Errorf("targeting: decode selector: %w", err)
	}
	if a.V != 0 && a.V != SchemaVersion {
		return fmt.Errorf("targeting: unsupported selector schema version %d (want %d)", a.V, SchemaVersion)
	}
	if !isKnownOp(a.Op) {
		return fmt.Errorf("targeting: unknown op %q", a.Op)
	}
	*s = Selector(a)
	return nil
}

func isKnownOp(op Op) bool {
	switch op {
	case OpEq, OpIn, OpExists, OpAnd, OpOr, OpNot:
		return true
	}
	return false
}
