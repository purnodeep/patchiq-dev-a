package workflow

import "fmt"

// Graph is an in-memory representation of the workflow DAG for execution.
type Graph struct {
	Nodes         map[string]Node
	Edges         []Edge
	Outgoing      map[string][]Edge
	Incoming      map[string][]Edge
	TriggerNodeID string
}

// BuildGraph constructs the in-memory DAG from nodes and edges.
func BuildGraph(nodes []Node, edges []Edge) (*Graph, error) {
	g := &Graph{
		Nodes:    make(map[string]Node, len(nodes)),
		Edges:    edges,
		Outgoing: make(map[string][]Edge),
		Incoming: make(map[string][]Edge),
	}
	for _, n := range nodes {
		g.Nodes[n.ID] = n
		if n.NodeType == NodeTrigger {
			g.TriggerNodeID = n.ID
		}
	}
	if g.TriggerNodeID == "" {
		return nil, fmt.Errorf("build graph: no trigger node found")
	}
	for _, e := range edges {
		g.Outgoing[e.SourceNodeID] = append(g.Outgoing[e.SourceNodeID], e)
		g.Incoming[e.TargetNodeID] = append(g.Incoming[e.TargetNodeID], e)
	}
	return g, nil
}

// TopologicalOrder returns node IDs in topological order using Kahn's algorithm.
func (g *Graph) TopologicalOrder() []string {
	inDegree := make(map[string]int, len(g.Nodes))
	for id := range g.Nodes {
		inDegree[id] = len(g.Incoming[id])
	}
	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}
	var order []string
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		order = append(order, curr)
		for _, e := range g.Outgoing[curr] {
			inDegree[e.TargetNodeID]--
			if inDegree[e.TargetNodeID] == 0 {
				queue = append(queue, e.TargetNodeID)
			}
		}
	}
	return order
}

// Successors returns the target node IDs of all outgoing edges from nodeID.
func (g *Graph) Successors(nodeID string) []string {
	edges := g.Outgoing[nodeID]
	ids := make([]string, len(edges))
	for i, e := range edges {
		ids[i] = e.TargetNodeID
	}
	return ids
}

// DecisionBranches returns the "yes" and "no" branch target node IDs for a decision node.
func (g *Graph) DecisionBranches(nodeID string) (yesNodeID, noNodeID string) {
	for _, e := range g.Outgoing[nodeID] {
		switch e.Label {
		case "yes":
			yesNodeID = e.TargetNodeID
		case "no":
			noNodeID = e.TargetNodeID
		}
	}
	return
}

// SkippedDescendants returns nodes reachable exclusively from skipRoot but not from keepRoot.
func (g *Graph) SkippedDescendants(decisionID, skipRoot, keepRoot string) []string {
	keepReachable := g.reachableFrom(keepRoot)
	keepReachable[decisionID] = true
	skipReachable := g.reachableFrom(skipRoot)

	var skipped []string
	for id := range skipReachable {
		if !keepReachable[id] {
			skipped = append(skipped, id)
		}
	}
	return skipped
}

func (g *Graph) reachableFrom(startID string) map[string]bool {
	visited := map[string]bool{startID: true}
	queue := []string{startID}
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		for _, e := range g.Outgoing[curr] {
			if !visited[e.TargetNodeID] {
				visited[e.TargetNodeID] = true
				queue = append(queue, e.TargetNodeID)
			}
		}
	}
	return visited
}
