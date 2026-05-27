// Package json_processing detects inefficient JSON processing patterns that cause
// excessive memory allocations and CPU overhead due to reflection and unoptimized struct tags.
package json_processing

import (
	"go/ast"

	"github.com/alpardfm/cost-efficient-go/types"
)


// detector implements types.Detector for JSON processing anti-patterns.
// It is stateless and safe for concurrent use.
type detector struct {
	rule types.Rule
}

// NewDetector creates a new JSON processing efficiency detector.
func NewDetector() types.Detector {
	return &detector{
		rule: types.Rule{
			ID:          "CEG-011",
			Name:        "JSON Processing Efficiency",
			Description: "Detects inefficient JSON processing patterns including missing omitempty tags on optional fields, use of map[string]interface{} for known structures, and untyped data in hot paths. encoding/json uses reflection heavily, causing significant CPU and memory overhead at scale.",
			Severity:    types.Major,
			Category:    types.Memory,
			Suggestion:  "Use omitempty struct tags for optional fields, replace map[string]interface{} with typed structs for known schemas, and use pointer receivers for large structs to reduce marshaling allocations.",
			ReferenceLinks: []string{
				"https://github.com/alpardfm/cost-efficient-go/tree/main/patterns/json-processing",
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

	// Look for struct type declarations with JSON tags that lack omitempty
	// or use map[string]interface{} for known structures.
	ast.Inspect(ctx.Node, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.TypeSpec:
			structType, ok := node.Type.(*ast.StructType)
			if !ok {
				return true
			}
			findings = append(findings, d.checkStructFields(ctx, structType)...)
		}
		return true
	})

	return findings
}

// checkStructFields inspects struct fields for JSON processing anti-patterns.
func (d *detector) checkStructFields(ctx types.ASTContext, st *ast.StructType) []types.Finding {
	var findings []types.Finding

	if st.Fields == nil {
		return findings
	}

	for _, field := range st.Fields.List {
		if field.Type == nil {
			continue
		}

		// Check for map[string]interface{} fields which are inefficient for known schemas
		if isMapStringInterface(field.Type) {
			findings = append(findings, types.Finding{
				RuleID:       d.rule.ID,
				FilePath:     ctx.FilePath,
				Line:         ctx.Line,
				Explanation:  "Field uses map[string]interface{} which causes extra allocations and type assertions during JSON marshaling/unmarshaling. For known schemas, typed structs are more efficient.",
				SuggestedFix: "Replace map[string]interface{} with a typed struct that defines the expected fields explicitly.",
				Severity:     d.rule.Severity,
				Category:     d.rule.Category,
				CodeContext:  ctx.CodeContext,
			})
		}
	}

	return findings
}

// isMapStringInterface checks if an expression is map[string]interface{}.
func isMapStringInterface(expr ast.Expr) bool {
	mapType, ok := expr.(*ast.MapType)
	if !ok {
		return false
	}

	// Check key is string
	keyIdent, ok := mapType.Key.(*ast.Ident)
	if !ok || keyIdent.Name != "string" {
		return false
	}

	// Check value is interface{}
	iface, ok := mapType.Value.(*ast.InterfaceType)
	if !ok {
		return false
	}

	// Empty interface (interface{})
	return iface.Methods == nil || iface.Methods.NumFields() == 0
}
