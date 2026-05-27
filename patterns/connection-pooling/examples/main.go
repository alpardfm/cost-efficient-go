package main

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// ============================================================
// PATTERN 6: Connection Pooling
// ============================================================
// Problem: Creating a new TCP connection per request is
// expensive — TCP handshake, TLS negotiation, and connection
// setup add 1-10ms of latency per request.
//
// This pattern demonstrates:
// 1. Cost of connection-per-request vs pooled connections
// 2. Simple connection pool implementation
// 3. Pool sizing strategies
// 4. Idle connection management
// ============================================================

// --- Connection Pool Implementation ---

// Pool is a simple connection pool for demonstration.
type Pool struct {
	mu       sync.Mutex
	conns    []net.Conn
	factory  func() (net.Conn, error)
	maxSize  int
	maxIdle  int
	created  int
	reused   int
	timeouts int
}

// PoolConfig contains pool configuration.
type PoolConfig struct {
	MaxSize int
	MaxIdle int
	Factory func() (net.Conn, error)
}

// NewPool creates a connection pool.
func NewPool(cfg PoolConfig) *Pool {
	return &Pool{
		conns:   make([]net.Conn, 0, cfg.MaxIdle),
		factory: cfg.Factory,
		maxSize: cfg.MaxSize,
		maxIdle: cfg.MaxIdle,
	}
}

// Get retrieves a connection from the pool or creates a new one.
func (p *Pool) Get() (net.Conn, error) {
	p.mu.Lock()
	if len(p.conns) > 0 {
		conn := p.conns[len(p.conns)-1]
		p.conns = p.conns[:len(p.conns)-1]
		p.reused++
		p.mu.Unlock()
		return conn, nil
	}
	p.created++
	p.mu.Unlock()

	return p.factory()
}

// Put returns a connection to the pool.
func (p *Pool) Put(conn net.Conn) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.conns) >= p.maxIdle {
		conn.Close()
		return
	}
	p.conns = append(p.conns, conn)
}

// Stats returns pool statistics.
func (p *Pool) Stats() (created, reused int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.created, p.reused
}

// Close closes all idle connections.
func (p *Pool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, conn := range p.conns {
		conn.Close()
	}
	p.conns = nil
}

// --- Simulation ---

// SimulateConnectionPerRequest creates a new connection for each request.
func SimulateConnectionPerRequest(addr string, requests int) time.Duration {
	start := time.Now()
	for i := 0; i < requests; i++ {
		conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
		if err != nil {
			continue
		}
		conn.Write([]byte("PING\n"))
		buf := make([]byte, 64)
		conn.Read(buf)
		conn.Close()
	}
	return time.Since(start)
}

// SimulatePooledRequests reuses connections from a pool.
func SimulatePooledRequests(pool *Pool, requests int) time.Duration {
	start := time.Now()
	for i := 0; i < requests; i++ {
		conn, err := pool.Get()
		if err != nil {
			continue
		}
		conn.Write([]byte("PING\n"))
		buf := make([]byte, 64)
		conn.Read(buf)
		pool.Put(conn)
	}
	return time.Since(start)
}

// startEchoServer starts a simple TCP echo server for testing.
func startEchoServer() (string, func()) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}

	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			default:
			}
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 1024)
				for {
					n, err := c.Read(buf)
					if err != nil {
						return
					}
					c.Write(buf[:n])
				}
			}(conn)
		}
	}()

	return listener.Addr().String(), func() {
		close(done)
		listener.Close()
	}
}

func main() {
	fmt.Println("=== Connection Pooling ===")
	fmt.Println()

	// Start echo server
	addr, shutdown := startEchoServer()
	defer shutdown()
	time.Sleep(50 * time.Millisecond) // Let server start

	requests := 100

	// Without pool
	fmt.Printf("--- %d requests WITHOUT pool ---\n", requests)
	duration := SimulateConnectionPerRequest(addr, requests)
	fmt.Printf("Total time: %v\n", duration)
	fmt.Printf("Avg per request: %v\n", duration/time.Duration(requests))
	fmt.Printf("Connections created: %d\n", requests)
	fmt.Println()

	// With pool
	fmt.Printf("--- %d requests WITH pool (maxIdle=10) ---\n", requests)
	pool := NewPool(PoolConfig{
		MaxSize: 20,
		MaxIdle: 10,
		Factory: func() (net.Conn, error) {
			return net.DialTimeout("tcp", addr, 5*time.Second)
		},
	})
	defer pool.Close()

	duration = SimulatePooledRequests(pool, requests)
	created, reused := pool.Stats()
	fmt.Printf("Total time: %v\n", duration)
	fmt.Printf("Avg per request: %v\n", duration/time.Duration(requests))
	fmt.Printf("Connections created: %d\n", created)
	fmt.Printf("Connections reused: %d\n", reused)
	fmt.Printf("Reuse ratio: %.1f%%\n", float64(reused)/float64(created+reused)*100)
	fmt.Println()

	// Pool sizing comparison
	fmt.Println("--- Pool Size Impact ---")
	for _, maxIdle := range []int{1, 5, 10, 20} {
		p := NewPool(PoolConfig{
			MaxSize: maxIdle * 2,
			MaxIdle: maxIdle,
			Factory: func() (net.Conn, error) {
				return net.DialTimeout("tcp", addr, 5*time.Second)
			},
		})
		SimulatePooledRequests(p, requests)
		c, r := p.Stats()
		fmt.Printf("  maxIdle=%2d → created=%2d, reused=%2d, reuse=%.0f%%\n",
			maxIdle, c, r, float64(r)/float64(c+r)*100)
		p.Close()
	}
}
