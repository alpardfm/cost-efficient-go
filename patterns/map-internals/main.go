package main

import (
	"fmt"
	"runtime"
	"strings"
	"time"
	"unsafe"
)

func main() {
	fmt.Println("🔬 DAY 3: Map Internals & Memory Overhead")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("📅 Date: %s\n\n", time.Now().Format("2006-01-02"))

	// The shocking truth about maps
	fmt.Println("🎯 SHOCKING DISCOVERY: Maps are MEMORY HOGS!")
	fmt.Println(strings.Repeat("-", 40))
	revealMapOverhead()

	// Benchmark: Map vs Slice vs Struct
	fmt.Println("\n📊 BENCHMARK: Map vs Alternatives")
	fmt.Println(strings.Repeat("-", 40))
	runComparisonBenchmarks()

	// Map internals deep dive
	fmt.Println("\n🔧 MAP INTERNALS DEEP DIVE")
	fmt.Println(strings.Repeat("-", 40))
	explainMapInternals()

	// Real-world scenarios
	fmt.Println("\n💼 REAL-WORLD USE CASES")
	fmt.Println(strings.Repeat("-", 40))
	analyzeRealWorldScenarios()

	// Optimization strategies
	fmt.Println("\n⚡ OPTIMIZATION STRATEGIES")
	fmt.Println(strings.Repeat("-", 40))
	shareOptimizationStrategies()

	// Cost analysis
	fmt.Println("\n💰 COST IMPACT ANALYSIS")
	fmt.Println(strings.Repeat("=", 60))
	calculateMapCostImpact()

	fmt.Println("\n✅ DAY 3 COMPLETED! 🎉")
	fmt.Println("\n🔜 Next: Day 4 - JSON Processing Efficiency")
}

func revealMapOverhead() {
	fmt.Println("Empty map sizes (64-bit system):")
	fmt.Println()

	// Measure empty maps
	var (
		m1 map[int]int
		m2 map[string]string
		m3 map[string]interface{}
		m4 map[int]struct{}
	)

	fmt.Printf("map[int]int:           %4d bytes\n", int(unsafe.Sizeof(m1)))
	fmt.Printf("map[string]string:     %4d bytes\n", int(unsafe.Sizeof(m2)))
	fmt.Printf("map[string]interface{}:%4d bytes\n", int(unsafe.Sizeof(m3)))
	fmt.Printf("map[int]struct{}:      %4d bytes\n", int(unsafe.Sizeof(m4)))

	fmt.Println("\n💡 Wait... these are just POINTERS (8 bytes)!")
	fmt.Println("The REAL overhead is in the heap-allocated hash table.")

	// Create maps and measure memory
	fmt.Println("\n📏 Actual memory usage with data:")
	measureMapMemory()
}

func measureMapMemory() {
	// Force GC and measure baseline
	runtime.GC()
	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Create map with 1000 entries
	m := make(map[int]string, 1000)
	for i := 0; i < 1000; i++ {
		m[i] = fmt.Sprintf("value_%d", i)
	}

	runtime.ReadMemStats(&m2)

	// Calculate memory used
	mapMemory := m2.TotalAlloc - m1.TotalAlloc
	expectedMemory := 1000 * (8 + 16) // key + value

	fmt.Printf("Map with 1000 int→string entries:\n")
	fmt.Printf("  Actual memory:   %8d bytes\n", mapMemory)
	fmt.Printf("  Expected (naive):%8d bytes\n", expectedMemory)
	fmt.Printf("  Overhead:        %8d bytes (%.1fx!)\n",
		float64(mapMemory)-float64(expectedMemory),
		float64(mapMemory)/float64(expectedMemory))

	// Compare with slice of structs
	runtime.GC()
	runtime.ReadMemStats(&m1)

	type Entry struct {
		Key   int
		Value string
	}
	slice := make([]Entry, 0, 1000)
	for i := 0; i < 1000; i++ {
		slice = append(slice, Entry{Key: i, Value: fmt.Sprintf("value_%d", i)})
	}

	runtime.ReadMemStats(&m2)
	sliceMemory := m2.TotalAlloc - m1.TotalAlloc

	fmt.Printf("\nSlice of structs (same data):\n")
	fmt.Printf("  Actual memory:   %8d bytes\n", sliceMemory)
	fmt.Printf("  Map vs Slice:    %8d bytes extra (%.1fx)\n",
		mapMemory-sliceMemory,
		float64(mapMemory)/float64(sliceMemory))
}

func runComparisonBenchmarks() {
	fmt.Println("Comparing data structures for 1000 key-value pairs:")
	fmt.Println()

	// Map benchmark
	start := time.Now()
	m := make(map[int]string)
	for i := 0; i < 1000; i++ {
		m[i] = fmt.Sprintf("value_%d", i)
	}
	mapTime := time.Since(start)

	// Slice of structs benchmark
	start = time.Now()
	type Entry struct {
		Key   int
		Value string
	}
	entries := make([]Entry, 0, 1000)
	for i := 0; i < 1000; i++ {
		entries = append(entries, Entry{Key: i, Value: fmt.Sprintf("value_%d", i)})
	}
	sliceTime := time.Since(start)

	// Parallel arrays benchmark
	start = time.Now()
	keys := make([]int, 0, 1000)
	values := make([]string, 0, 1000)
	for i := 0; i < 1000; i++ {
		keys = append(keys, i)
		values = append(values, fmt.Sprintf("value_%d", i))
	}
	arrayTime := time.Since(start)

	fmt.Printf("1. Map[int]string:          %v\n", mapTime)
	fmt.Printf("2. Slice of structs:        %v (%.1fx faster)\n",
		sliceTime, float64(mapTime.Nanoseconds())/float64(sliceTime.Nanoseconds()))
	fmt.Printf("3. Parallel arrays:         %v (%.1fx faster)\n",
		arrayTime, float64(mapTime.Nanoseconds())/float64(arrayTime.Nanoseconds()))

	// Memory comparison
	fmt.Println("\n💾 Memory efficiency (lower is better):")
	fmt.Printf("  Map overhead per entry:    ~50 bytes\n")
	fmt.Printf("  Slice of structs overhead: ~0 bytes (exact size)\n")
	fmt.Printf("  Memory ratio: Map uses ~3-10x more memory!\n")
}

func explainMapInternals() {
	fmt.Println("Go map is a hash table with buckets:")
	fmt.Println()

	fmt.Println("map[int]string memory layout per entry:")
	fmt.Println("┌────────────┬────────────┬────────────┬────────────┐")
	fmt.Println("│   Key (8)  │  Value(16) │ Next*(8)   │  Overflow  │")
	fmt.Println("│            │            │            │   header   │")
	fmt.Println("└────────────┴────────────┴────────────┴────────────┘")
	fmt.Println("  Total: ~40-50 bytes per entry (excluding strings)")
	fmt.Println()

	fmt.Println("📈 MAP GROWTH PATTERN:")
	fmt.Println("  • Initial: 1 bucket (8 entries)")
	fmt.Println("  • Load factor > 6.5: double buckets")
	fmt.Println("  • Each growth: rehash ALL entries")
	fmt.Println("  • Memory waste: ~25% on average")
	fmt.Println()

	fmt.Println("⚡ PERFORMANCE CHARACTERISTICS:")
	fmt.Println("  • O(1) average lookups (but with high constant factor)")
	fmt.Println("  • Memory locality: POOR (random access)")
	fmt.Println("  • Cache misses: HIGH")
	fmt.Println("  • GC pressure: HIGH (many small objects)")
}

func analyzeRealWorldScenarios() {
	fmt.Println("When to use maps (✅) vs alternatives (❌):")
	fmt.Println()

	fmt.Println("✅ GOOD MAP USE CASES:")
	fmt.Println("  1. Configuration lookup (small, static)")
	fmt.Println("     config := map[string]string{\"port\": \"8080\"}")
	fmt.Println("  2. Set operations (map[T]struct{})")
	fmt.Println("     seen := make(map[string]struct{})")
	fmt.Println("  3. Frequency counting (map[T]int)")
	fmt.Println("     freq := make(map[string]int)")
	fmt.Println("  4. Sparse data (lots of missing keys)")
	fmt.Println()

	fmt.Println("❌ BAD MAP USE CASES (use slices instead):")
	fmt.Println("  1. Iterating all elements frequently")
	fmt.Println("     ❌ map[int]User → ✅ []User")
	fmt.Println("  2. Small, fixed number of fields")
	fmt.Println("     ❌ map[string]interface{} → ✅ struct")
	fmt.Println("  3. Sequential integer keys (0,1,2,3...)")
	fmt.Println("     ❌ map[int]Data → ✅ []Data")
	fmt.Println("  4. Memory-constrained environments")
	fmt.Println()

	fmt.Println("🔍 REAL EXAMPLE: User ID → Name lookup")
	fmt.Println("Requirements: 1M users, need O(1) lookup by ID")
	fmt.Println()

	fmt.Println("Option A: map[int]string")
	fmt.Println("  • Memory: ~50MB (50 bytes × 1M)")
	fmt.Println("  • Lookup: O(1), fast")
	fmt.Println("  • Iteration: slow, random order")
	fmt.Println()

	fmt.Println("Option B: []string with binary search")
	fmt.Println("  • Memory: ~16MB (16 bytes × 1M)")
	fmt.Println("  • Lookup: O(log n), still fast")
	fmt.Println("  • Iteration: fast, sequential")
	fmt.Println("  • Bonus: cache-friendly, less GC")
}

func shareOptimizationStrategies() {
	fmt.Println("1. 🎯 PRE-ALLOCATE MAPS (like slices!)")
	fmt.Println("   ❌ m := make(map[int]string)")
	fmt.Println("   ✅ m := make(map[int]string, expectedSize)")
	fmt.Println("   Benefit: Avoids rehashing, saves 25% memory")
	fmt.Println()

	fmt.Println("2. 🏗️ USE map[T]struct{} FOR SETS")
	fmt.Println("   ❌ set := make(map[string]bool)")
	fmt.Println("   ✅ set := make(map[string]struct{})")
	fmt.Println("   Benefit: 0 bytes value vs 1 byte bool")
	fmt.Println()

	fmt.Println("3. 🔄 REUSE MAPS WITH CLEAR()")
	fmt.Println("   ❌ m = make(map[int]string) // New allocation")
	fmt.Println("   ✅ clear(m) // Reuse existing")
	fmt.Println("   Benefit: No new allocation, less GC")
	fmt.Println()

	fmt.Println("4. 📦 USE SYNC.POOL FOR TEMPORARY MAPS")
	fmt.Println("   var mapPool = sync.Pool{")
	fmt.Println("       New: func() interface{} {")
	fmt.Println("           return make(map[int]string, 1000)")
	fmt.Println("       }")
	fmt.Println("   }")
	fmt.Println("   Benefit: Eliminates allocation in hot paths")
	fmt.Println()

	fmt.Println("5. 🚫 AVOID map[string]interface{} FOR CONFIGS")
	fmt.Println("   ❌ config := map[string]interface{}{...}")
	fmt.Println("   ✅ type Config struct { Port int `json:\"port\"` }")
	fmt.Println("   Benefit: Type safety, less memory, faster access")
}

func calculateMapCostImpact() {
	fmt.Println("📈 MAP OVERHEAD CALCULATION:")

	// Constants
	mapEntryOverhead := 50.0   // bytes per map entry
	sliceEntryOverhead := 16.0 // bytes per slice entry (int + string)
	entries := 1_000_000.0     // 1 million entries
	awsCostPerGBMonth := 3.75  // $/GB-month

	fmt.Printf("Scenario: Storing 1M user ID → name mappings\n")
	fmt.Printf("Each entry: int key + string value (~16 bytes data)\n\n")

	// Map memory
	mapMemoryGB := (entries * mapEntryOverhead) / (1024 * 1024 * 1024)
	mapCost := mapMemoryGB * awsCostPerGBMonth

	// Slice memory
	sliceMemoryGB := (entries * sliceEntryOverhead) / (1024 * 1024 * 1024)
	sliceCost := sliceMemoryGB * awsCostPerGBMonth

	// Savings
	savingsGB := mapMemoryGB - sliceMemoryGB
	savingsCost := mapCost - sliceCost

	fmt.Printf("Memory Usage:\n")
	fmt.Printf("  Map[int]string:      %.2f GB\n", mapMemoryGB)
	fmt.Printf("  Slice of structs:    %.2f GB\n", sliceMemoryGB)
	fmt.Printf("  Map overhead:        %.2f GB (%.1fx!)\n",
		savingsGB, mapMemoryGB/sliceMemoryGB)

	fmt.Printf("\nMonthly AWS Cost (t3.medium):\n")
	fmt.Printf("  Map cost:            $%.2f\n", mapCost)
	fmt.Printf("  Slice cost:          $%.2f\n", sliceCost)
	fmt.Printf("  Monthly savings:     $%.2f\n", savingsCost)
	fmt.Printf("  Annual savings:      $%.2f\n", savingsCost*12)

	fmt.Printf("\n🚨 ADDITIONAL COSTS (not quantified):\n")
	fmt.Printf("  1. GC Pressure: Maps cause more frequent GC\n")
	fmt.Printf("  2. CPU Cache Misses: Poor locality → slower execution\n")
	fmt.Printf("  3. Memory Fragmentation: Random allocations\n")
	fmt.Printf("  4. Iteration Speed: 2-3x slower than slices\n")

	fmt.Printf("\n🎯 DECISION FRAMEWORK:\n")
	fmt.Printf("  Use Map when: O(1) lookup critical, data sparse\n")
	fmt.Printf("  Use Slice when: Iteration frequent, memory constrained\n")
	fmt.Printf("  Hybrid approach: Small map + large slice for different ops\n")
}
