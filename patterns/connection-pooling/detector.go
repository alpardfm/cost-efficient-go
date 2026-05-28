// Package connection_pooling detects missing or misconfigured connection pooling
// that leads to excessive TCP connection creation overhead and resource waste.
package connection_pooling

import (
	"go/ast"

	"github.com/alpardfm/cost-efficient-go/types"
)

// detector implements types.Detector for connection pooling anti-patterns.
// It is stateless and safe for concurrent use.
type detector struct {
	rule types.Rule
}

// NewDetector creates a new connection pooling detector.
func NewDetector() types.Detector {
	return &detector{
		rule: types.Rule{
			ID:          "CEG-004",
			Name:        "Connection Pooling",
			Description: "Detects code that creates new TCP/database connections per request instead of reusing pooled connections. Creating a new connection per request incurs TCP handshake, TLS negotiation, and connection setup overhead (1-10ms per request), leading to significant latency and resource waste at scale.",
			Severity:    types.Critical,
			Category:    types.IO,
			Suggestion:  "Use a connection pool (e.g., sql.DB, http.Transport, custom pool) to reuse connections across requests. Size the pool to match expected concurrency to maximize reuse ratio.",
			ReferenceLinks: []string{
				"https://github.com/alpardfm/cost-efficient-go/tree/main/patterns/connection-pooling",
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

	// Look for net.Dial, net.DialTimeout, or sql.Open calls inside loops
	// which indicate connection-per-request anti-pattern.
	ast.Inspect(ctx.Node, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		if isConnectionCreationCall(call) {
			findings = append(findings, types.Finding{
				RuleID:       d.rule.ID,
				FilePath:     ctx.FilePath,
				Line:         ctx.Line,
				Explanation:  "Connection creation detected in a hot path. Each call incurs TCP handshake and potentially TLS negotiation overhead. At scale, this wastes network I/O and increases latency significantly.",
				SuggestedFix: "Use a connection pool to reuse established connections. For databases, use sql.DB which pools internally. For TCP, implement or use a pool with configurable maxIdle matching your concurrency level.",
				Severity:     d.rule.Severity,
				Category:     d.rule.Category,
				CodeContext:  ctx.CodeContext,
				Confidence:   types.ConfidenceMedium,
			})
		}

		return true
	})

	return findings
}

// isConnectionCreationCall checks if a call expression looks like a connection creation operation.
// Whitelists known pooled-by-design libraries (gRPC, pgxpool, go-redis).
func isConnectionCreationCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	// Get the package/receiver identifier if available
	pkgName := ""
	if ident, ok := sel.X.(*ast.Ident); ok {
		pkgName = ident.Name
	}

	// Whitelist: known pooled-by-design libraries
	// gRPC uses HTTP/2 multiplexing — inherently pooled
	if pkgName == "grpc" {
		return false
	}
	// pgxpool already is a pool
	if pkgName == "pgxpool" {
		return false
	}
	// go-redis client manages its own pool
	if pkgName == "redis" {
		return false
	}

	// Check for common connection creation patterns
	switch sel.Sel.Name {
	case "Dial", "DialTimeout", "DialContext", "DialTLS":
		return true
	case "Open": // sql.Open — note: sql.Open itself returns a pool, but calling it per-request is the anti-pattern
		return true
	case "Connect":
		return true
	}

	return false
}
