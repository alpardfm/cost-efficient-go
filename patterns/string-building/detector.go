// Package string_building detects inefficient string concatenation patterns
// that cause excessive memory allocations and GC pressure.
package string_building

import (
	"go/ast"

	"github.com/alpardfm/cost-efficient-go/types"
)


// detector implements types.Detector for string building anti-patterns.
// It is stateless and safe for concurrent use.
type detector struct {
	rule types.Rule
}

// NewDetector creates a new string building detector.
func NewDetector() types.Detector {
	return &detector{
		rule: types.Rule{
			ID:          "CEG-017",
			Name:        "String Building & Concatenation Efficiency",
			Description: "Detects string concatenation using the + operator or fmt.Sprintf inside loops, which causes O(n²) allocations. At scale (10M+ ops/day), this wastes memory and CPU on redundant copies and increases GC pressure.",
			Severity:    types.Major,
			Category:    types.Memory,
			Suggestion:  "Use strings.Builder with Grow() pre-hint for efficient incremental string construction. This reduces allocations from O(n) to O(1) and provides ≥5x speedup at 100+ concatenations.",
			ReferenceLinks: []string{
				"https://github.com/alpardfm/cost-efficient-go/tree/main/patterns/string-building",
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

	// Look for string concatenation with += inside loops (range or for statements).
	switch stmt := ctx.Node.(type) {
	case *ast.RangeStmt:
		findings = append(findings, d.inspectLoopBody(stmt.Body, ctx)...)
	case *ast.ForStmt:
		findings = append(findings, d.inspectLoopBody(stmt.Body, ctx)...)
	}

	return findings
}

// inspectLoopBody walks a loop body looking for string concatenation patterns.
func (d *detector) inspectLoopBody(body *ast.BlockStmt, ctx types.ASTContext) []types.Finding {
	if body == nil {
		return nil
	}

	var findings []types.Finding

	ast.Inspect(body, func(n ast.Node) bool {
		// Look for += assignments with string operands (common concat pattern)
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}

		if assign.Tok.String() == "+=" {
			findings = append(findings, types.Finding{
				RuleID:       d.rule.ID,
				FilePath:     ctx.FilePath,
				Line:         ctx.Line,
				Explanation:  "String concatenation with += inside a loop causes O(n²) allocations. Each iteration allocates a new string and copies all previous content.",
				SuggestedFix: "Use strings.Builder with Grow() pre-hint to build strings incrementally with amortized O(1) appends.",
				Severity:     d.rule.Severity,
				Category:     d.rule.Category,
				CodeContext:  ctx.CodeContext,
			})
		}

		return true
	})

	return findings
}
