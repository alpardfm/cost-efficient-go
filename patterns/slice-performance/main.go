package main

import (
	"fmt"
	"strings"
	"time"
)

func main() {
	fmt.Println("🔬 DAY 2: Slice Performance & Pre-allocation")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("📅 Date: %s\n\n", time.Now().Format("2006-01-02"))

	// Problem demonstration
	fmt.Println("🎯 PROBLEM: Dynamic slice growth is expensive!")
	fmt.Println(strings.Repeat("-", 40))
	demoSliceGrowthProblem()

	// Benchmark comparisons
	fmt.Println("\n📊 BENCHMARK COMPARISONS")
	fmt.Println(strings.Repeat("-", 40))

	fmt.Println("1. Naive Append (no pre-allocation):")
	t1, m1 := benchmarkNaiveAppend(1_000_000)
	fmt.Printf("   Time: %v, Allocations: %d\n", t1, m1)

	fmt.Println("\n2. With make() and capacity:")
	t2, m2 := benchmarkWithMake(1_000_000)
	fmt.Printf("   Time: %v, Allocations: %d\n", t2, m2)
	fmt.Printf("   Improvement: %.1f%% faster, %d fewer allocations\n",
		float64(t1.Nanoseconds()-t2.Nanoseconds())/float64(t1.Nanoseconds())*100,
		m1-m2)

	fmt.Println("\n3. Fixed array (when size is known):")
	t3, m3 := benchmarkFixedArray(1_000_000)
	fmt.Printf("   Time: %v, Allocations: %d\n", t3, m3)
	fmt.Printf("   Improvement: %.1f%% faster, %d fewer allocations\n",
		float64(t1.Nanoseconds()-t3.Nanoseconds())/float64(t1.Nanoseconds())*100,
		m1-m3)

	// Slice internals explanation
	fmt.Println("\n🔧 SLICE INTERNALS EXPLANATION")
	fmt.Println(strings.Repeat("-", 40))
	explainSliceInternals()

	// Real-world example: Processing user data
	fmt.Println("\n💼 REAL-WORLD EXAMPLE: Processing User Data")
	fmt.Println(strings.Repeat("-", 40))
	demoUserProcessing()

	// Cost analysis
	fmt.Println("\n💰 COST IMPACT ANALYSIS")
	fmt.Println(strings.Repeat("=", 60))
	calculateCostImpact(t1, t2, m1, m2)

	fmt.Println("\n✅ DAY 2 COMPLETED! 🎉")
	fmt.Println("\n🔜 Next: Day 3 - Map Internals & Memory Overhead")
}

// ========== BENCHMARK FUNCTIONS ==========

func benchmarkNaiveAppend(count int) (time.Duration, int) {
	start := time.Now()
	allocations := 0

	var data []int
	for i := 0; i < count; i++ {
		// This causes multiple reallocations as slice grows
		data = append(data, i)
		allocations++
	}

	return time.Since(start), allocations
}

func benchmarkWithMake(count int) (time.Duration, int) {
	start := time.Now()
	allocations := 1 // Only one allocation for make()

	// Pre-allocate with known capacity
	data := make([]int, 0, count)
	for i := 0; i < count; i++ {
		data = append(data, i)
	}

	return time.Since(start), allocations
}

func benchmarkFixedArray(count int) (time.Duration, int) {
	start := time.Now()
	allocations := 1 // Single allocation

	// When size is known upfront, use array or slice with exact size
	data := make([]int, count)
	for i := 0; i < count; i++ {
		data[i] = i // Direct assignment, no append
	}

	return time.Since(start), allocations
}

// ========== EXPLANATION FUNCTIONS ==========

func demoSliceGrowthProblem() {
	fmt.Println("Slice growth pattern in Go (capacity doubles each time):")
	fmt.Println()

	var s []int
	fmt.Printf("Start: len=%d, cap=%d\n", len(s), cap(s))

	growthPattern := []int{1, 2, 3, 4, 5, 9, 17, 33}
	for _, n := range growthPattern {
		for i := 0; i < n; i++ {
			s = append(s, i)
		}
		fmt.Printf("After %3d appends: len=%2d, cap=%2d (waste: %2d slots)\n",
			n, len(s), cap(s), cap(s)-len(s))
	}

	fmt.Println("\n💡 Problem: Each reallocation:")
	fmt.Println("  1. Allocate new, larger array")
	fmt.Println("  2. Copy all elements from old to new")
	fmt.Println("  3. GC needs to clean up old array")
	fmt.Println("  4. Memory is fragmented")
}

func explainSliceInternals() {
	fmt.Println("Slice is a 24-byte struct (on 64-bit):")
	fmt.Println("  - Pointer to array: 8 bytes")
	fmt.Println("  - Length: 8 bytes")
	fmt.Println("  - Capacity: 8 bytes")
	fmt.Println()

	fmt.Println("Growth Algorithm (until 1024 elements):")
	fmt.Println("  • Start capacity: 0")
	fmt.Println("  • If len < 1024: double capacity")
	fmt.Println("  • If len >= 1024: grow by 25%")
	fmt.Println()

	fmt.Println("📈 CAPACITY GROWTH TABLE:")
	fmt.Println("  Elements  | Final Capacity | Reallocations | Waste")
	fmt.Println("  ----------|----------------|---------------|------")

	capacities := []int{10, 100, 1000, 10000, 100000}
	for _, n := range capacities {
		cap, reallocs, waste := calculateGrowth(n)
		fmt.Printf("  %9d | %14d | %13d | %5.1f%%\n",
			n, cap, reallocs, float64(cap-n)/float64(cap)*100)
		_ = reallocs
		_ = waste
	}
}

func calculateGrowth(target int) (finalCap, reallocs, waste int) {
	cap := 0
	reallocs = 0

	for cap < target {
		oldCap := cap
		if cap == 0 {
			cap = 1
		} else if cap < 1024 {
			cap *= 2
		} else {
			cap = cap + cap/4 // 25% growth
		}
		if oldCap > 0 {
			reallocs++
		}
	}

	return cap, reallocs, cap - target
}

func demoUserProcessing() {
	fmt.Println("Scenario: Processing 100K users from database")
	fmt.Println()

	fmt.Println("❌ ANTI-PATTERN (common in real code):")
	fmt.Println("```go")
	fmt.Println("func getUsersBad(db *sql.DB) ([]User, error) {")
	fmt.Println("    var users []User  // No capacity!")
	fmt.Println("    rows, _ := db.Query(\"SELECT * FROM users\")")
	fmt.Println("    for rows.Next() {")
	fmt.Println("        var user User")
	fmt.Println("        rows.Scan(&user)")
	fmt.Println("        users = append(users, user) // Reallocates!")
	fmt.Println("    }")
	fmt.Println("    return users")
	fmt.Println("}")
	fmt.Println("```")

	fmt.Println("\n✅ OPTIMIZED VERSION:")
	fmt.Println("```go")
	fmt.Println("func getUsersGood(db *sql.DB) ([]User, error) {")
	fmt.Println("    // First, count users")
	fmt.Println("    var count int")
	fmt.Println("    db.QueryRow(\"SELECT COUNT(*) FROM users\").Scan(&count)")
	fmt.Println("    ")
	fmt.Println("    // Pre-allocate exact capacity")
	fmt.Println("    users := make([]User, 0, count)")
	fmt.Println("    ")
	fmt.Println("    rows, _ := db.Query(\"SELECT * FROM users\")")
	fmt.Println("    for rows.Next() {")
	fmt.Println("        var user User")
	fmt.Println("        rows.Scan(&user)")
	fmt.Println("        users = append(users, user) // No reallocations!")
	fmt.Println("    }")
	fmt.Println("    return users")
	fmt.Println("}")
	fmt.Println("```")

	fmt.Println("\n📊 BENEFITS:")
	fmt.Println("  • Zero reallocations")
	fmt.Println("  • Predictable memory usage")
	fmt.Println("  • Better cache locality")
	fmt.Println("  • Reduced GC pressure")
}

// ========== COST ANALYSIS ==========

func calculateCostImpact(t1, t2 time.Duration, alloc1, alloc2 int) {
	// Calculate time savings
	timeSavedNs := t1.Nanoseconds() - t2.Nanoseconds()
	timeSavedPercent := float64(timeSavedNs) / float64(t1.Nanoseconds()) * 100

	// Calculate allocation savings
	allocSaved := alloc1 - alloc2
	allocSavedPercent := float64(allocSaved) / float64(alloc1) * 100

	fmt.Println("📈 PERFORMANCE IMPROVEMENT:")
	fmt.Printf("  Time:       %v → %v (%.1f%% faster)\n", t1, t2, timeSavedPercent)
	fmt.Printf("  Allocations: %d → %d (%.1f%% reduction)\n", alloc1, alloc2, allocSavedPercent)

	// Cloud cost calculation
	fmt.Println("\n☁️  CLOUD COST CALCULATION:")

	// Assumptions
	requestsPerSecond := 100.0
	requestsPerDay := requestsPerSecond * 3600 * 24
	awsCostPerVCPUHour := 0.0416                            // t3.medium
	msSavedPerRequest := float64(timeSavedNs) / 1_000_000.0 // Convert ns to ms

	fmt.Println("Assumptions:")
	fmt.Printf("  • Requests per second: %.0f\n", requestsPerSecond)
	fmt.Printf("  • AWS t3.medium: $%.4f/hour per vCPU\n", awsCostPerVCPUHour)
	fmt.Printf("  • Time saved per request: %.3f ms\n", msSavedPerRequest)

	// CPU time saved per day (in hours)
	cpuSecondsSavedPerRequest := float64(timeSavedNs) / 1_000_000_000.0
	cpuHoursSavedPerDay := cpuSecondsSavedPerRequest * requestsPerDay / 3600

	// Cost savings
	dailySavings := cpuHoursSavedPerDay * awsCostPerVCPUHour
	monthlySavings := dailySavings * 30
	annualSavings := monthlySavings * 12

	fmt.Println("\n💰 CALCULATED SAVINGS:")
	fmt.Printf("  CPU time saved per day: %.4f hours\n", cpuHoursSavedPerDay)
	fmt.Printf("  Daily savings:          $%.4f\n", dailySavings)
	fmt.Printf("  Monthly savings:        $%.4f\n", monthlySavings)
	fmt.Printf("  Annual savings:         $%.4f\n", annualSavings)

	// Additional benefits
	fmt.Println("\n🎯 ADDITIONAL BENEFITS (not quantified):")
	fmt.Println("  1. Reduced GC Pressure:")
	fmt.Println("     • Fewer allocations → less work for garbage collector")
	fmt.Println("     • Lower CPU usage during GC pauses")
	fmt.Println("     • More predictable latency")

	fmt.Println("\n  2. Better Cache Performance:")
	fmt.Println("     • Contiguous memory → better cache locality")
	fmt.Println("     • Reduced cache misses → faster execution")

	fmt.Println("\n  3. Memory Fragmentation:")
	fmt.Println("     • Pre-allocated slices use contiguous memory")
	fmt.Println("     • Random allocations can fragment memory")
	fmt.Println("     • Fragmentation reduces available memory")

	// Practical recommendations
	fmt.Println("\n📝 PRACTICAL RECOMMENDATIONS:")
	fmt.Println("  1. Always use make() with capacity when size is known")
	fmt.Println("  2. For unknown sizes, estimate and add buffer (e.g., capacity = estimate * 1.5)")
	fmt.Println("  3. Use arrays when size is fixed at compile time")
	fmt.Println("  4. Consider sync.Pool for frequently allocated/deallocated slices")
	fmt.Println("  5. Profile with -benchmem to see allocation counts")

	// Code patterns to look for
	fmt.Println("\n🔍 CODE PATTERNS TO OPTIMIZE:")
	fmt.Println("  • Look for: var slice []T (no capacity)")
	fmt.Println("  • Especially in hot paths (loops, API handlers)")
	fmt.Println("  • Database query results processing")
	fmt.Println("  • JSON/XML unmarshaling loops")
}
