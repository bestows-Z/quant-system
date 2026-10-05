package trace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

type contextKey string

const TraceIDKey contextKey = "trace_id"

func NewTraceID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "trace-unavailable"
	}
	return hex.EncodeToString(b[:])
}

func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

func FromContext(ctx context.Context) string {
	if v, ok := ctx.Value(TraceIDKey).(string); ok {
		return v
	}
	return ""
}
