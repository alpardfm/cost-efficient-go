package registry

import (
	"testing"

	"github.com/alpardfm/cost-efficient-go/types"
)

// TestDetectorByID_ValidIDs verifies that DetectorByID returns the correct
// detector for every known rule ID (CEG-001 through CEG-020).
func TestDetectorByID_ValidIDs(t *testing.T) {
	expectedIDs := []struct {
		id       string
		name     string
		category types.Category
		severity types.Severity
	}{
		{"CEG-001", "batch-processing", types.IO, types.Major},
		{"CEG-002", "caching-strategies", types.Memory, types.Major},
		{"CEG-003", "channel-patterns", types.Concurrency, types.Minor},
		{"CEG-004", "connection-pooling", types.IO, types.Critical},
		{"CEG-005", "context-cancellation", types.Concurrency, types.Critical},
		{"CEG-006", "efficient-logging", types.Memory, types.Minor},
		{"CEG-007", "error-handling", types.ErrorHandling, types.Major},
		{"CEG-008", "goroutine-leak", types.Concurrency, types.Critical},
		{"CEG-009", "http-client-optimization", types.IO, types.Major},
		{"CEG-010", "interface-dispatch", types.Memory, types.Minor},
		{"CEG-011", "json-processing", types.Memory, types.Major},
		{"CEG-012", "map-internals", types.Memory, types.Minor},
		{"CEG-013", "profiling-benchmarking", types.Memory, types.Minor},
		{"CEG-014", "query-optimization", types.IO, types.Critical},
		{"CEG-015", "redis-pipeline", types.IO, types.Major},
		{"CEG-016", "slice-performance", types.Memory, types.Major},
		{"CEG-017", "string-building", types.Memory, types.Major},
		{"CEG-018", "struct-alignment", types.Memory, types.Minor},
		{"CEG-019", "sync-pool", types.Memory, types.Major},
		{"CEG-020", "worker-pool", types.Concurrency, types.Major},
	}

	for _, tc := range expectedIDs {
		t.Run(tc.id, func(t *testing.T) {
			d, ok := DetectorByID(tc.id)
			if !ok {
				t.Fatalf("DetectorByID(%q) returned false, expected detector to exist", tc.id)
			}
			if d == nil {
				t.Fatalf("DetectorByID(%q) returned nil detector", tc.id)
			}

			rule := d.Rule()
			if rule.ID != tc.id {
				t.Errorf("expected rule ID %q, got %q", tc.id, rule.ID)
			}
			if rule.Category != tc.category {
				t.Errorf("detector %s: expected category %d, got %d", tc.id, tc.category, rule.Category)
			}
			if rule.Severity != tc.severity {
				t.Errorf("detector %s: expected severity %d, got %d", tc.id, tc.severity, rule.Severity)
			}
		})
	}
}

// TestDetectorByID_UnknownIDs verifies that DetectorByID returns (nil, false)
// for IDs that do not exist in the registry.
func TestDetectorByID_UnknownIDs(t *testing.T) {
	unknownIDs := []string{
		"CEG-000",
		"CEG-021",
		"CEG-999",
		"INVALID",
		"",
		"ceg-001", // case-sensitive check
	}

	for _, id := range unknownIDs {
		t.Run(id, func(t *testing.T) {
			d, ok := DetectorByID(id)
			if ok {
				t.Errorf("DetectorByID(%q) returned true, expected false", id)
			}
			if d != nil {
				t.Errorf("DetectorByID(%q) returned non-nil detector, expected nil", id)
			}
		})
	}
}

// TestDetectorsByCategory_AllCategories verifies that DetectorsByCategory returns
// the correct subset of detectors for each category.
func TestDetectorsByCategory_AllCategories(t *testing.T) {
	tests := []struct {
		category    types.Category
		categoryStr string
		expectedIDs []string
	}{
		{
			category:    types.Memory,
			categoryStr: "Memory",
			expectedIDs: []string{
				"CEG-002", "CEG-006", "CEG-010", "CEG-011",
				"CEG-012", "CEG-013", "CEG-016", "CEG-017",
				"CEG-018", "CEG-019",
			},
		},
		{
			category:    types.Concurrency,
			categoryStr: "Concurrency",
			expectedIDs: []string{
				"CEG-003", "CEG-005", "CEG-008", "CEG-020",
			},
		},
		{
			category:    types.IO,
			categoryStr: "IO",
			expectedIDs: []string{
				"CEG-001", "CEG-004", "CEG-009", "CEG-014", "CEG-015",
			},
		},
		{
			category:    types.ErrorHandling,
			categoryStr: "ErrorHandling",
			expectedIDs: []string{
				"CEG-007",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.categoryStr, func(t *testing.T) {
			result := DetectorsByCategory(tc.category)

			if len(result) != len(tc.expectedIDs) {
				t.Fatalf("DetectorsByCategory(%s): expected %d detectors, got %d",
					tc.categoryStr, len(tc.expectedIDs), len(result))
			}

			// Build a set of returned IDs for verification
			gotIDs := make(map[string]bool)
			for _, d := range result {
				gotIDs[d.Rule().ID] = true
			}

			for _, expectedID := range tc.expectedIDs {
				if !gotIDs[expectedID] {
					t.Errorf("DetectorsByCategory(%s): missing expected detector %s",
						tc.categoryStr, expectedID)
				}
			}
		})
	}
}

// TestDetectorsBySeverity_AllSeverities verifies that DetectorsBySeverity returns
// the correct subset of detectors for each severity level.
func TestDetectorsBySeverity_AllSeverities(t *testing.T) {
	tests := []struct {
		severity    types.Severity
		severityStr string
		expectedIDs []string
	}{
		{
			severity:    types.Minor,
			severityStr: "Minor",
			expectedIDs: []string{
				"CEG-003", "CEG-006", "CEG-010", "CEG-012",
				"CEG-013", "CEG-018",
			},
		},
		{
			severity:    types.Major,
			severityStr: "Major",
			expectedIDs: []string{
				"CEG-001", "CEG-002", "CEG-007", "CEG-009",
				"CEG-011", "CEG-015", "CEG-016", "CEG-017",
				"CEG-019", "CEG-020",
			},
		},
		{
			severity:    types.Critical,
			severityStr: "Critical",
			expectedIDs: []string{
				"CEG-004", "CEG-005", "CEG-008", "CEG-014",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.severityStr, func(t *testing.T) {
			result := DetectorsBySeverity(tc.severity)

			if len(result) != len(tc.expectedIDs) {
				t.Fatalf("DetectorsBySeverity(%s): expected %d detectors, got %d",
					tc.severityStr, len(tc.expectedIDs), len(result))
			}

			// Build a set of returned IDs for verification
			gotIDs := make(map[string]bool)
			for _, d := range result {
				gotIDs[d.Rule().ID] = true
			}

			for _, expectedID := range tc.expectedIDs {
				if !gotIDs[expectedID] {
					t.Errorf("DetectorsBySeverity(%s): missing expected detector %s",
						tc.severityStr, expectedID)
				}
			}
		})
	}
}

// TestAllDetectors_Returns20 verifies that AllDetectors returns exactly 20 detectors
// and that each has a unique, non-empty rule ID.
func TestAllDetectors_Returns20(t *testing.T) {
	all := AllDetectors()

	if len(all) != 20 {
		t.Fatalf("AllDetectors(): expected 20 detectors, got %d", len(all))
	}

	// Verify all IDs are unique and non-empty
	seen := make(map[string]bool)
	for i, d := range all {
		if d == nil {
			t.Fatalf("AllDetectors()[%d] is nil", i)
		}
		id := d.Rule().ID
		if id == "" {
			t.Fatalf("AllDetectors()[%d] has empty rule ID", i)
		}
		if seen[id] {
			t.Fatalf("AllDetectors() contains duplicate ID: %s", id)
		}
		seen[id] = true
	}
}
