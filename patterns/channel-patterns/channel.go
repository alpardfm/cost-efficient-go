package channel_patterns

import (
	"runtime"
	"sync"
	"time"
)

// --- Global variables to prevent compiler optimization ---
var (
	globalResult int
	globalSum    int64
)

// UnbufferedChannel demonstrates synchronous communication between
// a producer and consumer. Every send blocks until the receiver reads,
// creating tight coupling and limiting throughput.
// The bottleneck is channel synchronization, not processing work.
func UnbufferedChannel(messageCount int) time.Duration {
	ch := make(chan int)

	start := time.Now()

	// Consumer — reads as fast as possible
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		sum := int64(0)
		for val := range ch {
			sum += int64(val)
		}
		globalSum = sum
	}()

	// Producer — blocks on every send until consumer reads
	for i := 0; i < messageCount; i++ {
		ch <- i
	}
	close(ch)
	wg.Wait()

	return time.Since(start)
}

// BufferedChannel demonstrates asynchronous communication with a buffer.
// Producer can send up to `size` messages without blocking, decoupling
// producer and consumer speeds. Buffer size must be > 0 for valid comparison.
func BufferedChannel(messageCount, size int) time.Duration {
	if size <= 0 {
		size = 1 // Enforce minimum buffer size > 0
	}
	ch := make(chan int, size)

	start := time.Now()

	// Consumer — reads as fast as possible
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		sum := int64(0)
		for val := range ch {
			sum += int64(val)
		}
		globalSum = sum
	}()

	// Producer — can burst up to `size` messages without blocking
	for i := 0; i < messageCount; i++ {
		ch <- i
	}
	close(ch)
	wg.Wait()

	return time.Since(start)
}

// MutexBased demonstrates a channel-free approach using sync.Mutex.
// Useful when you need shared state access without the overhead of
// channel scheduling. Lower overhead for simple counter/accumulator patterns.
func MutexBased(messageCount int) time.Duration {
	var mu sync.Mutex
	var sum int64

	start := time.Now()

	var wg sync.WaitGroup
	wg.Add(1)

	// Consumer goroutine reads from a shared slice
	data := make([]int, 0, messageCount)
	done := make(chan struct{})

	go func() {
		defer wg.Done()
		<-done // Wait for producer to finish
		mu.Lock()
		localSum := int64(0)
		for _, val := range data {
			localSum += int64(val)
		}
		sum = localSum
		mu.Unlock()
	}()

	// Producer — append under lock
	mu.Lock()
	for i := 0; i < messageCount; i++ {
		data = append(data, i)
	}
	mu.Unlock()
	close(done)
	wg.Wait()

	globalSum = sum
	return time.Since(start)
}

// FanOutFanIn distributes work across multiple worker goroutines (fan-out)
// and collects results into a single channel (fan-in). This pattern
// maximizes CPU utilization on multi-core systems for CPU-bound work.
func FanOutFanIn(items []int, numWorkers int) []int {
	if numWorkers <= 0 {
		numWorkers = runtime.GOMAXPROCS(0)
	}

	// Fan-out: distribute work to workers
	jobs := make(chan int, len(items))
	results := make(chan int, len(items))

	// Start workers
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range jobs {
				// Simulate CPU-bound processing
				results <- ProcessItem(item)
			}
		}()
	}

	// Send all jobs
	for _, item := range items {
		jobs <- item
	}
	close(jobs)

	// Wait for all workers to finish, then close results
	go func() {
		wg.Wait()
		close(results)
	}()

	// Fan-in: collect all results
	output := make([]int, 0, len(items))
	for result := range results {
		output = append(output, result)
	}

	return output
}

// ProcessItem simulates CPU-bound work on a single item.
// Uses iterative computation to simulate real processing cost.
func ProcessItem(item int) int {
	// Simulate moderate CPU work (not trivial, not too heavy)
	result := item
	for j := 0; j < 100; j++ {
		result = (result*31 + j) % 1000000007
	}
	return result
}

// FanOutFanInSequential processes items sequentially for comparison.
func FanOutFanInSequential(items []int) []int {
	output := make([]int, 0, len(items))
	for _, item := range items {
		output = append(output, ProcessItem(item))
	}
	return output
}
