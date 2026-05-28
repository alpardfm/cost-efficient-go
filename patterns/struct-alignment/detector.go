// Package struct_alignment detects struct field ordering that causes excessive
// memory padding, wasting heap space and reducing cache efficiency.
package struct_alignment

import (
	"go/ast"

	"github.com/alpardfm/cost-efficient-go/types"
)

// detector implements types.Detector for struct alignment anti-patterns.
// It is stateless and safe for concurrent use.
type detector struct {
	rule types.Rule
}

// NewDetector creates a new struct alignment detector.
func NewDetector() types.Detector {
	return &detector{
		rule: types.Rule{
			ID:          "CEG-018",
			Name:        "Struct Field Alignment",
			Description: "Detects struct definitions where field ordering causes excessive padding bytes due to memory alignment requirements. Reordering fields from largest to smallest alignment reduces struct size, lowering heap usage and GC pressure at scale.",
			Severity:    types.Minor,
			Category:    types.Memory,
			Suggestion:  "Reorder struct fields from largest to smallest alignment requirement (e.g., pointers/strings first, then int64, int32, int16/bool last) to minimize padding bytes.",
			ReferenceLinks: []string{
				"https://github.com/alpardfm/cost-efficient-go/tree/main/patterns/struct-alignment",
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

	// Look for struct type declarations that may have suboptimal field ordering.
	var findings []types.Finding

	genDecl, ok := ctx.Node.(*ast.GenDecl)
	if !ok {
		return []types.Finding{}
	}

	for _, spec := range genDecl.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			continue
		}

		// Check if the struct has fields that could benefit from reordering
		if structType.Fields != nil && len(structType.Fields.List) > 1 {
			if hasPotentialPaddingIssue(structType) {
				findings = append(findings, types.Finding{
					RuleID:       d.rule.ID,
					FilePath:     ctx.FilePath,
					Line:         ctx.Line,
					Explanation:  "Struct field ordering may cause excessive padding. Fields with smaller alignment requirements placed between larger ones introduce padding bytes.",
					SuggestedFix: "Reorder fields from largest to smallest alignment: pointers/strings (8 bytes), int64/float64 (8 bytes), int32/float32 (4 bytes), int16 (2 bytes), bool/int8/byte (1 byte).",
					Severity:     d.rule.Severity,
					Category:     d.rule.Category,
					CodeContext:  ctx.CodeContext,
					Confidence:   types.ConfidenceLow,
				})
			}
		}
	}

	return findings
}

// hasPotentialPaddingIssue checks if a struct's field ordering might cause padding.
// A simple heuristic: if a small-alignment field appears before a large-alignment field,
// there may be unnecessary padding.
func hasPotentialPaddingIssue(structType *ast.StructType) bool {
	if structType.Fields == nil || len(structType.Fields.List) < 2 {
		return false
	}

	// Simple heuristic: check if any field with a small type identifier
	// appears before a field with a large type identifier.
	var sawSmall bool
	for _, field := range structType.Fields.List {
		typeName := fieldTypeName(field)
		if isSmallType(typeName) {
			sawSmall = true
		} else if sawSmall && isLargeType(typeName) {
			return true
		}
	}

	return false
}

// fieldTypeName extracts the type name from a field as a string.
func fieldTypeName(field *ast.Field) string {
	switch t := field.Type.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.ArrayType:
		return "array"
	case *ast.MapType:
		return "map"
	case *ast.StarExpr:
		return "pointer"
	default:
		return ""
	}
}

// isSmallType returns true for types with 1-byte alignment.
func isSmallType(name string) bool {
	switch name {
	case "bool", "byte", "int8", "uint8":
		return true
	}
	return false
}

// isLargeType returns true for types with 8-byte alignment.
func isLargeType(name string) bool {
	switch name {
	case "string", "int", "uint", "int64", "uint64", "float64",
		"pointer", "map", "array", "interface":
		return true
	}
	return false
}
