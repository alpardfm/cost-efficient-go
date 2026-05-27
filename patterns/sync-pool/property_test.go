package main

import (
	"runtime"
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

	properties.Property("pooled allocation has fewer or equal allocs than naive", prop.ForAll(
		func(size int) bool {
			pool := NewBufferPool(size)

			// Warm the pool with one cycle
			buf := pool.Get()
			pool.Put(buf)

			// Measure naive allocation
			runtime.GC()
			var memBefore, memAfter runtime.MemStats
			runtime.ReadMemStats(&memBefore)
			for i := 0; i < 10; i++ {
				globalSink = NaiveBufferAlloc(size)
			}
			runtime.ReadMemStats(&memAfter)
			naiveAllocs := memAfter.Mallocs - memBefore.Mallocs

			// Measure pooled allocation
			runtime.GC()
			runtime.ReadMemStats(&memBefore)
			for i := 0; i < 10; i++ {
				globalSink = PooledBufferAlloc(pool)
			}
			runtime.ReadMemStats(&memAfter)
			pooledAllocs := memAfter.Mallocs - memBefore.Mallocs

			return pooledAllocs <= naiveAllocs
		},
		gen.IntRange(64, 65536),
	))

	properties.TestingRun(t)
}
