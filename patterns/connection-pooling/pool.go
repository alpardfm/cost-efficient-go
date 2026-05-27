package connection_pooling

import (
	"net"
	"sync"
	"time"
)

// Pool is a simple connection pool for demonstration and benchmarking.
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
