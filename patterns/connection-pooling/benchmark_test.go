package main

import (
	"net"
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
