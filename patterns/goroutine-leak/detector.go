// Package goroutine_leak detects goroutine leak patterns where goroutines are spawned
// without proper exit paths, leading to unbounded memory growth and eventual OOM kills.
package goroutine_leak

import (
	"go/ast"

	"github.com/alpardfm/cost-efficient-go/types"
)


// detector implements types.Detector for goroutine leak anti-patterns.
// It is stateless and safe for concurrent use.
type detector struct {
	rule types.Rule
}

// NewDetector creates a new goroutine leak detector.
func NewDetector() types.Detector {
	return &detector{
		rule: types.Rule{
			ID:          "CEG-008",
			Name:        "Goroutine Leak Detection",
			Description: "Detects goroutines spawned without exit paths (no context cancellation, no done channel, no timeout). Each leaked goroutine holds 2-8KB of stack memory indefinitely. At 1 leak/second, this accumulates 172-691 MB/day of unrecoverable memory, eventually causing OOM kills.",
			Severity:    types.Critical,
			Category:    types.Concurrency,
			Suggestion:  "Ensure every goroutine has an exit path via context.WithCancel, done channels, or timeouts. Use select with ctx.Done() to respond to cancellation signals. Track goroutine lifecycle with sync.WaitGroup.",
			ReferenceLinks: []string{
				"https://github.com/alpardfm/cost-efficient-go/tree/main/patterns/goroutine-leak",
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

	// Look for go statements (goroutine launches) that lack proper exit paths.
	// Specifically, detect goroutines with channel receives that have no
	// corresponding select with context cancellation or timeout.
	goStmt, ok := ctx.Node.(*ast.GoStmt)
	if !ok {
		return []types.Finding{}
	}

	// Analyze the goroutine's function body for blocking operations without exit paths
	funcLit, ok := goStmt.Call.Fun.(*ast.FuncLit)
	if !ok {
		return []types.Finding{}
	}

	if hasBlockingWithoutExit(funcLit.Body) {
		findings = append(findings, types.Finding{
			RuleID:       d.rule.ID,
			FilePath:     ctx.FilePath,
			Line:         ctx.Line,
			Explanation:  "Goroutine contains a blocking operation (channel receive) without a select statement or context cancellation check. This goroutine may block indefinitely, leaking memory.",
			SuggestedFix: "Wrap blocking operations in a select statement with a ctx.Done() case or a done channel to ensure the goroutine can exit when no longer needed.",
			Severity:     d.rule.Severity,
			Category:     d.rule.Category,
			CodeContext:  ctx.CodeContext,
		})
	}

	return findings
}

// hasBlockingWithoutExit checks if a function body contains channel receives
// without corresponding select statements that provide exit paths.
func hasBlockingWithoutExit(body *ast.BlockStmt) bool {
	if body == nil {
		return false
	}

	hasChannelRecv := false
	hasSelect := false

	ast.Inspect(body, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.UnaryExpr:
			// Check for <-ch (channel receive as expression)
			unary, _ := n.(*ast.UnaryExpr)
			if unary != nil && unary.Op.String() == "<-" {
				hasChannelRecv = true
			}
		case *ast.SelectStmt:
			hasSelect = true
		}
		return true
	})

	// A goroutine with channel receives but no select is suspicious
	return hasChannelRecv && !hasSelect
}
