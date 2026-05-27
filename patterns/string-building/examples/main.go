package main

import (
	"fmt"
	"strings"
	"time"

	sb "github.com/alpardfm/cost-efficient-go/patterns/string-building"
)

func main() {
	fmt.Println("🔬 String Building & Concatenation Efficiency")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("📅 Date: %s\n\n", time.Now().Format("2006-01-02"))

	// Problem demonstration
	fmt.Println("🎯 PROBLEM: String concatenation with + operator is O(n²)!")
	fmt.Println(strings.Repeat("-", 40))
	demoConcatenationProblem()

	// Benchmark comparisons at different sizes
	fmt.Println("\n📊 BENCHMARK COMPARISONS")
	fmt.Println(strings.Repeat("-", 40))

	sizes := []int{10, 50, 100, 500}
	for _, size := range sizes {
		fmt.Printf("\n--- %d concatenations ---\n", size)
		parts := generateTestStrings(size)
		benchmarkAllMethods(parts)
	}

	// Real-world use cases
	fmt.Println("\n\n💼 REAL-WORLD USE CASES")
	fmt.Println(strings.Repeat("=", 60))

	fmt.Println("\n1. Log Message Formatting:")
	fmt.Println(strings.Repeat("-", 40))
	demoLogMessage()

	fmt.Println("\n2. SQL Query Building:")
	fmt.Println(strings.Repeat("-", 40))
	demoSQLQuery()

	fmt.Println("\n3. JSON Response Construction:")
	fmt.Println(strings.Repeat("-", 40))
	demoJSONResponse()

	// Cost analysis
	fmt.Println("\n\n💰 COST IMPACT ANALYSIS")
	fmt.Println(strings.Repeat("=", 60))
	calculateCostImpact()

	fmt.Println("\n✅ PATTERN COMPLETED! 🎉")
}

// ========== DEMO FUNCTIONS ==========

func demoConcatenationProblem() {
	fmt.Println("String concatenation with + operator:")
	fmt.Println()
	fmt.Println("  s = s + \"a\"  → allocates new string every time!")
	fmt.Println("  10 concats  → 10 allocations, copies ~55 bytes total")
	fmt.Println("  100 concats → 100 allocations, copies ~5,050 bytes total")
	fmt.Println("  500 concats → 500 allocations, copies ~125,250 bytes total")
	fmt.Println()
	fmt.Println("  strings.Builder uses internal []byte buffer:")
	fmt.Println("  - Grows capacity 2x when needed (amortized O(1))")
	fmt.Println("  - Single allocation with Grow() pre-hint")
	fmt.Println("  - Final String() is zero-copy (unsafe pointer cast)")
}

func benchmarkAllMethods(parts []string) {
	methods := []struct {
		name string
		fn   func([]string) string
	}{
		{"+ operator   ", sb.ConcatPlus},
		{"fmt.Sprintf  ", sb.ConcatSprintf},
		{"strings.Builder", sb.ConcatBuilder},
		{"bytes.Buffer ", sb.ConcatBuffer},
	}

	for _, m := range methods {
		start := time.Now()
		iterations := 1000
		var result string
		for i := 0; i < iterations; i++ {
			result = m.fn(parts)
		}
		elapsed := time.Since(start)
		avgNs := elapsed.Nanoseconds() / int64(iterations)
		fmt.Printf("  %s: %8d ns/op (result len: %d)\n", m.name, avgNs, len(result))
	}
}

func demoLogMessage() {
	fields := map[string]string{
		"user_id":    "12345",
		"request_id": "abc-def-ghi",
		"method":     "POST",
		"path":       "/api/v1/users",
		"latency_ms": "42",
	}

	logLine := sb.FormatLogMessage(time.Now(), "INFO", "request completed", fields)
	fmt.Printf("  Output: %s\n", logLine)

	// Benchmark: Builder vs naive + concatenation
	start := time.Now()
	for i := 0; i < 10000; i++ {
		_ = sb.FormatLogMessage(time.Now(), "INFO", "request completed", fields)
	}
	builderTime := time.Since(start)

	start = time.Now()
	for i := 0; i < 10000; i++ {
		_ = formatLogMessageNaive(time.Now(), "INFO", "request completed", fields)
	}
	naiveTime := time.Since(start)

	fmt.Printf("  Builder: %v/10K ops | Naive: %v/10K ops\n", builderTime, naiveTime)
	fmt.Printf("  Speedup: %.1fx faster\n", float64(naiveTime.Nanoseconds())/float64(builderTime.Nanoseconds()))
}

func demoSQLQuery() {
	query := sb.BuildSQLQuery(
		"users",
		[]string{"id", "name", "email", "created_at"},
		[]string{"status = $1", "age > $2", "country = $3"},
		"created_at DESC",
		50,
	)
	fmt.Printf("  Output: %s\n", query)

	// Benchmark
	start := time.Now()
	for i := 0; i < 10000; i++ {
		_ = sb.BuildSQLQuery(
			"users",
			[]string{"id", "name", "email", "created_at"},
			[]string{"status = $1", "age > $2", "country = $3"},
			"created_at DESC",
			50,
		)
	}
	elapsed := time.Since(start)
	fmt.Printf("  Performance: %v/10K ops (%d ns/op)\n", elapsed, elapsed.Nanoseconds()/10000)
}

func demoJSONResponse() {
	data := map[string]string{
		"id":    "12345",
		"name":  "John Doe",
		"email": "john@example.com",
	}

	jsonStr := sb.ConstructJSONResponse(200, "success", data)
	fmt.Printf("  Output: %s\n", jsonStr)

	// Benchmark
	start := time.Now()
	for i := 0; i < 10000; i++ {
		_ = sb.ConstructJSONResponse(200, "success", data)
	}
	elapsed := time.Since(start)
	fmt.Printf("  Performance: %v/10K ops (%d ns/op)\n", elapsed, elapsed.Nanoseconds()/10000)
}

// formatLogMessageNaive is the naive version using + operator for comparison.
func formatLogMessageNaive(timestamp time.Time, level string, message string, fields map[string]string) string {
	result := "[" + timestamp.Format(time.RFC3339) + "] [" + level + "] " + message
	for key, value := range fields {
		result += " " + key + "=" + value
	}
	return result
}

// generateTestStrings creates a slice of test strings for benchmarking.
func generateTestStrings(count int) []string {
	parts := make([]string, count)
	for i := 0; i < count; i++ {
		parts[i] = fmt.Sprintf("part_%d_", i)
	}
	return parts
}

// ========== COST ANALYSIS ==========

func calculateCostImpact() {
	// Benchmark at 100 concatenations (requirement: 5x faster at 100+)
	parts := generateTestStrings(100)

	iterations := 5000
	start := time.Now()
	for i := 0; i < iterations; i++ {
		_ = sb.ConcatPlus(parts)
	}
	plusTime := time.Since(start)

	start = time.Now()
	for i := 0; i < iterations; i++ {
		_ = sb.ConcatBuilder(parts)
	}
	builderTime := time.Since(start)

	plusNsPerOp := plusTime.Nanoseconds() / int64(iterations)
	builderNsPerOp := builderTime.Nanoseconds() / int64(iterations)
	speedup := float64(plusNsPerOp) / float64(builderNsPerOp)

	fmt.Println("📈 PERFORMANCE AT 100 CONCATENATIONS:")
	fmt.Printf("  + operator:      %d ns/op\n", plusNsPerOp)
	fmt.Printf("  strings.Builder: %d ns/op\n", builderNsPerOp)
	fmt.Printf("  Speedup:         %.1fx faster\n", speedup)

	// Cost projection at scale: 10M log entries per day
	fmt.Println("\n☁️  COST PROJECTION (10M log entries/day):")
	fmt.Println("  Assumptions:")
	fmt.Println("  • Each log entry = ~5 field concatenations")
	fmt.Println("  • AWS t3.medium: $0.0416/vCPU-hour")
	fmt.Println("  • 10,000,000 log entries per day")

	// CPU time saved per day
	nsSavedPerOp := plusNsPerOp - builderNsPerOp
	opsPerDay := int64(10_000_000)
	totalNsSavedPerDay := nsSavedPerOp * opsPerDay
	cpuHoursSavedPerDay := float64(totalNsSavedPerDay) / 1e9 / 3600

	awsCostPerVCPUHour := 0.0416
	dailySavings := cpuHoursSavedPerDay * awsCostPerVCPUHour
	monthlySavings := dailySavings * 30
	annualSavings := monthlySavings * 12

	fmt.Printf("\n  CPU time saved per day:  %.4f vCPU-hours\n", cpuHoursSavedPerDay)
	fmt.Printf("  Daily savings:           $%.4f\n", dailySavings)
	fmt.Printf("  Monthly savings:         $%.2f\n", monthlySavings)
	fmt.Printf("  Annual savings:          $%.2f\n", annualSavings)

	// Memory savings
	fmt.Println("\n  Memory Impact:")
	bytesWastedPerPlusOp := int64(5000)
	bytesWastedPerBuilderOp := int64(0)
	memSavedPerOp := bytesWastedPerPlusOp - bytesWastedPerBuilderOp
	totalMemSavedPerDay := memSavedPerOp * opsPerDay

	fmt.Printf("  Memory saved per day:    %.2f GB (intermediate allocs avoided)\n",
		float64(totalMemSavedPerDay)/1024/1024/1024)
	fmt.Println("  GC pressure reduction:   Significant (fewer short-lived objects)")

	// Scale projections
	fmt.Println("\n  📊 SCALE PROJECTIONS:")
	scales := []struct {
		name string
		ops  int64
	}{
		{"1M ops/day", 1_000_000},
		{"10M ops/day", 10_000_000},
		{"100M ops/day", 100_000_000},
	}

	fmt.Printf("  %-15s | %-18s | %-12s | %-12s\n", "Scale", "CPU Hours Saved", "Monthly $", "Annual $")
	fmt.Printf("  %s\n", strings.Repeat("-", 65))
	for _, scale := range scales {
		cpuHours := float64(nsSavedPerOp*scale.ops) / 1e9 / 3600
		monthly := cpuHours * awsCostPerVCPUHour * 30
		annual := monthly * 12
		fmt.Printf("  %-15s | %14.4f hrs | $%10.2f | $%10.2f\n",
			scale.name, cpuHours, monthly, annual)
	}

	fmt.Println("\n📝 RECOMMENDATIONS:")
	fmt.Println("  1. Always use strings.Builder for 3+ concatenations")
	fmt.Println("  2. Use Grow() when total length is known or estimable")
	fmt.Println("  3. For log formatting: pre-build format with Builder")
	fmt.Println("  4. For SQL building: Builder with WriteString (avoid Sprintf in loop)")
	fmt.Println("  5. For JSON: manual Builder when encoding/json is too slow on hot paths")
}
