package redis_pipeline

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// ============================================================
// Benchmarks: Redis Pipeline & Connection Efficiency
// ============================================================
// Requirements 10.1: Compare individual GET/SET vs pipeline for 10-100 ops
// Requirements 10.3: Measure pool size impact on throughput and latency
// ============================================================

// Global vars to prevent compiler optimization
var (
	benchVal   string
	benchFound bool
	benchDur   time.Duration
	benchOps   int
)

// benchLatency is the simulated network latency for benchmarks.
// 100μs keeps benchmarks fast while still showing the difference
// between individual round-trips and pipelined batches.
const benchLatency = 100 * time.Microsecond

// --- Benchmark: Individual vs Pipeline Operations ---
// Measures ns/op for individual GET/SET vs pipeline at 10, 50, 100 operations.
// Individual: 2*N round-trips (N SETs + N GETs)
// Pipeline: 2 round-trips (1 batch SET + 1 batch GET)

func BenchmarkIndividualOps_10(b *testing.B) {
	mock := NewRedisMock(benchLatency, 10)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchDur = IndividualOps(mock, 10)
	}
}

func BenchmarkPipelineOps_10(b *testing.B) {
	mock := NewRedisMock(benchLatency, 10)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchDur = PipelineOps(mock, 10)
	}
}

func BenchmarkIndividualOps_50(b *testing.B) {
	mock := NewRedisMock(benchLatency, 10)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchDur = IndividualOps(mock, 50)
	}
}

func BenchmarkPipelineOps_50(b *testing.B) {
	mock := NewRedisMock(benchLatency, 10)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchDur = PipelineOps(mock, 50)
	}
}

func BenchmarkIndividualOps_100(b *testing.B) {
	mock := NewRedisMock(benchLatency, 10)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchDur = IndividualOps(mock, 100)
	}
}

func BenchmarkPipelineOps_100(b *testing.B) {
	mock := NewRedisMock(benchLatency, 10)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchDur = PipelineOps(mock, 100)
	}
}

// --- Sub-benchmarks: Individual vs Pipeline (table-driven) ---
// Provides a cleaner comparison view with sub-benchmarks.

func BenchmarkIndividualVsPipeline(b *testing.B) {
	sizes := []int{10, 50, 100}

	for _, numOps := range sizes {
		b.Run(fmt.Sprintf("Individual_%d", numOps), func(b *testing.B) {
			mock := NewRedisMock(benchLatency, 10)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchDur = IndividualOps(mock, numOps)
			}
		})

		b.Run(fmt.Sprintf("Pipeline_%d", numOps), func(b *testing.B) {
			mock := NewRedisMock(benchLatency, 10)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchDur = PipelineOps(mock, numOps)
			}
		})
	}
}

// --- Benchmark: Pool Size Impact on Concurrent Access ---
// Measures throughput with varying pool sizes (1, 5, 10, 20, 50)
// under concurrent access from 20 workers.

func BenchmarkPoolSize(b *testing.B) {
	poolSizes := []int{1, 5, 10, 20, 50}
	const concurrency = 20
	const opsPerWorker = 20

	for _, poolSize := range poolSizes {
		b.Run(fmt.Sprintf("pool=%d", poolSize), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchDur, benchOps = PoolSizeDemo(poolSize, concurrency, opsPerWorker)
			}
		})
	}
}

// BenchmarkPoolSizeThroughput measures raw throughput (ops/sec) at different
// pool sizes with concurrent workers competing for connections.
func BenchmarkPoolSizeThroughput(b *testing.B) {
	poolSizes := []int{1, 5, 10, 20, 50}
	const concurrency = 20
	const opsPerWorker = 10

	for _, poolSize := range poolSizes {
		b.Run(fmt.Sprintf("pool=%d_workers=%d", poolSize, concurrency), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				mock := NewRedisMock(benchLatency, poolSize)

				// Semaphore to simulate pool size limiting concurrent connections
				pool := make(chan struct{}, poolSize)
				for j := 0; j < poolSize; j++ {
					pool <- struct{}{}
				}

				var wg sync.WaitGroup
				for w := 0; w < concurrency; w++ {
					wg.Add(1)
					go func(workerID int) {
						defer wg.Done()
						for op := 0; op < opsPerWorker; op++ {
							<-pool
							key := fmt.Sprintf("bench:w%d:k%d", workerID, op)
							mock.Set(key, "data")
							benchVal, benchFound = mock.Get(key)
							pool <- struct{}{}
						}
					}(w)
				}
				wg.Wait()
			}
		})
	}
}

// --- Benchmark: Lua Script vs Individual Operations ---
// Compares individual SET-if-not-exists (2 round-trips each) vs
// Lua script (1 round-trip total).

func BenchmarkLuaScript(b *testing.B) {
	sizes := []int{10, 50, 100}

	for _, numKeys := range sizes {
		b.Run(fmt.Sprintf("Individual_%d", numKeys), func(b *testing.B) {
			mock := NewRedisMock(benchLatency, 10)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				indivTime, _ := LuaScriptDemo(mock, numKeys)
				benchDur = indivTime
			}
		})

		b.Run(fmt.Sprintf("LuaScript_%d", numKeys), func(b *testing.B) {
			mock := NewRedisMock(benchLatency, 10)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, luaTime := LuaScriptDemo(mock, numKeys)
				benchDur = luaTime
			}
		})
	}
}

// --- Benchmark: Raw Pipeline Execution (no timing wrapper) ---
// Directly benchmarks ExecPipeline to measure allocation overhead.

func BenchmarkExecPipeline_SET(b *testing.B) {
	sizes := []int{10, 50, 100}

	for _, numOps := range sizes {
		b.Run(fmt.Sprintf("ops=%d", numOps), func(b *testing.B) {
			mock := NewRedisMock(0, 10) // No latency — measure pure allocation cost
			cmds := make([]PipelineCommand, numOps)
			for i := 0; i < numOps; i++ {
				cmds[i] = PipelineCommand{
					Op:    "SET",
					Key:   fmt.Sprintf("key:%d", i),
					Value: fmt.Sprintf("value:%d", i),
				}
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				results := mock.ExecPipeline(cmds)
				benchVal = results[0].Value
			}
		})
	}
}

func BenchmarkExecPipeline_GET(b *testing.B) {
	sizes := []int{10, 50, 100}

	for _, numOps := range sizes {
		b.Run(fmt.Sprintf("ops=%d", numOps), func(b *testing.B) {
			mock := NewRedisMock(0, 10) // No latency — measure pure allocation cost
			// Pre-populate data
			for i := 0; i < numOps; i++ {
				mock.data[fmt.Sprintf("key:%d", i)] = fmt.Sprintf("value:%d", i)
			}
			cmds := make([]PipelineCommand, numOps)
			for i := 0; i < numOps; i++ {
				cmds[i] = PipelineCommand{
					Op:  "GET",
					Key: fmt.Sprintf("key:%d", i),
				}
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				results := mock.ExecPipeline(cmds)
				benchVal = results[numOps-1].Value
				benchFound = results[numOps-1].Found
			}
		})
	}
}
