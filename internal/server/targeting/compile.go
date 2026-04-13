package targeting

import (
	"fmt"
	"strings"
)

// endpointAlias is the SQL alias the compiler gives the endpoints table.
// Fragments reference <alias>.id inside EXISTS subqueries, so the outer
// query in buildQuery must use the same alias.
const endpointAlias = "e"

// compile translates a validated selector into a SQL fragment suitable for
// use inside a WHERE clause, plus the positional arguments it references.
//
// Unexported by design: the resolver package is the only sanctioned
// consumer, and resolver.buildQuery always Validates before calling
// compile. Exposing this function would create a second ingress path that
// could skip validation, which is a SQL-safety footgun.
//
// The fragment assumes an outer query of the form:
//
//	SELECT ... FROM endpoints e WHERE <fragment>
//
// Tenant isolation is not expressed here — it is enforced by RLS once the
// caller sets app.current_tenant_id inside the transaction.
//
// compile defensively re-runs Validate so that a buggy caller that
// bypassed resolver.buildQuery cannot emit SQL from a malformed AST.
func compile(s Selector) (string, []any, error) {
	if err := Validate(s); err != nil {
		return "", nil, err
	}
	b := &compiler{}
	if err := b.emit(s); err != nil {
		return "", nil, err
	}
	return b.sb.String(), b.args, nil
}

type compiler struct {
	sb   strings.Builder
	args []any
}

func (c *compiler) placeholder(v any) string {
	c.args = append(c.args, v)
	return fmt.Sprintf("$%d", len(c.args))
}

// emit walks the AST and writes SQL into c.sb. Each leaf becomes an EXISTS
// subquery against endpoint_tags joined with tags; composites wrap those in
// parens with AND/OR/NOT.
func (c *compiler) emit(s Selector) error {
	switch s.Op {
	case OpEq:
		keyP := c.placeholder(s.Key)
		valP := c.placeholder(s.Value)
		fmt.Fprintf(&c.sb,
			"EXISTS (SELECT 1 FROM endpoint_tags et JOIN tags t ON t.id = et.tag_id "+
				"WHERE et.endpoint_id = %s.id AND lower(t.key) = lower(%s) AND lower(t.value) = lower(%s))",
			endpointAlias, keyP, valP,
		)
		return nil

	case OpIn:
		keyP := c.placeholder(s.Key)
		lowered := make([]string, len(s.Values))
		for i, v := range s.Values {
			lowered[i] = strings.ToLower(v)
		}
		valsP := c.placeholder(lowered)
		fmt.Fprintf(&c.sb,
			"EXISTS (SELECT 1 FROM endpoint_tags et JOIN tags t ON t.id = et.tag_id "+
				"WHERE et.endpoint_id = %s.id AND lower(t.key) = lower(%s) AND lower(t.value) = ANY(%s))",
			endpointAlias, keyP, valsP,
		)
		return nil

	case OpExists:
		keyP := c.placeholder(s.Key)
		fmt.Fprintf(&c.sb,
			"EXISTS (SELECT 1 FROM endpoint_tags et JOIN tags t ON t.id = et.tag_id "+
				"WHERE et.endpoint_id = %s.id AND lower(t.key) = lower(%s))",
			endpointAlias, keyP,
		)
		return nil

	case OpAnd, OpOr:
		if len(s.Args) == 0 {
			// Unreachable in production: buildQuery Validates before
			// compile, and Optimize preserves validity. Reaching here
			// means a caller skipped Validate — wrap ErrMalformedSelector
			// so handlers still map it to 400.
			return fmt.Errorf("%w: compile op=%s with zero args (Validate invariant broken)", ErrMalformedSelector, s.Op)
		}
		sep := " AND "
		if s.Op == OpOr {
			sep = " OR "
		}
		c.sb.WriteByte('(')
		for i, a := range s.Args {
			if i > 0 {
				c.sb.WriteString(sep)
			}
			if err := c.emit(a); err != nil {
				return err
			}
		}
		c.sb.WriteByte(')')
		return nil

	case OpNot:
		if s.Arg == nil {
			return fmt.Errorf("%w: compile op=not with nil arg (Validate invariant broken)", ErrMalformedSelector)
		}
		c.sb.WriteString("(NOT ")
		if err := c.emit(*s.Arg); err != nil {
			return err
		}
		c.sb.WriteByte(')')
		return nil
	}
	return fmt.Errorf("%w: compile unknown op %q", ErrMalformedSelector, s.Op)
}
