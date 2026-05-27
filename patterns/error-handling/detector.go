// Package error_handling detects inefficient error creation patterns that cause
// unnecessary heap allocations on hot paths, increasing GC pressure and cost.
package error_handling

import (
	"go/ast"

	"github.com/alpardfm/cost-efficient-go/types"
)


// detector implements types.Detector for error handling anti-patterns.
// It is stateless and safe for concurrent use.
type detector struct {
	rule types.Rule
}

// NewDetector creates a new error handling detector.
func NewDetector() types.Detector {
	return &detector{
		rule: types.Rule{
			ID:          "CEG-007",
			Name:        "Error Handling Efficiency",
			Description: "Detects inefficient error creation patterns (errors.New, fmt.Errorf) on hot paths that allocate on every call. At scale with high error rates, this creates millions of unnecessary heap allocations per day, increasing GC pressure and memory costs.",
			Severity:    types.Major,
			Category:    types.ErrorHandling,
			Suggestion:  "Use sentinel errors (package-level var) or pre-allocated custom error types instead of creating new error values on every call. Sentinel errors are zero-allocation on use.",
			ReferenceLinks: []string{
				"https://github.com/alpardfm/cost-efficient-go/tree/main/patterns/error-handling",
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

	// Look for calls to errors.New() or fmt.Errorf() inside loops,
	// which indicate allocating error creation on hot paths.
	forStmt, isFor := ctx.Node.(*ast.ForStmt)
	rangeStmt, isRange := ctx.Node.(*ast.RangeStmt)

	var body *ast.BlockStmt
	switch {
	case isFor:
		body = forStmt.Body
	case isRange:
		body = rangeStmt.Body
	default:
		return []types.Finding{}
	}

	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		if isAllocatingErrorCall(call) {
			findings = append(findings, types.Finding{
				RuleID:       d.rule.ID,
				FilePath:     ctx.FilePath,
				Line:         ctx.Line,
				Explanation:  "Allocating error creation (errors.New or fmt.Errorf) detected inside a loop. Each call allocates a new error on the heap, causing GC pressure on hot paths.",
				SuggestedFix: "Replace with a sentinel error (package-level var) or pre-allocated custom error type to achieve zero allocations on the hot path.",
				Severity:     d.rule.Severity,
				Category:     d.rule.Category,
				CodeContext:  ctx.CodeContext,
			})
		}

		return true
	})

	return findings
}

// isAllocatingErrorCall checks if a call expression is errors.New() or fmt.Errorf().
func isAllocatingErrorCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	// errors.New()
	if ident.Name == "errors" && sel.Sel.Name == "New" {
		return true
	}

	// fmt.Errorf()
	if ident.Name == "fmt" && sel.Sel.Name == "Errorf" {
		return true
	}

	return false
}
