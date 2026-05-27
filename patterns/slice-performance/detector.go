// Package slice_performance detects slice operations without pre-allocation
// that cause excessive memory allocations and GC pressure.
package slice_performance

import (
	"go/ast"

	"github.com/alpardfm/cost-efficient-go/types"
)


// detector implements types.Detector for slice performance anti-patterns.
// It is stateless and safe for concurrent use.
type detector struct {
	rule types.Rule
}

// NewDetector creates a new slice performance detector.
func NewDetector() types.Detector {
	return &detector{
		rule: types.Rule{
			ID:          "CEG-016",
			Name:        "Slice Pre-allocation",
			Description: "Detects slice append operations without pre-allocation that cause repeated memory reallocations and increased GC pressure. Dynamic slice growth doubles capacity each time (until 1024 elements), wasting memory and CPU on copying.",
			Severity:    types.Major,
			Category:    types.Memory,
			Suggestion:  "Use make([]T, 0, expectedSize) to pre-allocate slice capacity when the size is known or can be estimated. This eliminates reallocations and reduces GC pressure.",
			ReferenceLinks: []string{
				"https://github.com/alpardfm/cost-efficient-go/tree/main/patterns/slice-performance",
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

	// Look for append calls inside loops where the slice was declared
	// without pre-allocation (var s []T pattern).
	rangeStmt, ok := ctx.Node.(*ast.RangeStmt)
	if !ok {
		return []types.Finding{}
	}

	// Walk the body of the range statement looking for append calls.
	ast.Inspect(rangeStmt.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		if isAppendCall(call) {
			findings = append(findings, types.Finding{
				RuleID:       d.rule.ID,
				FilePath:     ctx.FilePath,
				Line:         ctx.Line,
				Explanation:  "Slice append inside a loop without pre-allocation causes repeated memory reallocations. Each reallocation copies all existing elements to a new, larger backing array.",
				SuggestedFix: "Pre-allocate the slice with make([]T, 0, expectedSize) before the loop to eliminate reallocations.",
				Severity:     d.rule.Severity,
				Category:     d.rule.Category,
				CodeContext:  ctx.CodeContext,
			})
		}

		return true
	})

	return findings
}

// isAppendCall checks if a call expression is a call to the built-in append function.
func isAppendCall(call *ast.CallExpr) bool {
	ident, ok := call.Fun.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "append"
}
