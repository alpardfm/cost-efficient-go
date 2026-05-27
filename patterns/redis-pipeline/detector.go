// Package redis_pipeline detects individual Redis operations that should be pipelined
// to reduce network round-trip overhead and improve cost efficiency.
package redis_pipeline

import (
	"go/ast"

	"github.com/alpardfm/cost-efficient-go/types"
)


// detector implements types.Detector for Redis pipeline anti-patterns.
// It is stateless and safe for concurrent use.
type detector struct {
	rule types.Rule
}

// NewDetector creates a new Redis pipeline detector.
func NewDetector() types.Detector {
	return &detector{
		rule: types.Rule{
			ID:          "CEG-015",
			Name:        "Redis Pipeline vs Individual Operations",
			Description: "Detects individual Redis GET/SET operations inside loops that should be pipelined to reduce network round-trip overhead. At scale (10M+ cache ops/day), individual operations waste network I/O, connection time, and increase p99 latency.",
			Severity:    types.Major,
			Category:    types.IO,
			Suggestion:  "Collect Redis commands and execute them in a single pipeline batch. Use MGET/MSET or pipeline APIs to amortize network round-trip cost across multiple operations.",
			ReferenceLinks: []string{
				"https://github.com/alpardfm/cost-efficient-go/tree/main/patterns/redis-pipeline",
			},
		},
	}
}

// Rule returns the detection rule metadata for this detector.
func (d *detector) Rule() types.Rule {
	return d.rule
}

// Detect analyzes the given AST context and returns findings.
// Returns an empty slice if no issues are detected or if Node is nil.
// Does not panic and does not perform file I/O.
func (d *detector) Detect(ctx types.ASTContext) []types.Finding {
	if ctx.Node == nil {
		return []types.Finding{}
	}

	// Look for function calls inside range/for loops that indicate
	// individual Redis operations (e.g., Get, Set, Do, Cmd).
	var findings []types.Finding

	rangeStmt, ok := ctx.Node.(*ast.RangeStmt)
	if !ok {
		return []types.Finding{}
	}

	// Walk the body of the range statement looking for call expressions
	// that suggest individual Redis operations.
	ast.Inspect(rangeStmt.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		if isIndividualRedisCall(call) {
			findings = append(findings, types.Finding{
				RuleID:       d.rule.ID,
				FilePath:     ctx.FilePath,
				Line:         ctx.Line,
				Explanation:  "Individual Redis operation detected inside a loop. Each iteration incurs a network round-trip, leading to O(n) latency instead of O(1) with pipelining.",
				SuggestedFix: "Collect commands and use a Redis pipeline (e.g., Pipeline(), MGET, MSET) to batch operations into a single round-trip.",
				Severity:     d.rule.Severity,
				Category:     d.rule.Category,
				CodeContext:  ctx.CodeContext,
			})
		}

		return true
	})

	return findings
}

// isIndividualRedisCall checks if a call expression looks like an individual Redis operation.
func isIndividualRedisCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	// Common individual Redis method names
	switch sel.Sel.Name {
	case "Get", "Set", "Del", "Incr", "Decr", "HGet", "HSet",
		"LPush", "RPush", "SAdd", "ZAdd", "Do", "Cmd",
		"SetEX", "SetNX", "GetSet", "Expire":
		return true
	}

	return false
}
