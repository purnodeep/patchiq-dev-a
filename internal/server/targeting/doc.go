// Package targeting is the single abstraction for selecting endpoints by
// key=value tag selectors. Engines (policy evaluation, deployment dispatch,
// workflow filters, compliance scoping) must depend on Resolver and never
// construct endpoint-selection SQL directly.
//
// A Selector is a JSONB-serializable AST with a small fixed vocabulary of
// operators (eq, in, exists, and, or, not). Validate rejects malformed or
// pathological trees; Optimize normalizes them; Compile turns them into a
// parameterized SQL fragment suitable for dropping into a WHERE clause.
//
// See docs/plans/tags-replace-groups.md for the design rationale and the
// decision to replace the legacy groups feature with this package.
package targeting
