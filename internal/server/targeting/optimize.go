package targeting

// Optimize rewrites the AST into a canonical, slightly smaller form without
// changing its semantics. Called before Compile so the generated SQL reflects
// the simplified tree.
//
// Rules:
//  1. Flatten nested same-op composites: and(a, and(b, c)) -> and(a, b, c).
//  2. Unwrap single-arg and/or to their child: and(x) -> x.
//  3. Collapse double negation: not(not(x)) -> x.
//
// Leaf selectors (eq/in/exists) are returned unchanged.
func Optimize(s Selector) Selector {
	switch s.Op {
	case OpAnd, OpOr:
		flat := make([]Selector, 0, len(s.Args))
		for _, a := range s.Args {
			opt := Optimize(a)
			if opt.Op == s.Op {
				flat = append(flat, opt.Args...)
			} else {
				flat = append(flat, opt)
			}
		}
		if len(flat) == 1 {
			return flat[0]
		}
		return Selector{Op: s.Op, Args: flat}

	case OpNot:
		if s.Arg == nil {
			return s
		}
		inner := Optimize(*s.Arg)
		if inner.Op == OpNot && inner.Arg != nil {
			return *inner.Arg
		}
		return Selector{Op: OpNot, Arg: &inner}

	default:
		return s
	}
}
