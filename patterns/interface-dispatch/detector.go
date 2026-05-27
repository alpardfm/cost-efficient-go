// Package interface_dispatch detects interface method calls in tight loops
// where concrete types should be used to enable compiler inlining and reduce
// indirect call overhead.
package interface_dispatch

import (
	"go/ast"

	"github.com/alpardfm/cost-efficient-go/types"
)


// detector implements types.Detector for interface dispatch anti-patterns.
// It is stateless and safe for concurrent use.
type detector struct {
	rule types.Rule
}

// NewDetector creates a new interface dispatch detector.
func NewDetector() types.Detector {
	return &detector{
		rule: types.Rule{
			ID:          "CEG-010",
			Name:        "Interface Dispatch vs Concrete Type",
			Description: "Detects interface method calls inside tight loops (1M+ iterations) where the concrete type could be used instead. Interface dispatch prevents compiler inlining and adds indirect call overhead (~1-3ns per call), which accumulates in hot paths.",
			Severity:    types.Minor,
			Category:    types.Memory,
			Suggestion:  "Use the 'interface at boundary, concrete internally' pattern: accept interfaces at API boundaries for flexibility, but use concrete types in hot inner loops to enable compiler inlining.",
			ReferenceLinks: []string{
				"https://github.com/alpardfm/cost-efficient-go/tree/main/patterns/interface-dispatch",
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

	// Look for interface method calls inside for/range loops that indicate
	// potential interface dispatch overhead in hot paths.
	var findings []types.Finding

	rangeStmt, ok := ctx.Node.(*ast.RangeStmt)
	if !ok {
		return []types.Finding{}
	}

	// Walk the body of the range statement looking for call expressions
	// through interface variables (method calls on non-concrete receivers).
	ast.Inspect(rangeStmt.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		if isInterfaceMethodCall(call) {
			findings = append(findings, types.Finding{
				RuleID:       d.rule.ID,
				FilePath:     ctx.FilePath,
				Line:         ctx.Line,
				Explanation:  "Interface method call detected inside a loop. Interface dispatch prevents compiler inlining and adds ~1-3ns overhead per call, which accumulates in tight loops.",
				SuggestedFix: "Use concrete type in the loop body. Accept interface at the boundary and type-assert to concrete type before entering the hot loop.",
				Severity:     d.rule.Severity,
				Category:     d.rule.Category,
				CodeContext:  ctx.CodeContext,
			})
		}

		return true
	})

	return findings
}

// isInterfaceMethodCall checks if a call expression looks like a method call
// through an interface (selector expression on an identifier).
func isInterfaceMethodCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	// Check if the receiver is a simple identifier (potential interface variable)
	_, isIdent := sel.X.(*ast.Ident)
	if !isIdent {
		return false
	}

	// Common method names that suggest interface dispatch in hot paths
	switch sel.Sel.Name {
	case "Process", "Transform", "Execute", "Handle", "Compute",
		"Calculate", "Apply", "Run", "Do", "Call":
		return true
	}

	return false
}
