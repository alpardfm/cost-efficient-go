package main

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: expand-cost-efficient-go, Property 7: All error creation methods satisfy the error interface and contain the message
func TestProperty_AllErrorMethodsSatisfyInterface(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("all error methods implement error interface and contain message", prop.ForAll(
		func(_ int) bool {
			// ErrorsNew
			err1 := ErrorsNew()
			if err1 == nil || err1.Error() == "" {
				return false
			}

			// FmtErrorf
			err2 := FmtErrorf()
			if err2 == nil || err2.Error() == "" {
				return false
			}

			// SentinelError
			var err3 error = SentinelError
			if err3 == nil || err3.Error() == "" {
				return false
			}

			// PreallocatedError
			var err4 error = PreallocatedError
			if err4 == nil || err4.Error() == "" {
				return false
			}

			// All should contain "operation failed"
			contains := func(e error, substr string) bool {
				msg := e.Error()
				for i := 0; i <= len(msg)-len(substr); i++ {
					if msg[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}

			return contains(err1, "operation failed") &&
				contains(err2, "operation failed") &&
				contains(err3, "operation failed") &&
				contains(err4, "operation failed")
		},
		gen.IntRange(0, 100),
	))

	properties.TestingRun(t)
}

// Feature: expand-cost-efficient-go, Property 8: Sentinel errors produce zero heap allocations (testing.AllocsPerRun = 0)
func TestProperty_SentinelErrorsZeroAlloc(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("sentinel error produces zero allocations", prop.ForAll(
		func(iterations int) bool {
			if iterations < 1 {
				iterations = 1
			}

			// Measure sentinel error allocations
			sentinelAllocs := testing.AllocsPerRun(iterations, func() {
				globalErr = SentinelError
			})

			// Measure preallocated error allocations
			preallocAllocs := testing.AllocsPerRun(iterations, func() {
				globalErr = PreallocatedError
			})

			// Both should be zero
			return sentinelAllocs == 0 && preallocAllocs == 0
		},
		gen.IntRange(1, 100),
	))

	properties.TestingRun(t)
}
