// Package tags provides a boolean expression evaluator for endpoint tag matching.
// Expressions support AND, OR, and NOT operators over key-value tag pairs.
package tags

import (
	"encoding/json"
	"fmt"
	"slices"
)

// Expression represents a boolean expression over endpoint tags.
// A leaf expression has Tag and Value set. A compound expression has Op and Conditions.
type Expression struct {
	Op         string        `json:"op,omitempty"`         // "AND", "OR", "NOT"
	Tag        string        `json:"tag,omitempty"`        // leaf: tag key
	Value      string        `json:"value,omitempty"`      // leaf: tag value
	Conditions []*Expression `json:"conditions,omitempty"` // compound: sub-expressions
}

// ParseExpression parses a JSON-encoded tag expression.
// Returns (nil, nil) for nil or empty input.
func ParseExpression(data []byte) (*Expression, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var expr Expression
	if err := json.Unmarshal(data, &expr); err != nil {
		return nil, fmt.Errorf("parse tag expression: %w", err)
	}
	return &expr, nil
}

// Evaluate checks whether an endpoint's tags satisfy the given expression.
// A nil expression matches everything (used for "remaining" waves).
func Evaluate(expr *Expression, endpointTags map[string][]string) bool {
	if expr == nil {
		return true
	}

	// Compound expression with operator.
	switch expr.Op {
	case "AND":
		for _, cond := range expr.Conditions {
			if !Evaluate(cond, endpointTags) {
				return false
			}
		}
		return true
	case "OR":
		for _, cond := range expr.Conditions {
			if Evaluate(cond, endpointTags) {
				return true
			}
		}
		return false
	case "NOT":
		if len(expr.Conditions) == 0 {
			return true
		}
		return !Evaluate(expr.Conditions[0], endpointTags)
	}

	// Leaf expression: check tag key has matching value.
	values, ok := endpointTags[expr.Tag]
	if !ok {
		return false
	}
	return slices.Contains(values, expr.Value)
}
