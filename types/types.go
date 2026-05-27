package types

import "go/ast"

// Severity represents the cost impact level of a detected pattern.
type Severity int

const (
	Minor    Severity = iota // Low cost impact, optimization opportunity
	Major                    // Moderate cost impact, should be addressed
	Critical                 // High cost impact, likely production issue
)

// Category represents the primary resource concern of a pattern.
type Category int

const (
	Memory        Category = iota // Heap allocations, GC pressure, memory layout
	Concurrency                   // Goroutines, synchronization, parallelism
	IO                            // Network, disk, database operations
	ErrorHandling                 // Error propagation, resource cleanup
)

// Rule describes a single cost-efficiency detection rule.
type Rule struct {
	ID             string   // Unique identifier (CEG-001 through CEG-020)
	Name           string   // Human-readable pattern name
	Description    string   // Detailed description of the anti-pattern
	Severity       Severity // Cost impact level
	Category       Category // Primary resource concern
	Suggestion     string   // Template string describing the recommended fix
	ReferenceLinks []string // Links to documentation or pattern README
}

// Finding represents a single detection result.
type Finding struct {
	RuleID       string   // References Rule.ID
	FilePath     string   // Source file where the issue was found
	Line         int      // Line number in the source file
	Explanation  string   // Human-readable description of why this is problematic
	SuggestedFix string   // Concrete code suggestion or pattern template
	Severity     Severity // Inherited from the Rule
	Category     Category // Inherited from the Rule
	CodeContext  string   // Relevant source code snippet
}

// ASTContext carries Go AST information passed to detectors.
// The consuming analyzer is responsible for constructing this.
type ASTContext struct {
	FilePath    string   // Path to the source file being analyzed
	Line        int      // Line number of the node
	Node        ast.Node // The AST node to analyze (may be nil)
	CodeContext string   // Surrounding source code for context
}

// Detector is the interface that all pattern detectors implement.
type Detector interface {
	// Detect analyzes the given AST context and returns findings.
	// Returns an empty slice if no issues are detected.
	// Must not panic, even with nil Node.
	// Must not perform file I/O.
	Detect(ctx ASTContext) []Finding

	// Rule returns the detection rule metadata for this detector.
	Rule() Rule
}
