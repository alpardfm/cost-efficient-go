// Package efficient_logging detects inefficient logging patterns that cause
// unnecessary heap allocations and GC pressure in high-throughput services.
package efficient_logging

import (
	"go/ast"

	"github.com/alpardfm/cost-efficient-go/types"
)


// detector implements types.Detector for efficient logging anti-patterns.
// It is stateless and safe for concurrent use.
type detector struct {
	rule types.Rule
}

// NewDetector creates a new efficient logging detector.
func NewDetector() types.Detector {
	return &detector{
		rule: types.Rule{
			ID:          "CEG-006",
			Name:        "Efficient Logging Patterns",
			Description: "Detects logging calls that allocate on every invocation (e.g., log.Printf with format strings) instead of using zero-allocation structured loggers. At high throughput (100K+ logs/sec), per-call allocations cause significant GC pressure and CPU overhead.",
			Severity:    types.Minor,
			Category:    types.Memory,
			Suggestion:  "Use a zero-allocation structured logger (zerolog, zap) or implement check-then-log pattern to avoid formatting work when the log level is disabled.",
			ReferenceLinks: []string{
				"https://github.com/alpardfm/cost-efficient-go/tree/main/patterns/efficient-logging",
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

	// Look for calls to log.Printf, fmt.Sprintf used in logging contexts,
	// or other allocating log patterns.
	call, ok := ctx.Node.(*ast.CallExpr)
	if !ok {
		return []types.Finding{}
	}

	if isAllocatingLogCall(call) {
		findings = append(findings, types.Finding{
			RuleID:       d.rule.ID,
			FilePath:     ctx.FilePath,
			Line:         ctx.Line,
			Explanation:  "Allocating log call detected. log.Printf and similar format-based logging allocate on every call due to format string processing and argument boxing. At high throughput this causes significant GC pressure.",
			SuggestedFix: "Replace with a zero-allocation structured logger (zerolog, zap) or use check-then-log pattern to skip formatting when the level is disabled.",
			Severity:     d.rule.Severity,
			Category:     d.rule.Category,
			CodeContext:  ctx.CodeContext,
		})
	}

	return findings
}

// isAllocatingLogCall checks if a call expression is an allocating log call.
func isAllocatingLogCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	// Check for common allocating log methods
	switch sel.Sel.Name {
	case "Printf", "Sprintf", "Fprintf", "Errorf",
		"Infof", "Debugf", "Warnf", "Fatalf":
		return true
	}

	return false
}
