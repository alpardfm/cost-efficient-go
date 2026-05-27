package sync_pool

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: expand-cost-efficient-go, Property 1: Pool reuse never allocates more than fresh allocation
func TestProperty_PoolReuseNeverAllocatesMoreThanFresh(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Note (Req 9.3): Switched from runtime.MemStats to testing.AllocsPerRun
	// because MemStats is unreliable under the race detector and GC timing
	// variations. The property being tested is the same: pooled allocation
	// should use fewer or equal heap allocations than naive allocation.
	properties.Property("pooled allocation has fewer or equal allocs than naive", prop.ForAll(
		func(size int) bool {
			pool := NewBufferPool(size)

			// Warm the pool with one cycle
			buf := pool.Get()
			pool.Put(buf)

			// Measure naive allocation count
			naiveAllocs := testing.AllocsPerRun(10, func() {
				GlobalSink = NaiveBufferAlloc(size)
			})

			// Measure pooled allocation count
			pooledAllocs := testing.AllocsPerRun(10, func() {
				GlobalSink = PooledBufferAlloc(pool)
			})

			return pooledAllocs <= naiveAllocs
		},
		gen.IntRange(64, 65536),
	))

	properties.TestingRun(t)
}
