// Package map_internals detects inefficient map usage patterns that cause
// excessive memory overhead, GC pressure, and poor cache locality.
package map_internals

import (
	"go/ast"

	"github.com/alpardfm/cost-efficient-go/types"
)


// detector implements types.Detector for map memory overhead anti-patterns.
// It is stateless and safe for concurrent use.
type detector struct {
	rule types.Rule
}

// NewDetector creates a new map internals detector.
func NewDetector() types.Detector {
	return &detector{
		rule: types.Rule{
			ID:          "CEG-012",
			Name:        "Map Internals & Memory Overhead",
			Description: "Detects map usage patterns that incur excessive memory overhead compared to alternative data structures. Go maps carry ~24 bytes of overhead per entry (tophash, bucket padding, overflow pointers) and exhibit poor cache locality, high GC pressure, and unpredictable iteration order.",
			Severity:    types.Minor,
			Category:    types.Memory,
			Suggestion:  "Consider using a slice of structs or parallel arrays when sequential integer keys are used, iteration is frequent, or memory is constrained. Pre-allocate maps with make(map[K]V, size) to avoid rehashing. Use map[T]struct{} instead of map[T]bool for sets.",
			ReferenceLinks: []string{
				"https://github.com/alpardfm/cost-efficient-go/tree/main/patterns/map-internals",
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

	// Detect map creation without pre-allocation inside loops or functions
	// that could benefit from size hints.
	ast.Inspect(ctx.Node, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		if isUnpreallocatedMapMake(call) {
			findings = append(findings, types.Finding{
				RuleID:       d.rule.ID,
				FilePath:     ctx.FilePath,
				Line:         ctx.Line,
				Explanation:  "Map created without size hint. Without pre-allocation, the map will rehash multiple times as it grows, wasting ~25% memory on average and causing additional allocations.",
				SuggestedFix: "Use make(map[K]V, expectedSize) to pre-allocate the map and avoid rehashing overhead.",
				Severity:     d.rule.Severity,
				Category:     d.rule.Category,
				CodeContext:  ctx.CodeContext,
			})
		}

		return true
	})

	return findings
}

// isUnpreallocatedMapMake checks if a call expression is a make(map[...]...) without a size hint.
func isUnpreallocatedMapMake(call *ast.CallExpr) bool {
	ident, ok := call.Fun.(*ast.Ident)
	if !ok || ident.Name != "make" {
		return false
	}

	// make(map[K]V) has 1 arg (the type), make(map[K]V, size) has 2 args
	if len(call.Args) != 1 {
		return false
	}

	// Check if the first argument is a map type
	_, ok = call.Args[0].(*ast.MapType)
	return ok
}
