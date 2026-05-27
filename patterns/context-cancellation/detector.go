// Package context_cancellation detects missing context cancellation propagation
// in multi-step call chains, which wastes CPU on abandoned work when clients disconnect.
package context_cancellation

import (
	"go/ast"

	"github.com/alpardfm/cost-efficient-go/types"
)


// detector implements types.Detector for context cancellation anti-patterns.
// It is stateless and safe for concurrent use.
type detector struct {
	rule types.Rule
}

// NewDetector creates a new context cancellation detector.
func NewDetector() types.Detector {
	return &detector{
		rule: types.Rule{
			ID:          "CEG-005",
			Name:        "Context Cancellation & Resource Cleanup",
			Description: "Detects multi-step call chains that ignore context cancellation, causing services to burn CPU on work that nobody will consume after client disconnects or timeouts fire. Common anti-patterns include using context.Background() in goroutines and blocking without ctx.Done() checks.",
			Severity:    types.Critical,
			Category:    types.Concurrency,
			Suggestion:  "Propagate the parent context through every step and check ctx.Done() before proceeding. Use select with ctx.Done() case for blocking operations. Never use context.Background() in goroutines that should inherit parent cancellation.",
			ReferenceLinks: []string{
				"https://github.com/alpardfm/cost-efficient-go/tree/main/patterns/context-cancellation",
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

	// Look for goroutine launches (go statements) that use context.Background()
	// instead of propagating the parent context.
	goStmt, ok := ctx.Node.(*ast.GoStmt)
	if !ok {
		// Also check for call expressions that ignore context (e.g., time.Sleep in a loop)
		return d.checkBlockingWithoutContext(ctx)
	}

	// Inspect the goroutine body for context.Background() usage
	ast.Inspect(goStmt.Call, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		if isContextBackgroundCall(call) {
			findings = append(findings, types.Finding{
				RuleID:       d.rule.ID,
				FilePath:     ctx.FilePath,
				Line:         ctx.Line,
				Explanation:  "context.Background() used inside a goroutine loses parent cancellation signal. When the parent context is cancelled, this goroutine becomes orphaned and continues burning resources.",
				SuggestedFix: "Pass the parent context to the goroutine instead of using context.Background(). This ensures cancellation propagates correctly.",
				Severity:     d.rule.Severity,
				Category:     d.rule.Category,
				CodeContext:  ctx.CodeContext,
			})
		}

		return true
	})

	return findings
}

// checkBlockingWithoutContext looks for blocking operations (like time.Sleep)
// inside function bodies that accept a context parameter but don't use it.
func (d *detector) checkBlockingWithoutContext(ctx types.ASTContext) []types.Finding {
	var findings []types.Finding

	funcDecl, ok := ctx.Node.(*ast.FuncDecl)
	if !ok {
		return findings
	}

	// Check if function accepts a context parameter
	if !hasContextParam(funcDecl) {
		return findings
	}

	// Look for time.Sleep calls (blocking without context check)
	ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		if isTimeSleepCall(call) {
			findings = append(findings, types.Finding{
				RuleID:       d.rule.ID,
				FilePath:     ctx.FilePath,
				Line:         ctx.Line,
				Explanation:  "time.Sleep used in a function that accepts context. This blocks without checking ctx.Done(), preventing timely cancellation when the context is cancelled.",
				SuggestedFix: "Replace time.Sleep with a select statement that includes a ctx.Done() case, allowing the function to exit early on cancellation.",
				Severity:     d.rule.Severity,
				Category:     d.rule.Category,
				CodeContext:  ctx.CodeContext,
			})
		}

		return true
	})

	return findings
}

// isContextBackgroundCall checks if a call expression is context.Background().
func isContextBackgroundCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	return ident.Name == "context" && sel.Sel.Name == "Background"
}

// isTimeSleepCall checks if a call expression is time.Sleep().
func isTimeSleepCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	return ident.Name == "time" && sel.Sel.Name == "Sleep"
}

// hasContextParam checks if a function declaration has a context.Context parameter.
func hasContextParam(funcDecl *ast.FuncDecl) bool {
	if funcDecl.Type == nil || funcDecl.Type.Params == nil {
		return false
	}

	for _, field := range funcDecl.Type.Params.List {
		if sel, ok := field.Type.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok {
				if ident.Name == "context" && sel.Sel.Name == "Context" {
					return true
				}
			}
		}
	}

	return false
}
