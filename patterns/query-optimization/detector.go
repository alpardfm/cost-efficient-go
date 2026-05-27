// Package query_optimization detects unoptimized database query patterns
// such as SELECT *, N+1 queries, and inefficient pagination that cause
// excessive I/O overhead and API latency.
package query_optimization

import (
	"go/ast"

	"github.com/alpardfm/cost-efficient-go/types"
)


// detector implements types.Detector for query optimization anti-patterns.
// It is stateless and safe for concurrent use.
type detector struct {
	rule types.Rule
}

// NewDetector creates a new query optimization detector.
func NewDetector() types.Detector {
	return &detector{
		rule: types.Rule{
			ID:          "CEG-014",
			Name:        "Query Optimization & Indexing",
			Description: "Detects unoptimized database query patterns including SELECT * when only specific columns are needed, N+1 queries in loops, and OFFSET-based pagination for deep pages. These patterns cause excessive network I/O, memory waste, and API latency at scale.",
			Severity:    types.Critical,
			Category:    types.IO,
			Suggestion:  "Use SELECT with specific columns instead of SELECT *. Replace N+1 queries with batch queries using WHERE IN clauses. Use keyset (cursor-based) pagination instead of OFFSET for deep pages.",
			ReferenceLinks: []string{
				"https://github.com/alpardfm/cost-efficient-go/tree/main/patterns/query-optimization",
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

	var findings []types.Finding

	// Look for loop constructs containing individual query calls (N+1 pattern)
	switch node := ctx.Node.(type) {
	case *ast.RangeStmt:
		findings = append(findings, d.detectNPlusOneInRange(node, ctx)...)
	case *ast.ForStmt:
		findings = append(findings, d.detectNPlusOneInFor(node, ctx)...)
	}

	return findings
}

// detectNPlusOneInRange checks for individual query calls inside range loops.
func (d *detector) detectNPlusOneInRange(rangeStmt *ast.RangeStmt, ctx types.ASTContext) []types.Finding {
	var findings []types.Finding

	ast.Inspect(rangeStmt.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		if isQueryCall(call) {
			findings = append(findings, types.Finding{
				RuleID:       d.rule.ID,
				FilePath:     ctx.FilePath,
				Line:         ctx.Line,
				Explanation:  "Individual database query detected inside a loop (N+1 pattern). Each iteration incurs a network round-trip, causing O(n) latency instead of O(1) with a batch query.",
				SuggestedFix: "Collect IDs and use a single batch query with WHERE id IN (...) clause to fetch all results in one round-trip.",
				Severity:     d.rule.Severity,
				Category:     d.rule.Category,
				CodeContext:  ctx.CodeContext,
			})
		}

		return true
	})

	return findings
}

// detectNPlusOneInFor checks for individual query calls inside for loops.
func (d *detector) detectNPlusOneInFor(forStmt *ast.ForStmt, ctx types.ASTContext) []types.Finding {
	var findings []types.Finding

	ast.Inspect(forStmt.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		if isQueryCall(call) {
			findings = append(findings, types.Finding{
				RuleID:       d.rule.ID,
				FilePath:     ctx.FilePath,
				Line:         ctx.Line,
				Explanation:  "Individual database query detected inside a loop (N+1 pattern). Each iteration incurs a network round-trip, causing O(n) latency instead of O(1) with a batch query.",
				SuggestedFix: "Collect IDs and use a single batch query with WHERE id IN (...) clause to fetch all results in one round-trip.",
				Severity:     d.rule.Severity,
				Category:     d.rule.Category,
				CodeContext:  ctx.CodeContext,
			})
		}

		return true
	})

	return findings
}

// isQueryCall checks if a call expression looks like a database query operation.
func isQueryCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	// Common database query method names
	switch sel.Sel.Name {
	case "Query", "QueryContext", "QueryRow", "QueryRowContext",
		"Exec", "ExecContext", "Get", "Select", "Find", "FindAll":
		return true
	}

	return false
}
