package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

type contextKey string

const (
	traceIDKey  contextKey = "trace_id"
	updateIDKey contextKey = "update_id"
)

func EnsureTraceID(ctx context.Context) context.Context {
	if TraceID(ctx) != "" {
		return ctx
	}
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return context.WithValue(ctx, traceIDKey, "trace-unavailable")
	}
	return context.WithValue(ctx, traceIDKey, hex.EncodeToString(buf[:]))
}

func WithUpdateID(ctx context.Context, updateID int64) context.Context {
	return context.WithValue(ctx, updateIDKey, updateID)
}

func TraceID(ctx context.Context) string {
	if v, ok := ctx.Value(traceIDKey).(string); ok {
		return v
	}
	return ""
}

func LogAttrs(ctx context.Context, attrs ...any) []any {
	out := make([]any, 0, len(attrs)+4)
	if traceID := TraceID(ctx); traceID != "" {
		out = append(out, "trace_id", traceID)
	}
	if updateID, ok := ctx.Value(updateIDKey).(int64); ok && updateID != 0 {
		out = append(out, "update_id", updateID)
	}
	out = append(out, attrs...)
	return out
}
