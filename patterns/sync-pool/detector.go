// Package sync_pool detects missing sync.Pool usage for buffer reuse,
// which causes excessive heap allocations and GC pressure at high throughput.
package sync_pool

import (
	"go/ast"

	"github.com/alpardfm/cost-efficient-go/types"
)


// detector implements types.Detector for sync.Pool anti-patterns.
// It is stateless and safe for concurrent use.
type detector struct {
	rule types.Rule
}

// NewDetector creates a new sync.Pool detector.
func NewDetector() types.Detector {
	return &detector{
		rule: types.Rule{
			ID:          "CEG-019",
			Name:        "sync.Pool Buffer Reuse",
			Description: "Detects repeated buffer allocations in hot paths that should use sync.Pool for reuse. At high throughput (100K+ req/sec), allocating new buffers per request creates massive GC pressure and wastes memory bandwidth.",
			Severity:    types.Major,
			Category:    types.Memory,
			Suggestion:  "Use sync.Pool to reuse buffers across requests. Wrap buffer allocation in a pool with appropriate New function, and return buffers after use to reduce GC pressure.",
			ReferenceLinks: []string{
				"https://github.com/alpardfm/cost-efficient-go/tree/main/patterns/sync-pool",
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

	// Look for make([]byte, ...) calls inside loops that suggest
	// repeated buffer allocation without pooling.
	var findings []types.Finding

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

	// Walk the body looking for make([]byte, ...) calls that indicate
	// buffer allocation inside a loop.
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		if isBufferAllocation(call) {
			findings = append(findings, types.Finding{
				RuleID:       d.rule.ID,
				FilePath:     ctx.FilePath,
				Line:         ctx.Line,
				Explanation:  "Buffer allocation detected inside a loop. Each iteration allocates a new buffer on the heap, creating GC pressure at high throughput.",
				SuggestedFix: "Use sync.Pool to reuse buffers. Create a pool with New: func() interface{} { return make([]byte, size) } and Get/Put buffers in the loop.",
				Severity:     d.rule.Severity,
				Category:     d.rule.Category,
				CodeContext:  ctx.CodeContext,
			})
		}

		return true
	})

	return findings
}

// isBufferAllocation checks if a call expression is a make([]byte, ...) call.
func isBufferAllocation(call *ast.CallExpr) bool {
	ident, ok := call.Fun.(*ast.Ident)
	if !ok || ident.Name != "make" {
		return false
	}

	if len(call.Args) < 1 {
		return false
	}

	// Check if the first argument is []byte
	arrayType, ok := call.Args[0].(*ast.ArrayType)
	if !ok {
		return false
	}

	eltIdent, ok := arrayType.Elt.(*ast.Ident)
	if !ok {
		return false
	}

	return eltIdent.Name == "byte"
}
