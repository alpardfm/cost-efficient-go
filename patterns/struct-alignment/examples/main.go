package main

import (
	"fmt"
	"strings"
	"time"
	"unsafe"
)

type BadUser struct {
	ID     int32
	Active bool
	Name   string
	Age    int8
}

type GoodUser struct {
	ID     int32
	Age    int8
	Active bool
	Name   string
}

func main() {
	fmt.Println("🔬 Memory Layout & Struct Alignment")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("📅 Date: %s\n\n", time.Now().Format("2006-01-02"))

	// Show struct sizes
	fmt.Println("📐 STRUCT SIZES ANALYSIS")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("BadUser size:  %d bytes\n", unsafe.Sizeof(BadUser{}))
	fmt.Printf("GoodUser size: %d bytes\n", unsafe.Sizeof(GoodUser{}))
	fmt.Printf("Savings per user: %d bytes (%.1f%%)\n\n",
		unsafe.Sizeof(BadUser{})-unsafe.Sizeof(GoodUser{}),
		float64(unsafe.Sizeof(BadUser{})-unsafe.Sizeof(GoodUser{}))/float64(unsafe.Sizeof(BadUser{}))*100)

	// Benchmark BadUser
	fmt.Println("📊 BENCHMARK: BEFORE OPTIMIZATION (BadUser)")
	fmt.Println(strings.Repeat("-", 40))
	badTime, badMemory := benchmarkBadUser(1_000_000)
	fmt.Printf("⏱️  Time for 1M users: %v\n", badTime)
	fmt.Printf("💾 Memory: %.2f MB\n", float64(badMemory)/(1024*1024))

	// Explanation
	fmt.Println("\n🔧 OPTIMIZATION EXPLANATION")
	fmt.Println(strings.Repeat("-", 40))
	explainMemoryLayout()

	// Benchmark GoodUser
	fmt.Println("\n📈 BENCHMARK: AFTER OPTIMIZATION (GoodUser)")
	fmt.Println(strings.Repeat("-", 40))
	goodTime, goodMemory := benchmarkGoodUser(1_000_000)
	fmt.Printf("⏱️  Time for 1M users: %v\n", goodTime)
	fmt.Printf("💾 Memory: %.2f MB\n", float64(goodMemory)/(1024*1024))

	// Results comparison
	fmt.Println("\n🏆 RESULTS COMPARISON")
	fmt.Println(strings.Repeat("=", 60))

	timeImprovement := float64(badTime.Nanoseconds()-goodTime.Nanoseconds()) / float64(badTime.Nanoseconds()) * 100
	memoryImprovement := float64(badMemory-goodMemory) / float64(badMemory) * 100

	fmt.Printf("⚡ Time Improvement:    %.1f%% faster\n", timeImprovement)
	fmt.Printf("💾 Memory Improvement: %.1f%% less memory\n", memoryImprovement)
	fmt.Printf("📦 Memory Saved:       %.2f MB per 1M users\n\n",
		float64(badMemory-goodMemory)/(1024*1024))

	// Cost analysis
	fmt.Println("💰 COST IMPACT ANALYSIS")
	fmt.Println(strings.Repeat("=", 60))
	calculateCostImpact(badMemory, goodMemory)

	fmt.Println("\n✅ PATTERN COMPLETED! 🎉")
}

func benchmarkBadUser(count int) (time.Duration, uintptr) {
	start := time.Now()

	// Pre-allocate slice
	users := make([]BadUser, 0, count)

	for i := 0; i < count; i++ {
		users = append(users, BadUser{
			ID:     int32(i),
			Active: i%2 == 0,
			Name:   fmt.Sprintf("User_%d_Test_Name_That_Is_Long", i),
			Age:    int8(i % 100),
		})
	}

	elapsed := time.Since(start)

	// Calculate total memory
	totalMemory := unsafe.Sizeof(BadUser{}) * uintptr(len(users))

	return elapsed, totalMemory
}

func benchmarkGoodUser(count int) (time.Duration, uintptr) {
	start := time.Now()

	// Pre-allocate slice
	users := make([]GoodUser, 0, count)

	for i := 0; i < count; i++ {
		users = append(users, GoodUser{
			ID:     int32(i),
			Age:    int8(i % 100),
			Active: i%2 == 0,
			Name:   fmt.Sprintf("User_%d_Test_Name_That_Is_Long", i),
		})
	}

	elapsed := time.Since(start)

	// Calculate total memory
	totalMemory := unsafe.Sizeof(GoodUser{}) * uintptr(len(users))

	return elapsed, totalMemory
}

func explainMemoryLayout() {
	fmt.Println("Go aligns struct fields to natural boundaries:")
	fmt.Println()
	fmt.Println("BAD STRUCT (32 bytes):")
	fmt.Println("  ID (int32):    4 bytes  @ offset 0")
	fmt.Println("  Active (bool): 1 byte   @ offset 4")
	fmt.Println("  <padding>:     3 bytes  (wasted!)")
	fmt.Println("  Name (string): 16 bytes @ offset 8")
	fmt.Println("  Age (int8):    1 byte   @ offset 24")
	fmt.Println("  <padding>:     7 bytes  (wasted!)")
	fmt.Println("  Total:         32 bytes")
	fmt.Println()
	fmt.Println("GOOD STRUCT (24 bytes):")
	fmt.Println("  ID (int32):    4 bytes  @ offset 0")
	fmt.Println("  Age (int8):    1 byte   @ offset 4")
	fmt.Println("  Active (bool): 1 byte   @ offset 5")
	fmt.Println("  <padding>:     2 bytes")
	fmt.Println("  Name (string): 16 bytes @ offset 8")
	fmt.Println("  Total:         24 bytes (8 bytes saved!)")
	fmt.Println()
	fmt.Println("💡 Rule: Group fields by size (largest to smallest)")
}

func calculateCostImpact(beforeMem, afterMem uintptr) {
	// Calculate memory saved
	memorySavedMB := float64(beforeMem-afterMem) / (1024 * 1024)

	// Cloud pricing assumptions (AWS us-east-1)
	awsT3MediumCost := 30.0  // $30/month for t3.medium
	awsRAMPerInstance := 8.0 // 8GB RAM
	costPerGBMonth := awsT3MediumCost / awsRAMPerInstance

	// For 1 million users
	monthlySavings := memorySavedMB / 1024 * costPerGBMonth

	fmt.Println("☁️  CLOUD ASSUMPTIONS (AWS us-east-1):")
	fmt.Printf("  • t3.medium instance: $%.2f/month\n", awsT3MediumCost)
	fmt.Printf("  • 8GB RAM per instance\n")
	fmt.Printf("  • Cost per GB-month: $%.2f\n", costPerGBMonth)
	fmt.Printf("  • 1 million users in memory\n")

	fmt.Println("\n🧮 CALCULATIONS:")
	fmt.Printf("  Memory saved: %.2f MB\n", memorySavedMB)
	fmt.Printf("  Monthly savings: $%.4f\n", monthlySavings)
	fmt.Printf("  Annual savings:  $%.4f\n", monthlySavings*12)

	fmt.Println("\n📈 SCALING PROJECTIONS:")
	fmt.Println("  For different user counts:")

	userCounts := []int{1_000_000, 10_000_000, 100_000_000, 1_000_000_000}
	for _, users := range userCounts {
		scaledSavings := monthlySavings * float64(users) / 1_000_000
		if users >= 1_000_000_000 {
			fmt.Printf("  • %d users: $%.2f/month savings\n", users, scaledSavings)
		} else {
			fmt.Printf("  • %d users: $%.4f/month savings\n", users, scaledSavings)
		}
	}

	fmt.Println("\n💡 ADDITIONAL BENEFITS (not quantified):")
	fmt.Println("  • Reduced GC pressure → lower CPU usage")
	fmt.Println("  • Better cache locality → faster access")
	fmt.Println("  • Lower memory bandwidth usage")
	fmt.Println("  • Reduced swap usage (if memory constrained)")

	fmt.Println("\n🎯 ACTION ITEMS:")
	fmt.Println("  1. Run: go test -bench=. -benchmem (see benchmark results)")
	fmt.Println("  2. Apply to your production structs")
	fmt.Println("  3. Monitor memory usage before/after")
	fmt.Println("  4. Share findings with your team")
}
