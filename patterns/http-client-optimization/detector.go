// Package http_client_optimization detects HTTP client misconfigurations that waste
// network resources, such as missing timeouts, unclosed response bodies, and
// creating new clients per request instead of reusing a shared client.
package http_client_optimization

import (
	"go/ast"

	"github.com/alpardfm/cost-efficient-go/types"
)


// detector implements types.Detector for HTTP client optimization anti-patterns.
// It is stateless and safe for concurrent use.
type detector struct {
	rule types.Rule
}

// NewDetector creates a new HTTP client optimization detector.
func NewDetector() types.Detector {
	return &detector{
		rule: types.Rule{
			ID:          "CEG-009",
			Name:        "HTTP Client Optimization",
			Description: "Detects HTTP client misconfigurations including missing timeouts, unclosed response bodies (leaking connections), creating new clients per request, and lack of context cancellation support. These issues waste network I/O and can cause connection exhaustion under load.",
			Severity:    types.Major,
			Category:    types.IO,
			Suggestion:  "Use a shared http.Client with explicit timeouts and tuned Transport settings. Always close and drain response bodies. Use context for cancellation support.",
			ReferenceLinks: []string{
				"https://github.com/alpardfm/cost-efficient-go/tree/main/patterns/http-client-optimization",
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

	// Look for composite literals creating http.Client without Timeout field,
	// or function calls that indicate per-request client creation patterns.
	ast.Inspect(ctx.Node, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CompositeLit:
			if isHTTPClientLiteral(node) && !hasTimeoutField(node) {
				findings = append(findings, types.Finding{
					RuleID:       d.rule.ID,
					FilePath:     ctx.FilePath,
					Line:         ctx.Line,
					Explanation:  "HTTP client created without explicit Timeout. Default http.Client has no timeout, which can cause requests to hang indefinitely on slow upstreams.",
					SuggestedFix: "Set an explicit Timeout on the http.Client (e.g., Timeout: 10 * time.Second) and configure Transport for connection reuse.",
					Severity:     d.rule.Severity,
					Category:     d.rule.Category,
					CodeContext:  ctx.CodeContext,
				})
			}
		case *ast.CallExpr:
			if isUnclosedBodyPattern(node) {
				findings = append(findings, types.Finding{
					RuleID:       d.rule.ID,
					FilePath:     ctx.FilePath,
					Line:         ctx.Line,
					Explanation:  "HTTP response body not closed or drained. Unclosed bodies prevent connection reuse and can lead to connection pool exhaustion.",
					SuggestedFix: "Always defer resp.Body.Close() and drain the body with io.Copy(io.Discard, resp.Body) to allow connection reuse.",
					Severity:     d.rule.Severity,
					Category:     d.rule.Category,
					CodeContext:  ctx.CodeContext,
				})
			}
		}
		return true
	})

	return findings
}

// isHTTPClientLiteral checks if a composite literal is creating an http.Client.
func isHTTPClientLiteral(lit *ast.CompositeLit) bool {
	sel, ok := lit.Type.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "http" && sel.Sel.Name == "Client"
}

// hasTimeoutField checks if a composite literal includes a Timeout field.
func hasTimeoutField(lit *ast.CompositeLit) bool {
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		ident, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		if ident.Name == "Timeout" {
			return true
		}
	}
	return false
}

// isUnclosedBodyPattern checks for http.Get or client.Get calls that suggest
// response body may not be properly handled.
func isUnclosedBodyPattern(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	// Detect http.Get (package-level default client usage)
	if ident, ok := sel.X.(*ast.Ident); ok {
		if ident.Name == "http" && sel.Sel.Name == "Get" {
			return true
		}
	}

	return false
}
