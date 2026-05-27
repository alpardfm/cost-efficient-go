package main

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// ============================================================
// PATTERN 9: Worker Pool Pattern
// ============================================================
// Problem: Spawning unbounded goroutines for every task causes:
// - Memory explosion (each goroutine ~2-8KB stack)
// - CPU thrashing from context switching
// - Upstream overload (too many concurrent connections)
//
// This pattern demonstrates:
// 1. Unbounded goroutines vs fixed worker pool
// 2. Channel-based job distribution
// 3. Graceful shutdown
// 4. Pool sizing strategies
// ============================================================

// --- Task Simulation ---

// Task represents a unit of work.
type Task struct {
	ID       int
	Duration time.Duration
}

// ProcessTask simulates CPU + I/O work.
func ProcessTask(task Task) int {
	// Simulate work
	result := 0
	for i := 0; i < 1000; i++ {
		result += i * task.ID
	}
	time.Sleep(task.Duration) // Simulate I/O
	return result
}

// --- Unbounded Goroutines (Bad) ---

// ProcessUnbounded spawns one goroutine per task — no limit.
func ProcessUnbounded(tasks []Task) []int {
	results := make([]int, len(tasks))
	var wg sync.WaitGroup

	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t Task) {
			defer wg.Done()
			results[idx] = ProcessTask(t)
		}(i, task)
	}

	wg.Wait()
	return results
}

// --- Worker Pool (Good) ---

// WorkerPool processes tasks with a fixed number of workers.
type WorkerPool struct {
	workers int
	jobs    chan indexedTask
	results []int
	wg      sync.WaitGroup
}

type indexedTask struct {
	index int
	task  Task
}

// NewWorkerPool creates a pool with n workers.
func NewWorkerPool(workers, taskCount int) *WorkerPool {
	return &WorkerPool{
		workers: workers,
		jobs:    make(chan indexedTask, taskCount),
		results: make([]int, taskCount),
	}
}

// Start launches workers.
func (p *WorkerPool) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for job := range p.jobs {
				p.results[job.index] = ProcessTask(job.task)
			}
		}()
	}
}

// Submit adds a task to the pool.
func (p *WorkerPool) Submit(index int, task Task) {
	p.jobs <- indexedTask{index: index, task: task}
}

// Wait closes the job channel and waits for all workers to finish.
func (p *WorkerPool) Wait() []int {
	close(p.jobs)
	p.wg.Wait()
	return p.results
}

// ProcessWithPool processes tasks using a fixed worker pool.
func ProcessWithPool(tasks []Task, workers int) []int {
	pool := NewWorkerPool(workers, len(tasks))
	pool.Start()

	for i, task := range tasks {
		pool.Submit(i, task)
	}

	return pool.Wait()
}

// --- errgroup.SetLimit (Idiomatic Alternative) ---

// ProcessWithErrgroup processes tasks using errgroup with SetLimit.
// This is the idiomatic Go approach for bounded concurrency when:
// - Tasks are independent and don't need backpressure
// - You want simple error propagation
// - The pool is short-lived (process a batch, then done)
func ProcessWithErrgroup(tasks []Task, workers int) []int {
	results := make([]int, len(tasks))

	g, _ := errgroup.WithContext(context.Background())
	g.SetLimit(workers)

	for i, task := range tasks {
		i, task := i, task // capture loop variables
		g.Go(func() error {
			results[i] = ProcessTask(task)
			return nil
		})
	}

	_ = g.Wait()
	return results
}

// --- Demonstration ---

func main() {
	fmt.Println("=== Worker Pool Pattern ===")
	fmt.Println()

	// Generate tasks
	tasks := make([]Task, 1000)
	for i := range tasks {
		tasks[i] = Task{
			ID:       i,
			Duration: time.Duration(rand.Intn(5)) * time.Millisecond,
		}
	}

	// 1. Unbounded goroutines
	fmt.Println("--- Unbounded Goroutines (1000 tasks) ---")
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	start := time.Now()
	ProcessUnbounded(tasks)
	unboundedDuration := time.Since(start)

	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)
	fmt.Printf("Time: %v\n", unboundedDuration)
	fmt.Printf("Goroutines spawned: %d\n", len(tasks))
	fmt.Printf("Memory allocated: %d KB\n", (memAfter.TotalAlloc-memBefore.TotalAlloc)/1024)
	fmt.Println()

	// 2. Worker pool with different sizes
	fmt.Println("--- Worker Pool (1000 tasks, varying workers) ---")
	for _, workers := range []int{4, 8, 16, 32, 64} {
		start := time.Now()
		ProcessWithPool(tasks, workers)
		duration := time.Since(start)
		fmt.Printf("  workers=%2d → %v\n", workers, duration)
	}
	fmt.Println()

	// 3. errgroup.SetLimit comparison
	fmt.Println("--- errgroup.SetLimit (1000 tasks, varying workers) ---")
	for _, workers := range []int{4, 8, 16, 32, 64} {
		start := time.Now()
		ProcessWithErrgroup(tasks, workers)
		duration := time.Since(start)
		fmt.Printf("  limit=%2d → %v\n", workers, duration)
	}
	fmt.Println()

	// 4. Goroutine count comparison
	fmt.Println("--- Peak Goroutine Count ---")
	fmt.Printf("  Unbounded: %d goroutines (1 per task)\n", len(tasks))
	fmt.Printf("  Pool(16):  %d goroutines (fixed)\n", 16)
	fmt.Printf("  Memory per goroutine: ~2-8 KB\n")
	fmt.Printf("  Unbounded 1M tasks: ~2-8 GB goroutine stacks!\n")
	fmt.Printf("  Pool(16) 1M tasks:  ~32-128 KB goroutine stacks\n")
}
