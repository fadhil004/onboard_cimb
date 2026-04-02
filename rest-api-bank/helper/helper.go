package helper

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

func GetIDFromPath(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return ""
	}
	return span.SpanContext().TraceID().String()
}

func GetIDFromTransactionPath(path string) string {
	parts := strings.Split(path, "/")
	// /accounts/{id}/transactions → id di index 2
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}

func NewAPIPath(method string, path string) string {
	return fmt.Sprintf("%s %s", method, path)
}