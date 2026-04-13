package targeting

import (
	"errors"
	"fmt"
	"unicode"
)

var (
	// ErrMalformedSelector is the parent sentinel for every structural
	// problem in a Selector. Every validate/compile error wraps it so HTTP
	// handlers can map to 400 via errors.Is without string-matching, and
	// "real" infra errors (pool, pgx) remain distinguishable.
	ErrMalformedSelector = errors.New("targeting: malformed selector")

	ErrEmpty         = fmt.Errorf("%w: empty", ErrMalformedSelector)
	ErrDepthExceeded = fmt.Errorf("%w: depth exceeds %d", ErrMalformedSelector, MaxDepth)
	ErrInvalidKey    = fmt.Errorf("%w: invalid key", ErrMalformedSelector)
	ErrInvalidValue  = fmt.Errorf("%w: invalid value", ErrMalformedSelector)
)

// Validate checks that the selector is well-formed. It enforces op-specific
// field rules, depth limit, and key/value charset. A validated selector is
// safe to pass to Compile.
func Validate(s Selector) error {
	return validate(s, 0)
}

func validate(s Selector, depth int) error {
	if depth > MaxDepth {
		return ErrDepthExceeded
	}
	// Guard programmatic construction paths too: a caller that builds a
	// Selector literal (not via JSON) must still pass through the same
	// version gate as the unmarshal path.
	if depth == 0 && s.V != 0 && s.V != SchemaVersion {
		return fmt.Errorf("%w: unsupported schema version %d (want %d)", ErrMalformedSelector, s.V, SchemaVersion)
	}
	switch s.Op {
	case OpEq:
		if err := checkKey(s.Key); err != nil {
			return err
		}
		if err := checkValue(s.Value); err != nil {
			return err
		}
		if len(s.Values) != 0 || len(s.Args) != 0 || s.Arg != nil {
			return fmt.Errorf("%w: op=eq must set only key and value", ErrMalformedSelector)
		}
		return nil

	case OpIn:
		if err := checkKey(s.Key); err != nil {
			return err
		}
		if len(s.Values) == 0 {
			return fmt.Errorf("%w: op=in requires non-empty values", ErrMalformedSelector)
		}
		for i, v := range s.Values {
			if err := checkValue(v); err != nil {
				return fmt.Errorf("targeting: op=in values[%d]: %w", i, err)
			}
		}
		if s.Value != "" || len(s.Args) != 0 || s.Arg != nil {
			return fmt.Errorf("%w: op=in must set only key and values", ErrMalformedSelector)
		}
		return nil

	case OpExists:
		if err := checkKey(s.Key); err != nil {
			return err
		}
		if s.Value != "" || len(s.Values) != 0 || len(s.Args) != 0 || s.Arg != nil {
			return fmt.Errorf("%w: op=exists must set only key", ErrMalformedSelector)
		}
		return nil

	case OpAnd, OpOr:
		if len(s.Args) == 0 {
			return fmt.Errorf("%w: op=%s requires at least one arg", ErrMalformedSelector, s.Op)
		}
		if s.Key != "" || s.Value != "" || len(s.Values) != 0 || s.Arg != nil {
			return fmt.Errorf("%w: op=%s must set only args", ErrMalformedSelector, s.Op)
		}
		for i, a := range s.Args {
			if err := validate(a, depth+1); err != nil {
				return fmt.Errorf("targeting: op=%s args[%d]: %w", s.Op, i, err)
			}
		}
		return nil

	case OpNot:
		if s.Arg == nil {
			return fmt.Errorf("%w: op=not requires arg", ErrMalformedSelector)
		}
		if s.Key != "" || s.Value != "" || len(s.Values) != 0 || len(s.Args) != 0 {
			return fmt.Errorf("%w: op=not must set only arg", ErrMalformedSelector)
		}
		return validate(*s.Arg, depth+1)

	case "":
		return ErrEmpty
	}
	return fmt.Errorf("%w: unknown op %q", ErrMalformedSelector, s.Op)
}

func checkKey(k string) error {
	if k == "" {
		return fmt.Errorf("%w: empty", ErrInvalidKey)
	}
	if len(k) > MaxKeyLen {
		return fmt.Errorf("%w: length %d > %d", ErrInvalidKey, len(k), MaxKeyLen)
	}
	for _, r := range k {
		if !isPrintableASCII(r) {
			return fmt.Errorf("%w: non-printable character", ErrInvalidKey)
		}
	}
	return nil
}

func checkValue(v string) error {
	if v == "" {
		return fmt.Errorf("%w: empty", ErrInvalidValue)
	}
	if len(v) > MaxValueLen {
		return fmt.Errorf("%w: length %d > %d", ErrInvalidValue, len(v), MaxValueLen)
	}
	for _, r := range v {
		if !isPrintableASCII(r) {
			return fmt.Errorf("%w: non-printable character", ErrInvalidValue)
		}
	}
	return nil
}

func isPrintableASCII(r rune) bool {
	return r >= 0x20 && r <= 0x7E && unicode.IsPrint(r)
}
