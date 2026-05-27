package main

import (
	"bytes"
	"fmt"
	"strings"
	"time"
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

// ========== 4 CONCATENATION METHODS ==========

// ConcatPlus concatenates strings using the + operator.
// This is O(n²) because each + creates a new string allocation.
func ConcatPlus(parts []string) string {
	var result string
	for _, part := range parts {
		result += part
	}
	return result
}

// ConcatSprintf concatenates strings using fmt.Sprintf.
// Uses reflection internally, adding overhead per call.
func ConcatSprintf(parts []string) string {
	var result string
	for _, part := range parts {
		result = fmt.Sprintf("%s%s", result, part)
	}
	return result
}

// ConcatBuilder concatenates strings using strings.Builder.
// Optimized for incremental string building with minimal allocations.
func ConcatBuilder(parts []string) string {
	var builder strings.Builder
	// Pre-grow to avoid reallocations
	totalLen := 0
	for _, part := range parts {
		totalLen += len(part)
	}
	builder.Grow(totalLen)

	for _, part := range parts {
		builder.WriteString(part)
	}
	return builder.String()
}

// ConcatBuffer concatenates strings using bytes.Buffer.
// Similar to Builder but with additional read capabilities.
func ConcatBuffer(parts []string) string {
	var buf bytes.Buffer
	// Pre-grow to avoid reallocations
	totalLen := 0
	for _, part := range parts {
		totalLen += len(part)
	}
	buf.Grow(totalLen)

	for _, part := range parts {
		buf.WriteString(part)
	}
	return buf.String()
}

// ========== 3 REAL-WORLD USE CASES ==========

// FormatLogMessage builds a structured log line with timestamp, level, and fields.
// Uses strings.Builder for efficient incremental construction.
func FormatLogMessage(timestamp time.Time, level string, message string, fields map[string]string) string {
	var builder strings.Builder

	// Estimate capacity: timestamp(30) + level(10) + message + fields
	estimatedLen := 50 + len(message) + len(fields)*30
	builder.Grow(estimatedLen)

	// Format: [2024-01-15T10:30:00Z] [INFO] message key1=value1 key2=value2
	builder.WriteByte('[')
	builder.WriteString(timestamp.Format(time.RFC3339))
	builder.WriteString("] [")
	builder.WriteString(level)
	builder.WriteString("] ")
	builder.WriteString(message)

	for key, value := range fields {
		builder.WriteByte(' ')
		builder.WriteString(key)
		builder.WriteByte('=')
		builder.WriteString(value)
	}

	return builder.String()
}

// BuildSQLQuery builds a parameterized SQL query with dynamic WHERE clauses.
// Uses strings.Builder to efficiently construct complex queries.
func BuildSQLQuery(table string, columns []string, conditions []string, orderBy string, limit int) string {
	var builder strings.Builder

	// Estimate capacity
	estimatedLen := 50 + len(table) + len(orderBy)
	for _, col := range columns {
		estimatedLen += len(col) + 2
	}
	for _, cond := range conditions {
		estimatedLen += len(cond) + 5
	}
	builder.Grow(estimatedLen)

	// SELECT columns
	builder.WriteString("SELECT ")
	for i, col := range columns {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(col)
	}

	// FROM table
	builder.WriteString(" FROM ")
	builder.WriteString(table)

	// WHERE conditions
	if len(conditions) > 0 {
		builder.WriteString(" WHERE ")
		for i, cond := range conditions {
			if i > 0 {
				builder.WriteString(" AND ")
			}
			builder.WriteString(cond)
		}
	}

	// ORDER BY
	if orderBy != "" {
		builder.WriteString(" ORDER BY ")
		builder.WriteString(orderBy)
	}

	// LIMIT
	if limit > 0 {
		builder.WriteString(fmt.Sprintf(" LIMIT %d", limit))
	}

	return builder.String()
}

// ConstructJSONResponse builds a JSON response string manually.
// For cases where encoding/json reflection overhead is too expensive on hot paths.
func ConstructJSONResponse(status int, message string, data map[string]string) string {
	var builder strings.Builder

	// Estimate capacity: base structure + data entries
	estimatedLen := 80 + len(message) + len(data)*40
	builder.Grow(estimatedLen)

	builder.WriteString(`{"status":`)
	builder.WriteString(fmt.Sprintf("%d", status))
	builder.WriteString(`,"message":"`)
	builder.WriteString(escapeJSON(message))
	builder.WriteByte('"')

	if len(data) > 0 {
		builder.WriteString(`,"data":{`)
		first := true
		for key, value := range data {
			if !first {
				builder.WriteByte(',')
			}
			builder.WriteByte('"')
			builder.WriteString(escapeJSON(key))
			builder.WriteString(`":"`)
			builder.WriteString(escapeJSON(value))
			builder.WriteByte('"')
			first = false
		}
		builder.WriteByte('}')
	}

	builder.WriteByte('}')
	return builder.String()
}

// ========== HELPER FUNCTIONS ==========

// escapeJSON escapes special characters in JSON strings.
func escapeJSON(s string) string {
	var builder strings.Builder
	for _, r := range s {
		switch r {
		case '"':
			builder.WriteString(`\"`)
		case '\\':
			builder.WriteString(`\\`)
		case '\n':
			builder.WriteString(`\n`)
		case '\r':
			builder.WriteString(`\r`)
		case '\t':
			builder.WriteString(`\t`)
		default:
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

// generateTestStrings creates a slice of test strings for benchmarking.
func generateTestStrings(count int) []string {
	parts := make([]string, count)
	for i := 0; i < count; i++ {
		parts[i] = fmt.Sprintf("part_%d_", i)
	}
	return parts
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
		{"+ operator   ", ConcatPlus},
		{"fmt.Sprintf  ", ConcatSprintf},
		{"strings.Builder", ConcatBuilder},
		{"bytes.Buffer ", ConcatBuffer},
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

	logLine := FormatLogMessage(time.Now(), "INFO", "request completed", fields)
	fmt.Printf("  Output: %s\n", logLine)

	// Benchmark: Builder vs naive + concatenation
	start := time.Now()
	for i := 0; i < 10000; i++ {
		_ = FormatLogMessage(time.Now(), "INFO", "request completed", fields)
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
	query := BuildSQLQuery(
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
		_ = BuildSQLQuery(
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

	jsonStr := ConstructJSONResponse(200, "success", data)
	fmt.Printf("  Output: %s\n", jsonStr)

	// Benchmark
	start := time.Now()
	for i := 0; i < 10000; i++ {
		_ = ConstructJSONResponse(200, "success", data)
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

// ========== COST ANALYSIS ==========

func calculateCostImpact() {
	// Benchmark at 100 concatenations (requirement: 5x faster at 100+)
	parts := generateTestStrings(100)

	iterations := 5000
	start := time.Now()
	for i := 0; i < iterations; i++ {
		_ = ConcatPlus(parts)
	}
	plusTime := time.Since(start)

	start = time.Now()
	for i := 0; i < iterations; i++ {
		_ = ConcatBuilder(parts)
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
	// + operator at 100 concats: ~100 allocations, ~5KB wasted per op
	// Builder at 100 concats: 1-2 allocations, ~0 waste
	bytesWastedPerPlusOp := int64(5000) // approximate intermediate allocations
	bytesWastedPerBuilderOp := int64(0) // pre-grown, single allocation
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
