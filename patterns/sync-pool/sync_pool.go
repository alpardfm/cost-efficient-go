package sync_pool

import (
	"sync"
)

// ============================================================
// PATTERN 11: sync.Pool — Memory Pooling for Buffer Reuse
// ============================================================
// Problem: Allocating new buffers for every HTTP request/response
// creates massive GC pressure at high throughput:
// - Each 4KB buffer = 1 heap allocation
// - At 100K req/sec = 400MB/sec of garbage
// - GC pause time grows with heap size
//
// This pattern demonstrates:
// 1. Buffer reuse for HTTP request/response processing
// 2. GC pressure reduction at high throughput
// 3. When NOT to use sync.Pool (small objects < 64 bytes)
// 4. Cost projection at 1M, 10M, 100M requests/day
// ============================================================

// --- BufferPool: sync.Pool Wrapper ---

// BufferPool wraps sync.Pool for []byte buffers of a specific size.
// It ensures buffers are properly reset before reuse.
type BufferPool struct {
	pool sync.Pool
	size int
}

// NewBufferPool creates a pool that produces []byte slices of the given size.
func NewBufferPool(size int) *BufferPool {
	bp := &BufferPool{size: size}
	bp.pool = sync.Pool{
		New: func() interface{} {
			buf := make([]byte, bp.size)
			return &buf
		},
	}
	return bp
}

// Get retrieves a buffer from the pool.
func (bp *BufferPool) Get() *[]byte {
	return bp.pool.Get().(*[]byte)
}

// Put returns a buffer to the pool after resetting it.
func (bp *BufferPool) Put(buf *[]byte) {
	b := *buf
	// Reset buffer contents to zero length but keep capacity
	*buf = b[:bp.size]
	bp.pool.Put(buf)
}

// --- Naive Implementation (Before) ---

// NaiveBufferAlloc allocates a new buffer every time — simulates
// typical HTTP handler that creates fresh buffers per request.
func NaiveBufferAlloc(size int) []byte {
	buf := make([]byte, size)
	// Simulate writing HTTP response data
	for i := range buf {
		buf[i] = byte(i % 256)
	}
	return buf
}

// --- Pooled Implementation (After) ---

// PooledBufferAlloc retrieves a buffer from the pool, uses it,
// and returns it for reuse — zero allocation on the hot path.
func PooledBufferAlloc(pool *BufferPool) []byte {
	buf := pool.Get()
	b := *buf
	// Simulate writing HTTP response data
	for i := range b {
		b[i] = byte(i % 256)
	}
	// In real code, you'd defer pool.Put(buf) after response is sent
	pool.Put(buf)
	return b
}

// --- Small Object Demo: When NOT to Use sync.Pool ---

// SmallObject represents a tiny struct (< 64 bytes) where pool
// overhead exceeds allocation cost.
type SmallObject struct {
	ID    int32
	Value int32
}

// GlobalSmallSink prevents compiler from optimizing away small object allocations.
var GlobalSmallSink *SmallObject

// GlobalSink prevents compiler from optimizing away allocations.
var GlobalSink []byte

// --- Cost Projection ---

// CostProjection calculates AWS cost savings from buffer pooling.
type CostProjection struct {
	RequestsPerDay    int
	BufferSize        int
	NaiveAllocPerReq  int64 // bytes allocated per request (naive)
	PooledAllocPerReq int64 // bytes allocated per request (pooled)
}
