// Package batch_processing detects individual I/O operations that should be batched
// to reduce network round-trip overhead and improve cost efficiency.
package batch_processing

import (
	"go/ast"

	"github.com/alpardfm/cost-efficient-go/types"
)


// detector implements types.Detector for batch processing anti-patterns.
// It is stateless and safe for concurrent use.
type detector struct {
	rule types.Rule
}

// NewDetector creates a new batch processing detector.
func NewDetector() types.Detector {
	return &detector{
		rule: types.Rule{
			ID:          "CEG-001",
			Name:        "Batch Processing vs Individual Operations",
			Description: "Detects individual I/O operations (INSERT, PUBLISH, HTTP calls) inside loops that should be batched to reduce network round-trip overhead. At scale (10M+ ops/day), individual operations waste network I/O, database connection time, and compute resources.",
			Severity:    types.Major,
			Category:    types.IO,
			Suggestion:  "Collect operations and execute them in batches. Use batch INSERT, pipeline PUBLISH, or bulk HTTP requests to amortize network round-trip cost across multiple records.",
			ReferenceLinks: []string{
				"https://github.com/alpardfm/cost-efficient-go/tree/main/patterns/batch-processing",
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
	// individual I/O operations (e.g., db.Exec, db.Query, Publish, Send).
	var findings []types.Finding

	rangeStmt, ok := ctx.Node.(*ast.RangeStmt)
	if !ok {
		return []types.Finding{}
	}

	// Walk the body of the range statement looking for call expressions
	// that suggest individual I/O operations.
	ast.Inspect(rangeStmt.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		if isIndividualIOCall(call) {
			findings = append(findings, types.Finding{
				RuleID:       d.rule.ID,
				FilePath:     ctx.FilePath,
				Line:         ctx.Line,
				Explanation:  "Individual I/O operation detected inside a loop. Each iteration incurs a network round-trip, leading to O(n) latency instead of O(n/batchSize).",
				SuggestedFix: "Collect items and use a batch operation (e.g., batch INSERT, pipeline PUBLISH) to amortize network overhead.",
				Severity:     d.rule.Severity,
				Category:     d.rule.Category,
				CodeContext:  ctx.CodeContext,
			})
		}

		return true
	})

	return findings
}

// isIndividualIOCall checks if a call expression looks like an individual I/O operation.
func isIndividualIOCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	// Common individual I/O method names
	switch sel.Sel.Name {
	case "Exec", "ExecContext", "Query", "QueryContext", "QueryRow", "QueryRowContext",
		"Publish", "Send", "Do", "Insert", "Put", "Write":
		return true
	}

	return false
}
