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
// Only flags connection creation inside function bodies (not package-level init).
func (d *detector) Detect(ctx types.ASTContext) []types.Finding {
	if ctx.Node == nil {
		return []types.Finding{}
	}

	// Only look at function declarations — connection creation at package level
	// (e.g., var db = sql.Open(...) in init or main) is typically one-time setup.
	funcDecl, ok := ctx.Node.(*ast.FuncDecl)
	if !ok {
		return []types.Finding{}
	}

	// Skip init() and main() — these are one-time setup functions
	if funcDecl.Name != nil {
		name := funcDecl.Name.Name
		if name == "init" || name == "main" {
			return []types.Finding{}
		}
	}

	var findings []types.Finding

	// Look for connection creation calls inside the function body
	if funcDecl.Body != nil {
		ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			if isConnectionCreationCall(call) {
				findings = append(findings, types.Finding{
					RuleID:       d.rule.ID,
					FilePath:     ctx.FilePath,
					Line:         ctx.Line,
					Explanation:  "Connection creation detected in a request-handling function. Each call incurs TCP handshake and potentially TLS negotiation overhead. At scale, this wastes network I/O and increases latency significantly.",
					SuggestedFix: "Move connection creation to application startup (init/main) or use a connection pool. For databases, sql.Open returns a pool — call it once and reuse the *sql.DB.",
					Severity:     d.rule.Severity,
					Category:     d.rule.Category,
					CodeContext:  ctx.CodeContext,
					Confidence:   types.ConfidenceMedium,
				})
			}

			return true
		})
	}

	return findings
}

// isConnectionCreationCall checks if a call expression looks like a connection creation operation.
// Whitelists known pooled-by-design libraries (gRPC, pgxpool, go-redis, mongo).
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
	switch pkgName {
	case "grpc": // gRPC uses HTTP/2 multiplexing — inherently pooled
		return false
	case "pgxpool": // pgxpool already is a pool
		return false
	case "redis": // go-redis client manages its own pool
		return false
	case "mongo": // mongo-go-driver manages its own pool
		return false
	case "amqp": // RabbitMQ connections are long-lived by design
		return false
	case "nats": // NATS connections are long-lived
		return false
	}

	// Check for common connection creation patterns
	switch sel.Sel.Name {
	case "Dial", "DialTimeout", "DialContext", "DialTLS":
		return true
	case "Open": // sql.Open returns a pool, but calling it per-request is the anti-pattern
		return true
	case "Connect":
		return true
	}

	return false
}
