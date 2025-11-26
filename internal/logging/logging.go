package logging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
)

type ctxKey string

const (
	traceKey    ctxKey = "trace_id"
	spanKey     ctxKey = "span_id"
	spanNameKey ctxKey = "span_name"
)

type TraceMeta struct {
	TraceID  string
	SpanID   string
	SpanName string
}

var (
	debugEnabled = true
	colorEnabled = os.Getenv("NO_COLOR") == ""

	green  = "\033[32m"
	yellow = "\033[33m"
	red    = "\033[31m"
	cyan   = "\033[36m"
	reset  = "\033[0m"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	if v := os.Getenv("MYDB_DEBUG"); v == "0" || v == "false" {
		debugEnabled = false
	}
}

func EnableDebug(on bool) {
	debugEnabled = on
}

// WithTrace ensures the context has trace/span IDs, generating new ones if missing.
func WithTrace(ctx context.Context) (context.Context, TraceMeta) {
	tr, sp, sn, ok := traceFromFull(ctx)
	if !ok {
		tr = newTraceID()
		sp = newSpanID()
		sn = "root"
		ctx = context.WithValue(ctx, traceKey, tr)
		ctx = context.WithValue(ctx, spanKey, sp)
		ctx = context.WithValue(ctx, spanNameKey, sn)
	}
	return ctx, TraceMeta{TraceID: tr, SpanID: sp, SpanName: sn}
}

// NewSpan creates a new child span within the same trace.
// It generates a new span ID while preserving the trace ID.
func NewSpan(ctx context.Context, spanName string) context.Context {
	_, _, _, ok := traceFromFull(ctx)
	if !ok {
		// If no trace exists, create a new one
		ctx, _ = WithTrace(ctx)
	}

	// Get trace ID to preserve it (create new span with same trace)
	tr, _, _, _ := traceFromFull(ctx)

	// Create new span ID but keep the same trace ID
	sp := newSpanID()
	ctx = context.WithValue(ctx, traceKey, tr)
	ctx = context.WithValue(ctx, spanKey, sp)
	ctx = context.WithValue(ctx, spanNameKey, spanName)
	return ctx
}

// TraceFrom extracts trace/span IDs from context.
func TraceFrom(ctx context.Context) (string, string, bool) {
	tr, ok1 := ctx.Value(traceKey).(string)
	sp, ok2 := ctx.Value(spanKey).(string)
	return tr, sp, ok1 && ok2
}

// traceFromFull extracts trace ID, span ID, and span name from context.
func traceFromFull(ctx context.Context) (string, string, string, bool) {
	tr, ok1 := ctx.Value(traceKey).(string)
	sp, ok2 := ctx.Value(spanKey).(string)
	sn, ok3 := ctx.Value(spanNameKey).(string)
	return tr, sp, sn, ok1 && ok2 && ok3
}

// SpanFrom extracts complete trace metadata from context.
func SpanFrom(ctx context.Context) (TraceMeta, bool) {
	tr, sp, sn, ok := traceFromFull(ctx)
	return TraceMeta{TraceID: tr, SpanID: sp, SpanName: sn}, ok
}

func Infof(format string, args ...any) {
	log.Print(prefix("", "INFO", green), fmt.Sprintf(format, args...))
}

func Warnf(format string, args ...any) {
	log.Print(prefix("", "WARN", yellow), fmt.Sprintf(format, args...))
}

func Errorf(format string, args ...any) {
	log.Print(prefix("", "ERROR", red), fmt.Sprintf(format, args...))
}

func Debugf(format string, args ...any) {
	if !debugEnabled {
		return
	}
	log.Print(prefix("", "DEBUG", cyan), fmt.Sprintf(format, args...))
}

func InfoContext(ctx context.Context, format string, args ...any) {
	log.Print(prefix(ctxPrefix(ctx), "INFO", green), fmt.Sprintf(format, args...))
}

func WarnContext(ctx context.Context, format string, args ...any) {
	log.Print(prefix(ctxPrefix(ctx), "WARN", yellow), fmt.Sprintf(format, args...))
}

func ErrorContext(ctx context.Context, format string, args ...any) {
	log.Print(prefix(ctxPrefix(ctx), "ERROR", red), fmt.Sprintf(format, args...))
}

func DebugContext(ctx context.Context, format string, args ...any) {
	if !debugEnabled {
		return
	}
	log.Print(prefix(ctxPrefix(ctx), "DEBUG", cyan), fmt.Sprintf(format, args...))
}

func ctxPrefix(ctx context.Context) string {
	tr, sp, sn, ok := traceFromFull(ctx)
	if !ok {
		return ""
	}
	if sn != "" {
		return fmt.Sprintf("[trace=%s|span=%s:%s] ", tr, sp, sn)
	}
	return fmt.Sprintf("[trace=%s|span=%s] ", tr, sp)
}

func prefix(trace, level, color string) string {
	out := ""
	if colorEnabled {
		out += color + level + reset + " "
	} else {
		out += level + " "
	}
	if trace != "" {
		out += trace
	}
	return out
}

// newTraceID returns a 16-byte (32 hex chars) trace ID matching OpenTelemetry size.
func newTraceID() string { return randomHex(16) }

// newSpanID returns an 8-byte (16 hex chars) span ID matching OpenTelemetry size.
func newSpanID() string { return randomHex(8) }

func randomHex(nBytes int) string {
	buf := make([]byte, nBytes)
	if _, err := rand.Read(buf); err != nil {
		// In the unlikely event of RNG failure, fall back to a predictable but unique-ish value.
		return fmt.Sprintf("fallback-%d", nBytes)
	}
	return hex.EncodeToString(buf)
}
