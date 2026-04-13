package workflow

import (
	"fmt"
	"maps"
	"strings"
)

// ValidationError aggregates all workflow validation violations.
type ValidationError struct {
	Violations []Violation
}

func (e *ValidationError) Error() string {
	msgs := make([]string, len(e.Violations))
	for i, v := range e.Violations {
		msgs[i] = v.Message
	}
	return fmt.Sprintf("workflow validation failed: %s", strings.Join(msgs, "; "))
}

// Violation represents a single validation rule violation.
type Violation struct {
	Code    string   `json:"code"`
	Message string   `json:"message"`
	NodeIDs []string `json:"node_ids,omitempty"`
}

// ValidateWorkflow checks a set of nodes and edges for structural and config validity.
// Returns a *ValidationError containing all violations, or nil if valid.
// Collects all violations rather than failing on the first.
func ValidateWorkflow(nodes []Node, edges []Edge) error {
	var violations []Violation

	// Index nodes by ID, checking for duplicates.
	nodeByID := make(map[string]Node, len(nodes))
	for _, n := range nodes {
		if _, dup := nodeByID[n.ID]; dup {
			violations = append(violations, Violation{
				Code:    "DUPLICATE_NODE_ID",
				Message: fmt.Sprintf("duplicate node ID %q", n.ID),
				NodeIDs: []string{n.ID},
			})
		}
		nodeByID[n.ID] = n
	}

	// Validate each node's config.
	for _, n := range nodes {
		if err := ValidateNodeConfig(n.NodeType, n.Config); err != nil {
			violations = append(violations, Violation{
				Code:    "INVALID_NODE_CONFIG",
				Message: fmt.Sprintf("node %q (%s): %s", n.ID, n.NodeType, err),
				NodeIDs: []string{n.ID},
			})
		}
	}

	// Collect trigger and complete node IDs.
	var triggerIDs, completeIDs []string
	for _, n := range nodes {
		switch n.NodeType {
		case NodeTrigger:
			triggerIDs = append(triggerIDs, n.ID)
		case NodeComplete:
			completeIDs = append(completeIDs, n.ID)
		}
	}

	if len(triggerIDs) == 0 {
		violations = append(violations, Violation{
			Code:    "MISSING_TRIGGER",
			Message: "workflow must have exactly one trigger node",
		})
	} else if len(triggerIDs) > 1 {
		violations = append(violations, Violation{
			Code:    "MULTIPLE_TRIGGERS",
			Message: fmt.Sprintf("workflow must have exactly one trigger node, found %d", len(triggerIDs)),
			NodeIDs: triggerIDs,
		})
	}

	if len(completeIDs) == 0 {
		violations = append(violations, Violation{
			Code:    "MISSING_COMPLETE",
			Message: "workflow must have exactly one complete node",
		})
	} else if len(completeIDs) > 1 {
		violations = append(violations, Violation{
			Code:    "MULTIPLE_COMPLETES",
			Message: fmt.Sprintf("workflow must have exactly one complete node, found %d", len(completeIDs)),
			NodeIDs: completeIDs,
		})
	}

	// Check edges reference existing nodes and detect self-loops.
	for _, e := range edges {
		if e.SourceNodeID == e.TargetNodeID {
			violations = append(violations, Violation{
				Code:    "SELF_LOOP",
				Message: fmt.Sprintf("edge from node %q to itself is not allowed", e.SourceNodeID),
				NodeIDs: []string{e.SourceNodeID},
			})
		}
		if _, ok := nodeByID[e.SourceNodeID]; !ok {
			violations = append(violations, Violation{
				Code:    "DANGLING_EDGE",
				Message: fmt.Sprintf("edge references unknown source node %q", e.SourceNodeID),
			})
		}
		if _, ok := nodeByID[e.TargetNodeID]; !ok {
			violations = append(violations, Violation{
				Code:    "DANGLING_EDGE",
				Message: fmt.Sprintf("edge references unknown target node %q", e.TargetNodeID),
			})
		}
	}

	// Build outgoing adjacency list and in-degree map.
	outgoing := make(map[string][]Edge)
	inDegree := make(map[string]int)
	for _, n := range nodes {
		inDegree[n.ID] = 0
	}
	for _, e := range edges {
		outgoing[e.SourceNodeID] = append(outgoing[e.SourceNodeID], e)
		inDegree[e.TargetNodeID]++
	}

	// Check trigger has no incoming edges
	if len(triggerIDs) == 1 {
		if inDegree[triggerIDs[0]] > 0 {
			violations = append(violations, Violation{
				Code:    "TRIGGER_HAS_INCOMING",
				Message: "trigger node must not have incoming edges",
				NodeIDs: triggerIDs,
			})
		}
	}

	// Check complete has no outgoing edges
	if len(completeIDs) == 1 {
		if len(outgoing[completeIDs[0]]) > 0 {
			violations = append(violations, Violation{
				Code:    "COMPLETE_HAS_OUTGOING",
				Message: "complete node must not have outgoing edges",
				NodeIDs: completeIDs,
			})
		}
	}

	// Check decision nodes have exactly 2 outgoing edges labeled "yes" and "no"
	for _, n := range nodes {
		if n.NodeType != NodeDecision {
			continue
		}
		out := outgoing[n.ID]
		if len(out) != 2 {
			violations = append(violations, Violation{
				Code:    "DECISION_EDGE_COUNT",
				Message: fmt.Sprintf("decision node %q must have exactly 2 outgoing edges, found %d", n.ID, len(out)),
				NodeIDs: []string{n.ID},
			})
			continue
		}
		labels := map[string]bool{}
		for _, e := range out {
			labels[e.Label] = true
		}
		if !labels["yes"] || !labels["no"] {
			violations = append(violations, Violation{
				Code:    "DECISION_EDGE_LABELS",
				Message: fmt.Sprintf("decision node %q outgoing edges must be labeled \"yes\" and \"no\"", n.ID),
				NodeIDs: []string{n.ID},
			})
		}
	}

	// Cycle detection via Kahn's algorithm (topological sort)
	degCopy := make(map[string]int, len(inDegree))
	maps.Copy(degCopy, inDegree)
	var queue []string
	for id, deg := range degCopy {
		if deg == 0 {
			queue = append(queue, id)
		}
	}
	visited := 0
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		visited++
		for _, e := range outgoing[curr] {
			degCopy[e.TargetNodeID]--
			if degCopy[e.TargetNodeID] == 0 {
				queue = append(queue, e.TargetNodeID)
			}
		}
	}
	if visited < len(nodes) {
		var cycleNodeIDs []string
		for id, deg := range degCopy {
			if deg > 0 {
				cycleNodeIDs = append(cycleNodeIDs, id)
			}
		}
		violations = append(violations, Violation{
			Code:    "CYCLE_DETECTED",
			Message: "workflow contains a cycle",
			NodeIDs: cycleNodeIDs,
		})
	}

	// Reachability: all nodes must be reachable from trigger via BFS.
	// Skip if cycle was detected (visited < len(nodes)) since cycle nodes would
	// show as unreachable, producing redundant violations.
	if len(triggerIDs) == 1 && visited == len(nodes) {
		reachable := make(map[string]bool)
		bfsQueue := []string{triggerIDs[0]}
		reachable[triggerIDs[0]] = true
		for len(bfsQueue) > 0 {
			curr := bfsQueue[0]
			bfsQueue = bfsQueue[1:]
			for _, e := range outgoing[curr] {
				if !reachable[e.TargetNodeID] {
					reachable[e.TargetNodeID] = true
					bfsQueue = append(bfsQueue, e.TargetNodeID)
				}
			}
		}
		var unreachable []string
		for _, n := range nodes {
			if !reachable[n.ID] {
				unreachable = append(unreachable, n.ID)
			}
		}
		if len(unreachable) > 0 {
			violations = append(violations, Violation{
				Code:    "UNREACHABLE_NODES",
				Message: fmt.Sprintf("%d node(s) not reachable from trigger", len(unreachable)),
				NodeIDs: unreachable,
			})
		}
	}

	if len(violations) > 0 {
		return &ValidationError{Violations: violations}
	}
	return nil
}
