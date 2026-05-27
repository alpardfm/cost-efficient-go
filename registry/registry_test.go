package registry

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/alpardfm/cost-efficient-go/types"
)

// mockDetector is a test helper implementing types.Detector.
type mockDetector struct {
	rule types.Rule
}

func (m *mockDetector) Detect(_ types.ASTContext) []types.Finding {
	return nil
}

func (m *mockDetector) Rule() types.Rule {
	return m.rule
}

func newMockDetector(id string, cat types.Category, sev types.Severity) *mockDetector {
	return &mockDetector{
		rule: types.Rule{
			ID:       id,
			Name:     "Test " + id,
			Category: cat,
			Severity: sev,
		},
	}
}

// saveAndResetRegistry saves the current registry state and resets it for isolated testing.
// Returns a restore function that should be deferred.
func saveAndResetRegistry() func() {
	mu.Lock()
	savedDetectors := detectors
	savedByID := byID
	savedFrozen := frozen

	detectors = nil
	byID = make(map[string]types.Detector)
	frozen = false
	mu.Unlock()

	return func() {
		mu.Lock()
		detectors = savedDetectors
		byID = savedByID
		frozen = savedFrozen
		mu.Unlock()
	}
}

func TestRegister(t *testing.T) {
	restore := saveAndResetRegistry()
	defer restore()

	d := newMockDetector("CEG-001", types.Memory, types.Major)
	Register(d)

	all := AllDetectors()
	if len(all) != 1 {
		t.Fatalf("expected 1 detector, got %d", len(all))
	}
	if all[0].Rule().ID != "CEG-001" {
		t.Fatalf("expected ID CEG-001, got %s", all[0].Rule().ID)
	}
}

func TestRegisterPanicsAfterFreeze(t *testing.T) {
	restore := saveAndResetRegistry()
	defer restore()

	Freeze()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when registering after freeze")
		}
	}()

	Register(newMockDetector("CEG-999", types.IO, types.Minor))
}

func TestRegisterPanicsOnDuplicateID(t *testing.T) {
	restore := saveAndResetRegistry()
	defer restore()

	Register(newMockDetector("CEG-001", types.Memory, types.Major))

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic on duplicate ID registration")
		}
	}()

	Register(newMockDetector("CEG-001", types.IO, types.Minor))
}

func TestAllDetectorsReturnsCopy(t *testing.T) {
	restore := saveAndResetRegistry()
	defer restore()

	Register(newMockDetector("CEG-001", types.Memory, types.Major))
	Register(newMockDetector("CEG-002", types.IO, types.Critical))

	all1 := AllDetectors()
	all2 := AllDetectors()

	if len(all1) != 2 || len(all2) != 2 {
		t.Fatalf("expected 2 detectors in each copy")
	}

	// Mutating one copy should not affect the other
	all1[0] = nil
	if all2[0] == nil {
		t.Fatal("AllDetectors did not return independent copies")
	}
}

func TestDetectorByID(t *testing.T) {
	restore := saveAndResetRegistry()
	defer restore()

	d := newMockDetector("CEG-005", types.Concurrency, types.Critical)
	Register(d)

	found, ok := DetectorByID("CEG-005")
	if !ok {
		t.Fatal("expected to find detector CEG-005")
	}
	if found.Rule().ID != "CEG-005" {
		t.Fatalf("expected ID CEG-005, got %s", found.Rule().ID)
	}

	_, ok = DetectorByID("CEG-999")
	if ok {
		t.Fatal("expected not to find detector CEG-999")
	}
}

func TestDetectorsByCategory(t *testing.T) {
	restore := saveAndResetRegistry()
	defer restore()

	Register(newMockDetector("CEG-001", types.Memory, types.Major))
	Register(newMockDetector("CEG-002", types.IO, types.Critical))
	Register(newMockDetector("CEG-003", types.Memory, types.Minor))
	Register(newMockDetector("CEG-004", types.Concurrency, types.Major))

	memDetectors := DetectorsByCategory(types.Memory)
	if len(memDetectors) != 2 {
		t.Fatalf("expected 2 Memory detectors, got %d", len(memDetectors))
	}

	ioDetectors := DetectorsByCategory(types.IO)
	if len(ioDetectors) != 1 {
		t.Fatalf("expected 1 IO detector, got %d", len(ioDetectors))
	}

	errDetectors := DetectorsByCategory(types.ErrorHandling)
	if len(errDetectors) != 0 {
		t.Fatalf("expected 0 ErrorHandling detectors, got %d", len(errDetectors))
	}
}

func TestDetectorsBySeverity(t *testing.T) {
	restore := saveAndResetRegistry()
	defer restore()

	Register(newMockDetector("CEG-001", types.Memory, types.Major))
	Register(newMockDetector("CEG-002", types.IO, types.Critical))
	Register(newMockDetector("CEG-003", types.Memory, types.Minor))
	Register(newMockDetector("CEG-004", types.Concurrency, types.Major))

	majorDetectors := DetectorsBySeverity(types.Major)
	if len(majorDetectors) != 2 {
		t.Fatalf("expected 2 Major detectors, got %d", len(majorDetectors))
	}

	criticalDetectors := DetectorsBySeverity(types.Critical)
	if len(criticalDetectors) != 1 {
		t.Fatalf("expected 1 Critical detector, got %d", len(criticalDetectors))
	}

	minorDetectors := DetectorsBySeverity(types.Minor)
	if len(minorDetectors) != 1 {
		t.Fatalf("expected 1 Minor detector, got %d", len(minorDetectors))
	}
}

// TestAllDetectorsRegistered is a build-time validation test that verifies
// all 20 pattern detectors are properly registered in the registry.
// This test validates Requirements 4.7, 7.1, 7.2, 7.3.
func TestAllDetectorsRegistered(t *testing.T) {
	// Expected pattern directories under patterns/
	expectedPatterns := []string{
		"batch-processing",
		"caching-strategies",
		"channel-patterns",
		"connection-pooling",
		"context-cancellation",
		"efficient-logging",
		"error-handling",
		"goroutine-leak",
		"http-client-optimization",
		"interface-dispatch",
		"json-processing",
		"map-internals",
		"profiling-benchmarking",
		"query-optimization",
		"redis-pipeline",
		"slice-performance",
		"string-building",
		"struct-alignment",
		"sync-pool",
		"worker-pool",
	}

	all := AllDetectors()

	// Verify exactly 20 detectors are registered
	if len(all) != 20 {
		t.Fatalf("expected exactly 20 registered detectors, got %d", len(all))
	}

	// Verify each detector has a unique ID matching CEG-XXX format
	idPattern := regexp.MustCompile(`^CEG-\d{3}$`)
	seenIDs := make(map[string]bool)

	for _, d := range all {
		id := d.Rule().ID

		// Verify ID format
		if !idPattern.MatchString(id) {
			t.Errorf("detector ID %q does not match expected format ^CEG-\\d{3}$", id)
		}

		// Verify no duplicate IDs
		if seenIDs[id] {
			t.Errorf("duplicate detector ID found: %s", id)
		}
		seenIDs[id] = true
	}

	// Scan patterns/ directories to ensure all have corresponding registered detectors.
	// Find the project root by looking for go.mod relative to this test file.
	projectRoot := findProjectRoot(t)
	patternsDir := filepath.Join(projectRoot, "patterns")

	entries, err := os.ReadDir(patternsDir)
	if err != nil {
		t.Fatalf("failed to read patterns directory: %v", err)
	}

	// Collect actual pattern directories
	var actualPatternDirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			actualPatternDirs = append(actualPatternDirs, entry.Name())
		}
	}

	// Verify all expected patterns exist as directories
	for _, expected := range expectedPatterns {
		found := false
		for _, actual := range actualPatternDirs {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected pattern directory %q not found in patterns/", expected)
		}
	}

	// Verify all pattern directories have a corresponding registered detector.
	// Each pattern directory should have a detector registered (we verify count matches).
	if len(actualPatternDirs) != len(all) {
		t.Errorf("mismatch: %d pattern directories but %d registered detectors",
			len(actualPatternDirs), len(all))
	}

	// Verify that Register() panics if called after initialization (frozen state).
	// First freeze the registry, then attempt to register.
	Freeze()
	defer func() {
		// Unfreeze for other tests that may run after this.
		mu.Lock()
		frozen = false
		mu.Unlock()
	}()

	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Error("expected Register() to panic after initialization (Freeze), but it did not")
			}
		}()
		Register(&mockDetector{
			rule: types.Rule{
				ID:   "CEG-999",
				Name: "Test Post-Init Registration",
			},
		})
	}()
}

// findProjectRoot locates the project root by walking up from the current
// working directory looking for go.mod.
func findProjectRoot(t *testing.T) string {
	t.Helper()

	// Start from the current working directory
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	// Walk up looking for go.mod
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root without finding go.mod
			t.Fatal("could not find project root (no go.mod found in parent directories)")
		}
		dir = parent
	}
}
