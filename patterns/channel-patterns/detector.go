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
		})
	}

	return findings
}

