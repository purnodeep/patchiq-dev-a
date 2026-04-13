/**
 * Tag-based target selector types — the TypeScript mirror of
 * internal/server/targeting/selector.go. Keep these in sync with the
 * Go AST: any new operator must land in both places.
 */

export interface TagEq {
  eq: { key: string; value: string };
}

export interface TagIn {
  in: { key: string; values: string[] };
}

export interface TagExists {
  exists: { key: string };
}

export interface TagAnd {
  and: Selector[];
}

export interface TagOr {
  or: Selector[];
}

export interface TagNot {
  not: Selector;
}

export type Selector = TagEq | TagIn | TagExists | TagAnd | TagOr | TagNot;

/**
 * UI predicate row — simpler than the full AST, sufficient for the common
 * case (list of "key op value(s)" clauses joined by AND). The builder
 * converts rows ↔ AST at save/load time.
 */
export interface PredicateRow {
  id: string;
  key: string;
  op: 'eq' | 'in' | 'exists';
  value: string;
  values: string[];
}

/**
 * Convert an AND-of-predicates row list into a canonical Selector AST.
 * Empty list → null (caller decides whether that means "no change" or
 * "clear selector"). Single row collapses directly without an `and`
 * wrapper so the saved JSON stays readable.
 */
export function rowsToSelector(rows: PredicateRow[]): Selector | null {
  const nodes = rows.map(rowToNode).filter((n): n is Selector => n !== null);
  if (nodes.length === 0) return null;
  if (nodes.length === 1) return nodes[0];
  return { and: nodes };
}

function rowToNode(row: PredicateRow): Selector | null {
  if (!row.key.trim()) return null;
  switch (row.op) {
    case 'eq':
      if (!row.value.trim()) return null;
      return { eq: { key: row.key, value: row.value } };
    case 'in': {
      const vs = row.values.map((v) => v.trim()).filter(Boolean);
      if (vs.length === 0) return null;
      return { in: { key: row.key, values: vs } };
    }
    case 'exists':
      return { exists: { key: row.key } };
  }
}

/**
 * Best-effort reverse: decompose a stored Selector back into predicate
 * rows for the builder UI. Anything more complex than "top-level AND of
 * simple predicates" falls back to a single opaque row so the user can
 * still see the raw JSON in a later follow-up viewer.
 */
export function selectorToRows(sel: Selector | null | undefined): PredicateRow[] {
  if (!sel) return [];
  const nodes: Selector[] = 'and' in sel ? sel.and : [sel];
  const rows: PredicateRow[] = [];
  for (const node of nodes) {
    const row = nodeToRow(node);
    if (row) rows.push({ ...row, id: crypto.randomUUID() });
  }
  return rows;
}

function nodeToRow(node: Selector): Omit<PredicateRow, 'id'> | null {
  if ('eq' in node) {
    return { key: node.eq.key, op: 'eq', value: node.eq.value, values: [] };
  }
  if ('in' in node) {
    return { key: node.in.key, op: 'in', value: '', values: [...node.in.values] };
  }
  if ('exists' in node) {
    return { key: node.exists.key, op: 'exists', value: '', values: [] };
  }
  return null;
}
