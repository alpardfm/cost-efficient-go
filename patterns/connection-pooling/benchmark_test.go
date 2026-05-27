package main

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

// ============================================================
// Benchmarks: Connection-per-request vs Pooled
// ============================================================

var testAddr string
var testShutdown func()

func init() {
	testAddr, testShutdown = startEchoServer()
	time.Sleep(50 * time.Millisecond)
}

func BenchmarkConnectionPerRequest(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		conn, err := net.DialTimeout("tcp", testAddr, 5*time.Second)
		if err != nil {
			b.Fatal(err)
		}
		conn.Write([]byte("PING\n"))
		buf := make([]byte, 64)
		conn.Read(buf)
		conn.Close()
	}
}

func BenchmarkPooledConnection(b *testing.B) {
	pool := NewPool(PoolConfig{
		MaxSize: 20,
		MaxIdle: 10,
		Factory: func() (net.Conn, error) {
			return net.DialTimeout("tcp", testAddr, 5*time.Second)
		},
	})
	defer pool.Close()

	// Warm up pool
	conn, _ := pool.Get()
	pool.Put(conn)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		conn, err := pool.Get()
		if err != nil {
			b.Fatal(err)
		}
		conn.Write([]byte("PING\n"))
		buf := make([]byte, 64)
		conn.Read(buf)
		pool.Put(conn)
	}
}

func BenchmarkPooledConnectionConcurrent(b *testing.B) {
	pool := NewPool(PoolConfig{
		MaxSize: 20,
		MaxIdle: 10,
		Factory: func() (net.Conn, error) {
			return net.DialTimeout("tcp", testAddr, 5*time.Second)
		},
	})
	defer pool.Close()

	// Warm up
	for i := 0; i < 5; i++ {
		conn, _ := pool.Get()
		pool.Put(conn)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			conn, err := pool.Get()
			if err != nil {
				b.Fatal(err)
			}
			conn.Write([]byte("PING\n"))
			buf := make([]byte, 64)
			conn.Read(buf)
			pool.Put(conn)
		}
	})
}

func BenchmarkTCPDialOnly(b *testing.B) {
	// Measures raw TCP dial cost (limited by OS port recycling)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		conn, err := net.DialTimeout("tcp", testAddr, 5*time.Second)
		if err != nil {
			b.Skip("port exhaustion — expected for rapid dial/close")
			return
		}
		conn.Close()
	}
}

// ============================================================
// Concurrent Benchmarks: Pool Size vs Concurrency Level
// ============================================================
// These benchmarks demonstrate WHEN pool size matters.
// With sequential access, even maxIdle=1 achieves 99% reuse.
// With concurrent access, pool size must match concurrency
// to avoid creating new connections on every request.
// ============================================================

// BenchmarkConcurrentPoolSize runs benchmarks across concurrency levels
// and pool sizes to show the sweet spot where pool size matches concurrency.
func BenchmarkConcurrentPoolSize(b *testing.B) {
	concurrencyLevels := []int{1, 5, 10, 25, 50, 100}
	poolSizes := []int{1, 5, 10, 25, 50, 100}

	for _, concurrency := range concurrencyLevels {
		for _, poolSize := range poolSizes {
			name := fmt.Sprintf("concurrency=%d/poolSize=%d", concurrency, poolSize)
			b.Run(name, func(b *testing.B) {
				pool := NewPool(PoolConfig{
					MaxSize: poolSize * 2,
					MaxIdle: poolSize,
					Factory: func() (net.Conn, error) {
						return net.DialTimeout("tcp", testAddr, 5*time.Second)
					},
				})
				defer pool.Close()

				// Warm up pool to maxIdle
				warmConns := make([]net.Conn, 0, poolSize)
				for i := 0; i < poolSize; i++ {
					conn, err := pool.Get()
					if err != nil {
						b.Fatal(err)
					}
					warmConns = append(warmConns, conn)
				}
				for _, conn := range warmConns {
					pool.Put(conn)
				}

				b.ResetTimer()
				b.ReportAllocs()
				b.SetParallelism(concurrency)
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						conn, err := pool.Get()
						if err != nil {
							b.Fatal(err)
						}
						conn.Write([]byte("PING\n"))
						buf := make([]byte, 64)
						conn.Read(buf)
						pool.Put(conn)
					}
				})
			})
		}
	}
}

// TestConcurrentPoolEfficiency measures connection creation vs reuse
// at different concurrency levels to demonstrate when pool size matters.
// Simulates realistic workload where each request holds a connection for
// a short duration (simulating query execution time), creating contention.
func TestConcurrentPoolEfficiency(t *testing.T) {
	concurrencyLevels := []int{1, 5, 10, 25, 50, 100}
	poolSizes := []int{1, 5, 10, 25, 50, 100}
	requestsPerGoroutine := 20
	// Simulate holding connection for 500µs (like a fast DB query)
	// This creates contention: while one goroutine holds a connection,
	// others must either get from pool or create new ones.
	holdDuration := 500 * time.Microsecond

	t.Logf("\n%-12s %-10s %-10s %-10s %-10s",
		"Concurrency", "PoolSize", "Created", "Reused", "Reuse%")
	t.Logf("%-12s %-10s %-10s %-10s %-10s",
		"───────────", "────────", "───────", "──────", "──────")

	for _, concurrency := range concurrencyLevels {
		for _, poolSize := range poolSizes {
			pool := NewPool(PoolConfig{
				MaxSize: poolSize * 2,
				MaxIdle: poolSize,
				Factory: func() (net.Conn, error) {
					return net.DialTimeout("tcp", testAddr, 5*time.Second)
				},
			})

			// Run concurrent workload — all goroutines start simultaneously
			var wg sync.WaitGroup
			startBarrier := make(chan struct{})
			wg.Add(concurrency)
			for g := 0; g < concurrency; g++ {
				go func() {
					defer wg.Done()
					<-startBarrier // Wait for all goroutines to be ready
					for r := 0; r < requestsPerGoroutine; r++ {
						conn, err := pool.Get()
						if err != nil {
							continue
						}
						conn.Write([]byte("PING\n"))
						buf := make([]byte, 64)
						conn.Read(buf)
						// Simulate holding connection (query execution time)
						time.Sleep(holdDuration)
						pool.Put(conn)
					}
				}()
			}
			close(startBarrier) // Release all goroutines at once
			wg.Wait()

			created, reused := pool.Stats()
			total := created + reused
			reusePercent := 0.0
			if total > 0 {
				reusePercent = float64(reused) / float64(total) * 100
			}

			t.Logf("%-12d %-10d %-10d %-10d %-9.1f%%",
				concurrency, poolSize, created, reused, reusePercent)

			pool.Close()
		}
		t.Log("") // separator between concurrency groups
	}
}

// BenchmarkPoolSizeMismatch specifically demonstrates the problem:
// high concurrency with a small pool forces many new connections.
func BenchmarkPoolSizeMismatch(b *testing.B) {
	// Case 1: Pool too small for concurrency (pool=5, concurrency=50)
	b.Run("undersized_pool5_conc50", func(b *testing.B) {
		pool := NewPool(PoolConfig{
			MaxSize: 10,
			MaxIdle: 5,
			Factory: func() (net.Conn, error) {
				return net.DialTimeout("tcp", testAddr, 5*time.Second)
			},
		})
		defer pool.Close()

		b.ResetTimer()
		b.ReportAllocs()
		b.SetParallelism(50)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				conn, err := pool.Get()
				if err != nil {
					b.Fatal(err)
				}
				conn.Write([]byte("PING\n"))
				buf := make([]byte, 64)
				conn.Read(buf)
				pool.Put(conn)
			}
		})
	})

	// Case 2: Pool matches concurrency (pool=50, concurrency=50)
	b.Run("matched_pool50_conc50", func(b *testing.B) {
		pool := NewPool(PoolConfig{
			MaxSize: 100,
			MaxIdle: 50,
			Factory: func() (net.Conn, error) {
				return net.DialTimeout("tcp", testAddr, 5*time.Second)
			},
		})
		defer pool.Close()

		// Warm up
		warmConns := make([]net.Conn, 0, 50)
		for i := 0; i < 50; i++ {
			conn, err := pool.Get()
			if err != nil {
				b.Fatal(err)
			}
			warmConns = append(warmConns, conn)
		}
		for _, conn := range warmConns {
			pool.Put(conn)
		}

		b.ResetTimer()
		b.ReportAllocs()
		b.SetParallelism(50)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				conn, err := pool.Get()
				if err != nil {
					b.Fatal(err)
				}
				conn.Write([]byte("PING\n"))
				buf := make([]byte, 64)
				conn.Read(buf)
				pool.Put(conn)
			}
		})
	})

	// Case 3: Pool oversized (pool=100, concurrency=50) — no extra benefit
	b.Run("oversized_pool100_conc50", func(b *testing.B) {
		pool := NewPool(PoolConfig{
			MaxSize: 200,
			MaxIdle: 100,
			Factory: func() (net.Conn, error) {
				return net.DialTimeout("tcp", testAddr, 5*time.Second)
			},
		})
		defer pool.Close()

		// Warm up to match actual concurrency (50), not pool size
		warmConns := make([]net.Conn, 0, 50)
		for i := 0; i < 50; i++ {
			conn, err := pool.Get()
			if err != nil {
				b.Fatal(err)
			}
			warmConns = append(warmConns, conn)
		}
		for _, conn := range warmConns {
			pool.Put(conn)
		}

		b.ResetTimer()
		b.ReportAllocs()
		b.SetParallelism(50)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				conn, err := pool.Get()
				if err != nil {
					b.Fatal(err)
				}
				conn.Write([]byte("PING\n"))
				buf := make([]byte, 64)
				conn.Read(buf)
				pool.Put(conn)
			}
		})
	})
}
