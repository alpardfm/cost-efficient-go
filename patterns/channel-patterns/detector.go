// Package channel_patterns detects inefficient channel usage patterns in Go code.
// It identifies unbuffered channels used in high-throughput producer-consumer scenarios
// where buffered channels or mutex-based alternatives would reduce scheduling overhead.
package channel_patterns

import (
	"go/ast"
	"strings"

	"github.com/alpardfm/cost-efficient-go/types"
)

// detector implements types.Detector for channel pattern anti-patterns.
// It is stateless and safe for concurrent use.
type detector struct {
	rule types.Rule
}

// NewDetector creates a new channel patterns detector.
func NewDetector() types.Detector {
	return &detector{
		rule: types.Rule{
			ID:          "CEG-003",
			Name:        "Channel Patterns & Performance Trade-offs",
			Description: "Detects unbuffered channels used in high-throughput producer-consumer scenarios where buffered channels would reduce goroutine scheduling overhead and improve throughput.",
			Severity:    types.Minor,
			Category:    types.Concurrency,
			Suggestion:  "Use buffered channels (size 64-256) for high-throughput producer-consumer patterns. Consider sync.Mutex for simple shared state access without communication semantics.",
			ReferenceLinks: []string{
				"https://github.com/alpardfm/cost-efficient-go/blob/main/patterns/channel-patterns/README.md",
			},
		},
	}
}

// Rule returns the detection rule metadata for this detector.
func (d *detector) Rule() types.Rule {
	return d.rule
}

// Detect analyzes the given AST context for unbuffered channel usage patterns.
// Returns an empty slice if no issues are detected or if Node is nil.
// Does not perform file I/O. Does not panic.
func (d *detector) Detect(ctx types.ASTContext) []types.Finding {
	if ctx.Node == nil {
		return []types.Finding{}
	}

	var findings []types.Finding

	// Look for make(chan T) calls without a buffer size argument (unbuffered channels)
	callExpr, ok := ctx.Node.(*ast.CallExpr)
	if !ok {
		return []types.Finding{}
	}

	// Check if this is a make() call
	ident, ok := callExpr.Fun.(*ast.Ident)
	if !ok || ident.Name != "make" {
		return []types.Finding{}
	}

	// Check if the first argument is a channel type
	if len(callExpr.Args) < 1 {
		return []types.Finding{}
	}

	_, isChan := callExpr.Args[0].(*ast.ChanType)
	if !isChan {
		return []types.Finding{}
	}

	// Unbuffered channel: make(chan T) with no second argument
	if len(callExpr.Args) == 1 {
		// Skip signal channels — these are intentionally unbuffered
		if isSignalChannel(callExpr.Args[0]) {
			return []types.Finding{}
		}

		codeContext := ctx.CodeContext
		if codeContext == "" {
			codeContext = "make(chan T)"
		}

		findings = append(findings, types.Finding{
			RuleID:       d.rule.ID,
			FilePath:     ctx.FilePath,
			Line:         ctx.Line,
			Explanation:  "Unbuffered channel detected. In high-throughput producer-consumer scenarios, unbuffered channels force goroutine scheduling on every send/receive, causing ~200ns overhead per operation.",
			SuggestedFix: "Use a buffered channel: make(chan T, 100). Buffer size 64-256 is optimal for most high-throughput workloads.",
			Severity:     d.rule.Severity,
			Category:     d.rule.Category,
			CodeContext:  strings.TrimSpace(codeContext),
			Confidence:   types.ConfidenceMedium,
		})
	}

	return findings
}

// isSignalChannel checks if a channel type is likely a signal channel
// that should be unbuffered by design.
// Signal channels: chan struct{}, chan bool, chan error, chan os.Signal
func isSignalChannel(expr ast.Expr) bool {
	chanType, ok := expr.(*ast.ChanType)
	if !ok {
		return false
	}

	switch t := chanType.Value.(type) {
	case *ast.StructType:
		// chan struct{} — classic done/signal channel
		if t.Fields == nil || len(t.Fields.List) == 0 {
			return true
		}
	case *ast.Ident:
		// chan bool, chan error — common signal types
		switch t.Name {
		case "bool", "error":
			return true
		}
	case *ast.SelectorExpr:
		// chan os.Signal
		if ident, ok := t.X.(*ast.Ident); ok {
			if ident.Name == "os" && t.Sel.Name == "Signal" {
				return true
			}
		}
	}

	return false
}
