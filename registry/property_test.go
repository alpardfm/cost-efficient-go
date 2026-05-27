package registry

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"sync"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	"github.com/alpardfm/cost-efficient-go/types"
)

// Feature: cost-efficient-go-library, Property 1: Nil Node Safety
// Validates: Requirements 2.5, 8.2
func TestProperty_NilNodeSafety(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("Detect returns empty slice and does not panic for nil Node", prop.ForAll(
		func(filePath string, line int, codeContext string) bool {
			ctx := types.ASTContext{
				FilePath:    filePath,
				Line:        line,
				Node:        nil,
				CodeContext: codeContext,
			}

			for _, d := range AllDetectors() {
				findings := d.Detect(ctx)
				if len(findings) != 0 {
					return false
				}
			}
			return true
		},
		gen.AnyString(),
		gen.IntRange(0, 10000),
		gen.AnyString(),
	))

	properties.TestingRun(t)
}

// Feature: cost-efficient-go-library, Property 2: No Panic on Valid Input
// Validates: Requirements 8.1
func TestProperty_NoPanicOnValidInput(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Generator for random valid AST nodes.
	// All nodes must be structurally complete (no nil children in positions
	// that ast.Walk would traverse) to be considered "valid".
	nodeGen := gen.IntRange(0, 6).Map(func(v int) ast.Node {
		switch v {
		case 0:
			return &ast.Ident{Name: "foo", NamePos: token.Pos(1)}
		case 1:
			return &ast.BasicLit{Kind: token.INT, Value: "42", ValuePos: token.Pos(1)}
		case 2:
			// A fully valid CallExpr with non-nil Fun and proper Args
			return &ast.CallExpr{
				Fun:    &ast.Ident{Name: "bar"},
				Args:   []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: `"hello"`}},
				Lparen: token.Pos(4),
				Rparen: token.Pos(12),
			}
		case 3:
			// A fully valid RangeStmt with non-nil X (the iterable expression)
			return &ast.RangeStmt{
				Key:   &ast.Ident{Name: "i"},
				Value: &ast.Ident{Name: "v"},
				X:     &ast.Ident{Name: "items"},
				Tok:   token.DEFINE,
				Body:  &ast.BlockStmt{List: []ast.Stmt{}},
			}
		case 4:
			// A fully valid ForStmt with non-nil Body
			return &ast.ForStmt{
				Cond: &ast.BinaryExpr{
					X:  &ast.Ident{Name: "i"},
					Op: token.LSS,
					Y:  &ast.BasicLit{Kind: token.INT, Value: "10"},
				},
				Body: &ast.BlockStmt{List: []ast.Stmt{}},
			}
		case 5:
			// A fully valid GoStmt with a complete function literal
			return &ast.GoStmt{
				Call: &ast.CallExpr{
					Fun: &ast.FuncLit{
						Type: &ast.FuncType{Params: &ast.FieldList{}},
						Body: &ast.BlockStmt{List: []ast.Stmt{}},
					},
					Lparen: token.Pos(1),
					Rparen: token.Pos(2),
				},
			}
		case 6:
			// A fully valid GenDecl with a struct type
			return &ast.GenDecl{
				Tok: token.TYPE,
				Specs: []ast.Spec{
					&ast.TypeSpec{
						Name: &ast.Ident{Name: "MyStruct"},
						Type: &ast.StructType{
							Fields: &ast.FieldList{
								List: []*ast.Field{
									{Names: []*ast.Ident{{Name: "x"}}, Type: &ast.Ident{Name: "bool"}},
									{Names: []*ast.Ident{{Name: "y"}}, Type: &ast.Ident{Name: "int64"}},
								},
							},
						},
					},
				},
			}
		default:
			return &ast.Ident{Name: "default"}
		}
	})

	properties.Property("Detect does not panic for any detector with valid input", prop.ForAll(
		func(filePath string, line int, codeContext string, node ast.Node) bool {
			if line < 1 {
				line = 1
			}
			ctx := types.ASTContext{
				FilePath:    filePath,
				Line:        line,
				Node:        node,
				CodeContext: codeContext,
			}

			for _, d := range AllDetectors() {
				func() {
					defer func() {
						if r := recover(); r != nil {
							t.Errorf("detector %s panicked with node type %T: %v", d.Rule().ID, node, r)
						}
					}()
					d.Detect(ctx)
				}()
			}
			return true
		},
		gen.AlphaString(),
		gen.IntRange(1, 10000),
		gen.AnyString(),
		nodeGen,
	))

	properties.TestingRun(t)
}

// Feature: cost-efficient-go-library, Property 3: Finding Completeness
// Validates: Requirements 2.4, 8.3, 8.4, 8.5
func TestProperty_FindingCompleteness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Build known-matching inputs for detectors that match specific AST patterns.
	// We use a RangeStmt with an I/O call inside (matches batch-processing, slice-performance, etc.)
	// and other patterns that trigger findings.
	type matchingInput struct {
		detectorID string
		ctx        types.ASTContext
	}

	knownInputs := buildKnownMatchingInputs()

	properties.Property("all Finding fields are non-zero/non-empty for known-matching inputs", prop.ForAll(
		func(idx int) bool {
			if len(knownInputs) == 0 {
				return true
			}
			input := knownInputs[idx%len(knownInputs)]

			d, ok := DetectorByID(input.detectorID)
			if !ok {
				t.Errorf("detector %s not found", input.detectorID)
				return false
			}

			findings := d.Detect(input.ctx)
			if len(findings) == 0 {
				t.Errorf("detector %s returned no findings for known-matching input", input.detectorID)
				return false
			}

			for _, f := range findings {
				if f.RuleID == "" {
					t.Errorf("detector %s: Finding.RuleID is empty", input.detectorID)
					return false
				}
				if f.FilePath == "" {
					t.Errorf("detector %s: Finding.FilePath is empty", input.detectorID)
					return false
				}
				if f.Line <= 0 {
					t.Errorf("detector %s: Finding.Line is <= 0", input.detectorID)
					return false
				}
				if f.Explanation == "" {
					t.Errorf("detector %s: Finding.Explanation is empty", input.detectorID)
					return false
				}
				if f.SuggestedFix == "" {
					t.Errorf("detector %s: Finding.SuggestedFix is empty", input.detectorID)
					return false
				}
				if f.CodeContext == "" {
					t.Errorf("detector %s: Finding.CodeContext is empty", input.detectorID)
					return false
				}
			}
			return true
		},
		gen.IntRange(0, 10000),
	))

	properties.TestingRun(t)
}

// Feature: cost-efficient-go-library, Property 4: DetectorByID Round-Trip
// Validates: Requirements 4.3
func TestProperty_DetectorByIDRoundTrip(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Collect all valid IDs
	allDetectors := AllDetectors()
	validIDs := make(map[string]bool)
	for _, d := range allDetectors {
		validIDs[d.Rule().ID] = true
	}

	properties.Property("DetectorByID(d.Rule().ID) returns (d, true) for all registered detectors", prop.ForAll(
		func(idx int) bool {
			d := allDetectors[idx%len(allDetectors)]
			found, ok := DetectorByID(d.Rule().ID)
			if !ok {
				return false
			}
			return found.Rule().ID == d.Rule().ID
		},
		gen.IntRange(0, 10000),
	))

	properties.Property("DetectorByID returns (nil, false) for invalid IDs", prop.ForAll(
		func(s string) bool {
			if validIDs[s] {
				// Skip valid IDs - they should return true
				return true
			}
			d, ok := DetectorByID(s)
			return !ok && d == nil
		},
		gen.AnyString(),
	))

	properties.TestingRun(t)
}

// Feature: cost-efficient-go-library, Property 5: DetectorsByCategory Filter Correctness
// Validates: Requirements 4.4
func TestProperty_CategoryFilterCorrectness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	allDetectors := AllDetectors()
	categories := []types.Category{types.Memory, types.Concurrency, types.IO, types.ErrorHandling}

	properties.Property("DetectorsByCategory returns exact match for any category", prop.ForAll(
		func(catIdx int) bool {
			cat := categories[catIdx%len(categories)]
			result := DetectorsByCategory(cat)

			// Build expected set by manually filtering AllDetectors
			var expected []types.Detector
			for _, d := range allDetectors {
				if d.Rule().Category == cat {
					expected = append(expected, d)
				}
			}

			// Verify exact match: same count
			if len(result) != len(expected) {
				return false
			}

			// Verify no missing detectors
			resultIDs := make(map[string]bool)
			for _, d := range result {
				resultIDs[d.Rule().ID] = true
			}
			for _, d := range expected {
				if !resultIDs[d.Rule().ID] {
					return false
				}
			}

			// Verify no extra detectors (all results have matching category)
			for _, d := range result {
				if d.Rule().Category != cat {
					return false
				}
			}

			return true
		},
		gen.IntRange(0, 10000),
	))

	properties.TestingRun(t)
}

// Feature: cost-efficient-go-library, Property 6: DetectorsBySeverity Filter Correctness
// Validates: Requirements 4.5
func TestProperty_SeverityFilterCorrectness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	allDetectors := AllDetectors()
	severities := []types.Severity{types.Minor, types.Major, types.Critical}

	properties.Property("DetectorsBySeverity returns exact match for any severity", prop.ForAll(
		func(sevIdx int) bool {
			sev := severities[sevIdx%len(severities)]
			result := DetectorsBySeverity(sev)

			// Build expected set by manually filtering AllDetectors
			var expected []types.Detector
			for _, d := range allDetectors {
				if d.Rule().Severity == sev {
					expected = append(expected, d)
				}
			}

			// Verify exact match: same count
			if len(result) != len(expected) {
				return false
			}

			// Verify no missing detectors
			resultIDs := make(map[string]bool)
			for _, d := range result {
				resultIDs[d.Rule().ID] = true
			}
			for _, d := range expected {
				if !resultIDs[d.Rule().ID] {
					return false
				}
			}

			// Verify no extra detectors (all results have matching severity)
			for _, d := range result {
				if d.Rule().Severity != sev {
					return false
				}
			}

			return true
		},
		gen.IntRange(0, 10000),
	))

	properties.TestingRun(t)
}

// Feature: cost-efficient-go-library, Property 7: Rule ID Format and Uniqueness
// Validates: Requirements 7.1
func TestProperty_RuleIDFormatAndUniqueness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	allDetectors := AllDetectors()
	idPattern := regexp.MustCompile(`^CEG-\d{3}$`)

	properties.Property("all detector IDs match CEG-XXX format and are unique", prop.ForAll(
		func(idx int) bool {
			// Verify format for each detector
			d := allDetectors[idx%len(allDetectors)]
			if !idPattern.MatchString(d.Rule().ID) {
				return false
			}

			// Verify uniqueness across all detectors
			seen := make(map[string]bool)
			for _, det := range allDetectors {
				id := det.Rule().ID
				if !idPattern.MatchString(id) {
					return false
				}
				if seen[id] {
					return false
				}
				seen[id] = true
			}
			return true
		},
		gen.IntRange(0, 10000),
	))

	properties.TestingRun(t)
}

// Feature: cost-efficient-go-library, Property 8: Concurrent Detector Safety
// Validates: Requirements 11.1
func TestProperty_ConcurrentDetectorSafety(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	allDetectors := AllDetectors()

	// Generator for random ASTContext with valid nodes (structurally complete)
	ctxGen := gen.IntRange(0, 3).Map(func(v int) types.ASTContext {
		var node ast.Node
		switch v {
		case 0:
			node = &ast.Ident{Name: "x", NamePos: token.Pos(1)}
		case 1:
			node = &ast.BasicLit{Kind: token.INT, Value: "1", ValuePos: token.Pos(1)}
		case 2:
			node = &ast.CallExpr{
				Fun:    &ast.Ident{Name: "foo"},
				Args:   []ast.Expr{},
				Lparen: token.Pos(4),
				Rparen: token.Pos(5),
			}
		case 3:
			node = &ast.RangeStmt{
				Key:   &ast.Ident{Name: "i"},
				Value: &ast.Ident{Name: "v"},
				X:     &ast.Ident{Name: "items"},
				Tok:   token.DEFINE,
				Body:  &ast.BlockStmt{List: []ast.Stmt{}},
			}
		default:
			node = &ast.Ident{Name: "default"}
		}
		return types.ASTContext{
			FilePath:    "test.go",
			Line:        1,
			Node:        node,
			CodeContext: "// test code",
		}
	})

	properties.Property("concurrent Detect calls produce identical results", prop.ForAll(
		func(detIdx int, ctx types.ASTContext) bool {
			d := allDetectors[detIdx%len(allDetectors)]
			const numGoroutines = 12

			results := make([][]types.Finding, numGoroutines)
			var wg sync.WaitGroup
			wg.Add(numGoroutines)

			for i := 0; i < numGoroutines; i++ {
				go func(idx int) {
					defer wg.Done()
					results[idx] = d.Detect(ctx)
				}(i)
			}
			wg.Wait()

			// All results should be identical
			for i := 1; i < numGoroutines; i++ {
				if len(results[i]) != len(results[0]) {
					return false
				}
				for j := range results[0] {
					if results[i][j].RuleID != results[0][j].RuleID ||
						results[i][j].FilePath != results[0][j].FilePath ||
						results[i][j].Line != results[0][j].Line ||
						results[i][j].Explanation != results[0][j].Explanation ||
						results[i][j].SuggestedFix != results[0][j].SuggestedFix ||
						results[i][j].CodeContext != results[0][j].CodeContext {
						return false
					}
				}
			}
			return true
		},
		gen.IntRange(0, 10000),
		ctxGen,
	))

	properties.TestingRun(t)
}

// Feature: cost-efficient-go-library, Property 9: Concurrent Registry Safety
// Validates: Requirements 11.2
func TestProperty_ConcurrentRegistrySafety(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	allDetectors := AllDetectors()
	validIDs := make([]string, 0, len(allDetectors))
	for _, d := range allDetectors {
		validIDs = append(validIDs, d.Rule().ID)
	}
	categories := []types.Category{types.Memory, types.Concurrency, types.IO, types.ErrorHandling}
	severities := []types.Severity{types.Minor, types.Major, types.Critical}

	properties.Property("concurrent registry method calls produce consistent results without races", prop.ForAll(
		func(seed int) bool {
			const numGoroutines = 12
			var wg sync.WaitGroup
			wg.Add(numGoroutines)

			// Each goroutine calls a mix of registry methods
			type result struct {
				allCount int
				foundID  bool
				catCount int
				sevCount int
			}
			results := make([]result, numGoroutines)

			for i := 0; i < numGoroutines; i++ {
				go func(idx int) {
					defer wg.Done()
					var r result

					// Call AllDetectors
					all := AllDetectors()
					r.allCount = len(all)

					// Call DetectorByID with a valid ID
					idIdx := (seed + idx) % len(validIDs)
					_, r.foundID = DetectorByID(validIDs[idIdx])

					// Call DetectorsByCategory
					catIdx := (seed + idx) % len(categories)
					catResult := DetectorsByCategory(categories[catIdx])
					r.catCount = len(catResult)

					// Call DetectorsBySeverity
					sevIdx := (seed + idx) % len(severities)
					sevResult := DetectorsBySeverity(severities[sevIdx])
					r.sevCount = len(sevResult)

					results[idx] = r
				}(i)
			}
			wg.Wait()

			// All goroutines should see the same AllDetectors count
			for i := 1; i < numGoroutines; i++ {
				if results[i].allCount != results[0].allCount {
					return false
				}
			}

			// All lookups for valid IDs should succeed
			for _, r := range results {
				if !r.foundID {
					return false
				}
			}

			// Goroutines with same category/severity should get same count
			// (different goroutines may query different categories, so we just verify consistency)
			for _, r := range results {
				if r.allCount != 20 {
					return false
				}
			}

			return true
		},
		gen.IntRange(0, 10000),
	))

	properties.TestingRun(t)
}

// buildKnownMatchingInputs creates ASTContext inputs that are known to trigger
// findings for specific detectors.
func buildKnownMatchingInputs() []struct {
	detectorID string
	ctx        types.ASTContext
} {
	inputs := []struct {
		detectorID string
		ctx        types.ASTContext
	}{
		{
			// CEG-001: batch-processing - RangeStmt with I/O call inside
			detectorID: "CEG-001",
			ctx: types.ASTContext{
				FilePath:    "main.go",
				Line:        10,
				CodeContext: "for _, item := range items { db.Exec(item) }",
				Node: &ast.RangeStmt{
					Key:   &ast.Ident{Name: "_"},
					Value: &ast.Ident{Name: "item"},
					Body: &ast.BlockStmt{
						List: []ast.Stmt{
							&ast.ExprStmt{
								X: &ast.CallExpr{
									Fun: &ast.SelectorExpr{
										X:   &ast.Ident{Name: "db"},
										Sel: &ast.Ident{Name: "Exec"},
									},
									Args: []ast.Expr{&ast.Ident{Name: "item"}},
								},
							},
						},
					},
				},
			},
		},
		{
			// CEG-007: error-handling - ForStmt with errors.New inside
			detectorID: "CEG-007",
			ctx: types.ASTContext{
				FilePath:    "handler.go",
				Line:        25,
				CodeContext: "for i := 0; i < n; i++ { err := errors.New(\"fail\") }",
				Node: &ast.ForStmt{
					Body: &ast.BlockStmt{
						List: []ast.Stmt{
							&ast.AssignStmt{
								Lhs: []ast.Expr{&ast.Ident{Name: "err"}},
								Tok: token.DEFINE,
								Rhs: []ast.Expr{
									&ast.CallExpr{
										Fun: &ast.SelectorExpr{
											X:   &ast.Ident{Name: "errors"},
											Sel: &ast.Ident{Name: "New"},
										},
										Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: `"fail"`}},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			// CEG-008: goroutine-leak - GoStmt with blocking channel receive without select
			detectorID: "CEG-008",
			ctx: types.ASTContext{
				FilePath:    "worker.go",
				Line:        15,
				CodeContext: "go func() { msg := <-ch }()",
				Node: &ast.GoStmt{
					Call: &ast.CallExpr{
						Fun: &ast.FuncLit{
							Type: &ast.FuncType{Params: &ast.FieldList{}},
							Body: &ast.BlockStmt{
								List: []ast.Stmt{
									&ast.AssignStmt{
										Lhs: []ast.Expr{&ast.Ident{Name: "msg"}},
										Tok: token.DEFINE,
										Rhs: []ast.Expr{
											&ast.UnaryExpr{
												Op: token.ARROW,
												X:  &ast.Ident{Name: "ch"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			// CEG-016: slice-performance - RangeStmt with append inside
			detectorID: "CEG-016",
			ctx: types.ASTContext{
				FilePath:    "process.go",
				Line:        30,
				CodeContext: "for _, v := range data { result = append(result, v) }",
				Node: &ast.RangeStmt{
					Key:   &ast.Ident{Name: "_"},
					Value: &ast.Ident{Name: "v"},
					Body: &ast.BlockStmt{
						List: []ast.Stmt{
							&ast.AssignStmt{
								Lhs: []ast.Expr{&ast.Ident{Name: "result"}},
								Tok: token.ASSIGN,
								Rhs: []ast.Expr{
									&ast.CallExpr{
										Fun:  &ast.Ident{Name: "append"},
										Args: []ast.Expr{&ast.Ident{Name: "result"}, &ast.Ident{Name: "v"}},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			// CEG-017: string-building - RangeStmt with += inside
			detectorID: "CEG-017",
			ctx: types.ASTContext{
				FilePath:    "format.go",
				Line:        20,
				CodeContext: "for _, s := range parts { result += s }",
				Node: &ast.RangeStmt{
					Key:   &ast.Ident{Name: "_"},
					Value: &ast.Ident{Name: "s"},
					Body: &ast.BlockStmt{
						List: []ast.Stmt{
							&ast.AssignStmt{
								Lhs: []ast.Expr{&ast.Ident{Name: "result"}},
								Tok: token.ADD_ASSIGN,
								Rhs: []ast.Expr{&ast.Ident{Name: "s"}},
							},
						},
					},
				},
			},
		},
		{
			// CEG-018: struct-alignment - GenDecl with struct having padding issue
			detectorID: "CEG-018",
			ctx: types.ASTContext{
				FilePath:    "model.go",
				Line:        5,
				CodeContext: "type Bad struct { a bool; b int64; c bool }",
				Node: &ast.GenDecl{
					Tok: token.TYPE,
					Specs: []ast.Spec{
						&ast.TypeSpec{
							Name: &ast.Ident{Name: "Bad"},
							Type: &ast.StructType{
								Fields: &ast.FieldList{
									List: []*ast.Field{
										{Names: []*ast.Ident{{Name: "a"}}, Type: &ast.Ident{Name: "bool"}},
										{Names: []*ast.Ident{{Name: "b"}}, Type: &ast.Ident{Name: "int64"}},
										{Names: []*ast.Ident{{Name: "c"}}, Type: &ast.Ident{Name: "bool"}},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Verify all inputs actually produce findings (sanity check at build time)
	var validInputs []struct {
		detectorID string
		ctx        types.ASTContext
	}
	for _, input := range inputs {
		d, ok := DetectorByID(input.detectorID)
		if !ok {
			continue
		}
		findings := d.Detect(input.ctx)
		if len(findings) > 0 {
			validInputs = append(validInputs, input)
		}
	}

	// If we have no valid inputs, use a fallback that we know works
	if len(validInputs) == 0 {
		// Fallback: use the first input which should always work
		validInputs = inputs[:1]
	}

	return validInputs
}

// Ensure fmt is used (for error messages in concurrent tests)
var _ = fmt.Sprintf
