package logging

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"time"
)

type ctxKey string

const (
	traceKey ctxKey = "trace_id"
	spanKey  ctxKey = "span_id"
)

type TraceMeta struct {
	TraceID string
	SpanID  string
}

var (
	debugEnabled = true
	colorEnabled = os.Getenv("NO_COLOR") == ""
	seq          uint64

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
	tr, sp, ok := TraceFrom(ctx)
	if !ok {
		tr = newID()
		sp = newID()
		ctx = context.WithValue(ctx, traceKey, tr)
		ctx = context.WithValue(ctx, spanKey, sp)
	}
	return ctx, TraceMeta{TraceID: tr, SpanID: sp}
}

// TraceFrom extracts trace/span IDs from context.
func TraceFrom(ctx context.Context) (string, string, bool) {
	tr, ok1 := ctx.Value(traceKey).(string)
	sp, ok2 := ctx.Value(spanKey).(string)
	return tr, sp, ok1 && ok2
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
	tr, sp, ok := TraceFrom(ctx)
	if !ok {
		return ""
	}
	return fmt.Sprintf("[trace=%s span=%s] ", tr, sp)
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

func newID() string {
	val := atomic.AddUint64(&seq, 1)
	return fmt.Sprintf("%x-%x", time.Now().UnixNano(), val)
}
