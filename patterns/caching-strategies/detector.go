package caching_strategies

import (
	"go/ast"

	"github.com/alpardfm/cost-efficient-go/types"
)

// detector implements types.Detector for caching strategy anti-patterns.
// It detects code that performs repeated expensive operations without caching,
// such as redundant database queries or computations that could be memoized.
type detector struct {
	rule types.Rule
}

// NewDetector creates a new caching strategies detector.
func NewDetector() types.Detector {
	return &detector{
		rule: types.Rule{
			ID:          "CEG-002",
			Name:        "Caching Strategies",
			Description: "Detects repeated expensive computations or database queries that could benefit from in-memory caching to reduce CPU usage and latency.",
			Severity:    types.Major,
			Category:    types.Memory,
			Suggestion:  "Introduce an in-memory cache (e.g., sync.Map or mutex-protected map with TTL) to avoid redundant expensive operations. Use cache-aside pattern: check cache first, fallback to source on miss.",
			ReferenceLinks: []string{
				"https://github.com/alpardfm/cost-efficient-go/blob/main/patterns/caching-strategies/README.md",
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
// This detector looks for patterns indicating repeated expensive operations
// without caching, such as function calls inside loops that could be memoized.
func (d *detector) Detect(ctx types.ASTContext) []types.Finding {
	if ctx.Node == nil {
		return []types.Finding{}
	}

	var findings []types.Finding

	// Look for function calls inside loops that may indicate repeated expensive operations
	// without caching (e.g., database queries, HTTP calls in a loop body).
	ast.Inspect(ctx.Node, func(n ast.Node) bool {
		rangeStmt, ok := n.(*ast.RangeStmt)
		if !ok {
			return true
		}

		// Check if the loop body contains function calls that look expensive
		ast.Inspect(rangeStmt.Body, func(inner ast.Node) bool {
			callExpr, ok := inner.(*ast.CallExpr)
			if !ok {
				return true
			}

			// Check for selector expressions that suggest expensive operations
			if sel, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
				methodName := sel.Sel.Name
				if isExpensiveMethod(methodName) {
					findings = append(findings, types.Finding{
						RuleID:       d.rule.ID,
						FilePath:     ctx.FilePath,
						Line:         ctx.Line,
						Explanation:  "Expensive operation '" + methodName + "' called inside a loop without caching. Each iteration incurs redundant cost that could be eliminated with memoization.",
						SuggestedFix: "Cache the result of '" + methodName + "' before the loop or use a cache-aside pattern to avoid repeated expensive calls.",
						Severity:     d.rule.Severity,
						Category:     d.rule.Category,
						CodeContext:  ctx.CodeContext,
					})
				}
			}

			return true
		})

		return true
	})

	return findings
}

// isExpensiveMethod checks if a method name suggests an expensive operation
// that would benefit from caching.
func isExpensiveMethod(name string) bool {
	expensiveMethods := map[string]bool{
		"Query":     true,
		"QueryRow":  true,
		"QueryRowx": true,
		"Exec":      true,
		"Get":       true,
		"Select":    true,
		"Find":      true,
		"FindOne":   true,
		"Fetch":     true,
		"Do":        true,
		"Request":   true,
		"RoundTrip": true,
	}
	return expensiveMethods[name]
}

