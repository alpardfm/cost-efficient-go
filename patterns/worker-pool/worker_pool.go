package worker_pool

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

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

// ProcessWithErrgroup processes tasks using errgroup with SetLimit.
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
