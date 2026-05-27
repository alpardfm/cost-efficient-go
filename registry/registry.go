// Package registry aggregates all pattern detectors and provides lookup functions.
// Detectors register themselves via init() functions. After package initialization
// completes, the registry is immutable and all read methods are safe for concurrent use.
package registry

import (
	"sync"

	"github.com/alpardfm/cost-efficient-go/types"
)

var (
	// detectors holds all registered detectors in registration order.
	detectors []types.Detector

	// byID maps rule IDs to their corresponding detectors for O(1) lookup.
	byID = make(map[string]types.Detector)

	// frozen indicates whether registration is closed.
	// Once Freeze is called, Register will panic.
	frozen bool

	// mu protects registration during init time.
	// After Freeze is called, no lock is needed for reads because data is immutable.
	mu sync.Mutex
)

// Register adds a detector to the global registry.
// Called by pattern init() functions. Panics if called after Freeze().
func Register(d types.Detector) {
	mu.Lock()
	defer mu.Unlock()

	if frozen {
		panic("registry: Register called after initialization is complete")
	}

	id := d.Rule().ID
	if _, exists := byID[id]; exists {
		panic("registry: duplicate detector ID: " + id)
	}

	detectors = append(detectors, d)
	byID[id] = d
}

// Freeze marks the registry as immutable. After this call, Register will panic.
// This should be called once all init() functions have completed.
// Typically invoked via an init() in the imports file that runs after all pattern inits.
func Freeze() {
	mu.Lock()
	defer mu.Unlock()
	frozen = true
}

// AllDetectors returns a copy of all registered detectors.
// Safe for concurrent use.
func AllDetectors() []types.Detector {
	result := make([]types.Detector, len(detectors))
	copy(result, detectors)
	return result
}

// DetectorByID returns the detector with the given rule ID.
// Returns (nil, false) if not found. Safe for concurrent use.
func DetectorByID(id string) (types.Detector, bool) {
	d, ok := byID[id]
	return d, ok
}

// DetectorsByCategory returns all detectors matching the given category.
// Safe for concurrent use.
func DetectorsByCategory(cat types.Category) []types.Detector {
	var result []types.Detector
	for _, d := range detectors {
		if d.Rule().Category == cat {
			result = append(result, d)
		}
	}
	return result
}

// DetectorsBySeverity returns all detectors matching the given severity.
// Safe for concurrent use.
func DetectorsBySeverity(sev types.Severity) []types.Detector {
	var result []types.Detector
	for _, d := range detectors {
		if d.Rule().Severity == sev {
			result = append(result, d)
		}
	}
	return result
}
