// Package profiling_benchmarking detects missing or incorrect profiling and benchmarking
// practices that lead to inaccurate performance measurements and wasted optimization effort.
package profiling_benchmarking

import (
	"go/ast"

	"github.com/alpardfm/cost-efficient-go/types"
)


// detector implements types.Detector for profiling and benchmarking anti-patterns.
// It is stateless and safe for concurrent use.
type detector struct {
	rule types.Rule
}

// NewDetector creates a new profiling and benchmarking detector.
func NewDetector() types.Detector {
	return &detector{
		rule: types.Rule{
			ID:          "CEG-013",
			Name:        "Profiling & Benchmarking Techniques",
			Description: "Detects common benchmarking pitfalls such as unused results that allow compiler elimination, missing warm-up phases, and lack of allocation reporting. Incorrect benchmarks lead to false optimization conclusions and wasted engineering effort.",
			Severity:    types.Minor,
			Category:    types.Memory,
			Suggestion:  "Assign benchmark results to a package-level sink variable to prevent compiler elimination. Use b.ReportAllocs() to track allocations. Include warm-up iterations before measurement to avoid cold-start bias.",
			ReferenceLinks: []string{
				"https://github.com/alpardfm/cost-efficient-go/tree/main/patterns/profiling-benchmarking",
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

	// Look for benchmark functions that discard call results (compiler elimination risk).
	funcDecl, ok := ctx.Node.(*ast.FuncDecl)
	if !ok {
		return []types.Finding{}
	}

	// Check if this is a benchmark function (starts with "Benchmark" and has *testing.B param)
	if !isBenchmarkFunc(funcDecl) {
		return []types.Finding{}
	}

	// Walk the function body looking for expression statements with call expressions
	// whose results are discarded (not assigned to anything).
	if funcDecl.Body != nil {
		ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
			// Look for for-loop bodies (the b.N loop)
			forStmt, ok := n.(*ast.ForStmt)
			if !ok {
				return true
			}

			// Check statements inside the for loop
			for _, stmt := range forStmt.Body.List {
				exprStmt, ok := stmt.(*ast.ExprStmt)
				if !ok {
					continue
				}
				// A bare function call whose return value is discarded
				if _, ok := exprStmt.X.(*ast.CallExpr); ok {
					findings = append(findings, types.Finding{
						RuleID:       d.rule.ID,
						FilePath:     ctx.FilePath,
						Line:         ctx.Line,
						Explanation:  "Benchmark function call result is discarded. The Go compiler may eliminate the entire computation, producing misleading benchmark results.",
						SuggestedFix: "Assign the result to a package-level sink variable (e.g., `benchSink = fn()`) to prevent compiler optimization from eliminating the benchmarked code.",
						Severity:     d.rule.Severity,
						Category:     d.rule.Category,
						CodeContext:  ctx.CodeContext,
					})
				}
			}

			return true
		})
	}

	return findings
}

// isBenchmarkFunc checks if a function declaration is a benchmark function.
func isBenchmarkFunc(fn *ast.FuncDecl) bool {
	if fn.Name == nil || len(fn.Name.Name) < 9 {
		return false
	}
	if fn.Name.Name[:9] != "Benchmark" {
		return false
	}
	// Check for *testing.B parameter
	if fn.Type == nil || fn.Type.Params == nil || len(fn.Type.Params.List) == 0 {
		return false
	}
	for _, param := range fn.Type.Params.List {
		if starExpr, ok := param.Type.(*ast.StarExpr); ok {
			if selExpr, ok := starExpr.X.(*ast.SelectorExpr); ok {
				if ident, ok := selExpr.X.(*ast.Ident); ok {
					if ident.Name == "testing" && selExpr.Sel.Name == "B" {
						return true
					}
				}
			}
		}
	}
	return false
}
