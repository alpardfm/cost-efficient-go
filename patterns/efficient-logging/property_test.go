package main

import (
	"io"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: expand-cost-efficient-go, Property 15: Disabled log level produces zero string formatting work (0 allocations)
func TestProperty_DisabledLogLevelZeroAllocations(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("disabled log level produces zero allocations", prop.ForAll(
		func(userID int) bool {
			// Create logger with INFO level (DEBUG is disabled)
			logger := NewZeroAllocLogger(io.Discard, LevelInfo, 512, 16)

			// Measure allocations for disabled level (DEBUG < INFO)
			allocs := testing.AllocsPerRun(100, func() {
				CheckThenLog(logger, LevelDebug, "debug_message", userID, "trace_action", 1.23)
			})

			// Should be zero allocations when level is disabled
			return allocs == 0
		},
		gen.IntRange(0, 1000000),
	))

	properties.TestingRun(t)
}
