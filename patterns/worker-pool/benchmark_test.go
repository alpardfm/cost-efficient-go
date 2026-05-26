package main

import (
	"testing"
	"time"
)

// ============================================================
// Benchmarks: Unbounded vs Worker Pool
// ============================================================

func generateTasks(n int, ioTime time.Duration) []Task {
	tasks := make([]Task, n)
	for i := range tasks {
		tasks[i] = Task{ID: i, Duration: ioTime}
	}
	return tasks
}

// --- CPU-bound tasks (no I/O wait) ---

func BenchmarkUnbounded100CPUOnly(b *testing.B) {
	tasks := generateTasks(100, 0)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ProcessUnbounded(tasks)
	}
}

func BenchmarkPool8Workers100CPUOnly(b *testing.B) {
	tasks := generateTasks(100, 0)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ProcessWithPool(tasks, 8)
	}
}

func BenchmarkPool16Workers100CPUOnly(b *testing.B) {
	tasks := generateTasks(100, 0)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ProcessWithPool(tasks, 16)
	}
}

// --- I/O-bound tasks (1ms simulated I/O) ---

func BenchmarkUnbounded100IO(b *testing.B) {
	tasks := generateTasks(100, 1*time.Millisecond)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ProcessUnbounded(tasks)
	}
}

func BenchmarkPool8Workers100IO(b *testing.B) {
	tasks := generateTasks(100, 1*time.Millisecond)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ProcessWithPool(tasks, 8)
	}
}

func BenchmarkPool16Workers100IO(b *testing.B) {
	tasks := generateTasks(100, 1*time.Millisecond)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ProcessWithPool(tasks, 16)
	}
}

func BenchmarkPool32Workers100IO(b *testing.B) {
	tasks := generateTasks(100, 1*time.Millisecond)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ProcessWithPool(tasks, 32)
	}
}

// --- Scale: 1000 tasks ---

func BenchmarkUnbounded1000IO(b *testing.B) {
	tasks := generateTasks(1000, 1*time.Millisecond)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ProcessUnbounded(tasks)
	}
}

func BenchmarkPool32Workers1000IO(b *testing.B) {
	tasks := generateTasks(1000, 1*time.Millisecond)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ProcessWithPool(tasks, 32)
	}
}

func BenchmarkPool64Workers1000IO(b *testing.B) {
	tasks := generateTasks(1000, 1*time.Millisecond)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ProcessWithPool(tasks, 64)
	}
}
