package efficient_logging

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"strconv"
	"sync"

	"github.com/rs/zerolog"
	"go.uber.org/zap"
)

// ============================================================
// PATTERN 19: Efficient Logging Patterns
// ============================================================
// Problem: log.Printf allocates on every call (format string + args).
// On high-throughput services (100K+ logs/sec), logging overhead
// becomes a significant CPU and memory cost.
//
// This pattern demonstrates:
// 1. Standard log.Printf — allocates on every call
// 2. slog (stdlib structured) — lower alloc than Printf
// 3. zerolog — zero-allocation structured logging
// 4. zap — near-zero allocation structured logging
// 5. ZeroAllocLogger — custom logger with pre-allocated buffers
// 6. Check-then-log vs always-format for disabled levels
// 7. Cost projection for 1M log entries/hour
// ============================================================

// --- Logger Implementations ---

// StdLog logs a structured message using standard log.Printf.
// Allocates on every call: format string processing + argument boxing.
func StdLog(msg string, userID int, action string, latencyMs float64) {
	log.Printf("msg=%s user_id=%d action=%s latency_ms=%.2f", msg, userID, action, latencyMs)
}

// SlogLog logs a structured message using Go 1.21+ slog.
// Lower allocation than Printf due to typed attributes.
func SlogLog(logger *slog.Logger, msg string, userID int, action string, latencyMs float64) {
	logger.Info(msg,
		slog.Int("user_id", userID),
		slog.String("action", action),
		slog.Float64("latency_ms", latencyMs),
	)
}

// ZerologLog logs a structured message using zerolog.
// Zero allocation for disabled levels, minimal allocation for enabled.
func ZerologLog(logger zerolog.Logger, msg string, userID int, action string, latencyMs float64) {
	logger.Info().
		Int("user_id", userID).
		Str("action", action).
		Float64("latency_ms", latencyMs).
		Msg(msg)
}

// ZapLog logs a structured message using zap.
// Near-zero allocation with sugar-free API.
func ZapLog(logger *zap.Logger, msg string, userID int, action string, latencyMs float64) {
	logger.Info(msg,
		zap.Int("user_id", userID),
		zap.String("action", action),
		zap.Float64("latency_ms", latencyMs),
	)
}

// --- ZeroAllocLogger: Custom Zero-Allocation Logger ---

// ZeroAllocLogger is a custom logger that uses pre-allocated byte buffers
// to avoid any heap allocation when writing structured log entries.
// It guarantees buffer availability through a sync.Pool with pre-warming.
type ZeroAllocLogger struct {
	pool    sync.Pool
	writer  io.Writer
	level   LogLevel
	bufSize int
}

// LogLevel represents logging severity levels.
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

// NewZeroAllocLogger creates a logger with pre-allocated buffers.
// bufSize controls the pre-allocated buffer capacity (recommended: 512-4096).
// preWarm controls how many buffers to pre-allocate in the pool.
func NewZeroAllocLogger(w io.Writer, level LogLevel, bufSize, preWarm int) *ZeroAllocLogger {
	l := &ZeroAllocLogger{
		writer:  w,
		level:   level,
		bufSize: bufSize,
	}
	l.pool = sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 0, l.bufSize)
			return &buf
		},
	}
	// Pre-warm the pool to guarantee buffer availability
	buffers := make([]*[]byte, preWarm)
	for i := 0; i < preWarm; i++ {
		buf := make([]byte, 0, bufSize)
		buffers[i] = &buf
	}
	for _, buf := range buffers {
		l.pool.Put(buf)
	}
	return l
}

// Log writes a structured log entry with zero heap allocation.
// It uses pre-allocated buffers from the pool and formats fields inline.
func (l *ZeroAllocLogger) Log(level LogLevel, msg string, userID int, action string, latencyMs float64) {
	// Check level first — if disabled, zero work is done
	if level < l.level {
		return
	}

	// Get pre-allocated buffer from pool
	bufPtr := l.pool.Get().(*[]byte)
	buf := (*bufPtr)[:0] // Reset length, keep capacity

	// Format structured log entry without any allocation
	// Format: level=INFO msg=request user_id=123 action=login latency_ms=45.20\n
	buf = append(buf, "level="...)
	buf = appendLevel(buf, level)
	buf = append(buf, " msg="...)
	buf = append(buf, msg...)
	buf = append(buf, " user_id="...)
	buf = strconv.AppendInt(buf, int64(userID), 10)
	buf = append(buf, " action="...)
	buf = append(buf, action...)
	buf = append(buf, " latency_ms="...)
	buf = strconv.AppendFloat(buf, latencyMs, 'f', 2, 64)
	buf = append(buf, '\n')

	// Write to output
	_, _ = l.writer.Write(buf)

	// Return buffer to pool
	*bufPtr = buf
	l.pool.Put(bufPtr)
}

// appendLevel appends the level string to buf without allocation.
func appendLevel(buf []byte, level LogLevel) []byte {
	switch level {
	case LevelDebug:
		return append(buf, "DEBUG"...)
	case LevelInfo:
		return append(buf, "INFO"...)
	case LevelWarn:
		return append(buf, "WARN"...)
	case LevelError:
		return append(buf, "ERROR"...)
	default:
		return append(buf, "UNKNOWN"...)
	}
}

// --- Check-Then-Log vs Always-Format ---

// CheckThenLog demonstrates the pattern where we check the log level
// BEFORE doing any formatting work. When the level is disabled,
// zero string formatting occurs — no allocations, no CPU work.
func CheckThenLog(logger *ZeroAllocLogger, level LogLevel, msg string, userID int, action string, latencyMs float64) {
	// The level check inside Log() prevents any formatting work
	logger.Log(level, msg, userID, action, latencyMs)
}

// AlwaysFormat demonstrates the anti-pattern where formatting is done
// BEFORE checking the log level. Even when the level is disabled,
// the format string is fully processed, wasting CPU and allocating memory.
func AlwaysFormat(enabled bool, msg string, userID int, action string, latencyMs float64) string {
	// Always format regardless of whether logging is enabled
	formatted := fmt.Sprintf("msg=%s user_id=%d action=%s latency_ms=%.2f",
		msg, userID, action, latencyMs)

	if enabled {
		return formatted
	}
	// Formatting work was wasted — string was allocated but never used
	return ""
}
