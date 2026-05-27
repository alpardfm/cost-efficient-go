package channel_patterns

import (
	"sort"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: expand-cost-efficient-go, Property 14: Fan-out/fan-in produces same aggregate result as sequential (same set of outputs)
func TestProperty_FanOutFanInSameAsSequential(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("fan-out/fan-in produces same set of results as sequential", prop.ForAll(
		func(items []int) bool {
			if len(items) == 0 {
				seqResult := FanOutFanInSequential(items)
				parResult := FanOutFanIn(items, 4)
				return len(seqResult) == 0 && len(parResult) == 0
			}

			// Sequential processing
			seqResult := FanOutFanInSequential(items)

			// Parallel processing with fan-out/fan-in
			parResult := FanOutFanIn(items, 4)

			// Both should have the same number of results
			if len(seqResult) != len(parResult) {
				return false
			}

			// Sort both results to compare (order may differ in parallel)
			sort.Ints(seqResult)
			sort.Ints(parResult)

			for i := range seqResult {
				if seqResult[i] != parResult[i] {
					return false
				}
			}
			return true
		},
		gen.SliceOfN(50, gen.IntRange(0, 10000)),
	))

	properties.TestingRun(t)
}
