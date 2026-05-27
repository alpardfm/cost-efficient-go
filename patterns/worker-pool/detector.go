// Package worker_pool detects unbounded goroutine spawning patterns that should
// use a fixed worker pool to control concurrency and reduce resource consumption.
package worker_pool

import (
	"go/ast"

	"github.com/alpardfm/cost-efficient-go/types"
)


// detector implements types.Detector for worker pool anti-patterns.
// It is stateless and safe for concurrent use.
type detector struct {
	rule types.Rule
}

// NewDetector creates a new worker pool detector.
func NewDetector() types.Detector {
	return &detector{
		rule: types.Rule{
			ID:          "CEG-020",
			Name:        "Worker Pool vs Unbounded Goroutines",
			Description: "Detects unbounded goroutine spawning inside loops where a fixed worker pool should be used. Spawning one goroutine per task causes memory explosion (each goroutine uses 2-8KB stack), CPU thrashing from context switching, and upstream overload from too many concurrent connections.",
			Severity:    types.Major,
			Category:    types.Concurrency,
			Suggestion:  "Use a fixed worker pool or errgroup.SetLimit() to bound concurrency. This controls memory usage, reduces context switching, and prevents upstream overload.",
			ReferenceLinks: []string{
				"https://github.com/alpardfm/cost-efficient-go/tree/main/patterns/worker-pool",
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

	// Look for go statements inside range/for loops that indicate
	// unbounded goroutine spawning.
	rangeStmt, ok := ctx.Node.(*ast.RangeStmt)
	if !ok {
		return []types.Finding{}
	}

	// Walk the body of the range statement looking for go statements.
	ast.Inspect(rangeStmt.Body, func(n ast.Node) bool {
		_, ok := n.(*ast.GoStmt)
		if !ok {
			return true
		}

		findings = append(findings, types.Finding{
			RuleID:       d.rule.ID,
			FilePath:     ctx.FilePath,
			Line:         ctx.Line,
			Explanation:  "Unbounded goroutine spawning detected inside a loop. Each iteration spawns a new goroutine, leading to memory explosion and CPU thrashing at scale.",
			SuggestedFix: "Use a fixed worker pool with channel-based job distribution, or use errgroup.SetLimit() to bound concurrency.",
			Severity:     d.rule.Severity,
			Category:     d.rule.Category,
			CodeContext:  ctx.CodeContext,
		})

		return true
	})

	return findings
}
