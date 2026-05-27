// Package string_building demonstrates and benchmarks efficient string
// concatenation techniques in Go, comparing + operator, fmt.Sprintf,
// strings.Builder, and bytes.Buffer approaches.
package string_building

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

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
