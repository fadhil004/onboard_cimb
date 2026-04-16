package helper

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math/rand"
	"strings"
	"time"

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

func NewAPIPath(method string, path string) string {
	return fmt.Sprintf("%s %s", method, path)
}

func NormalizePath(path string) string {
	if strings.Contains(path, "/accounts/") {
		return "/accounts/{id}"
	}
	if strings.Contains(path, "/registration-account-creation") {
		return "/registration-account-creation"
	}
	if strings.Contains(path, "/balance/") {
		return "/balance/{accountNumber}"
	}
	return path
}

func GenerateAccountNumber() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("0888%012d", r.Int63n(1_000_000_000_000))
}

func GenerateAuthCode(partnerRefNo, accountID string) string {
	seed := partnerRefNo + accountID + fmt.Sprintf("%d", time.Now().UnixNano())
	h := sha256.New()
	h.Write([]byte(seed))
	first := fmt.Sprintf("%x", h.Sum(nil))
	h.Reset()
	h.Write([]byte(first + seed))
	return first + fmt.Sprintf("%x", h.Sum(nil))[:64]
}

func GenerateAPIKey() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("%04d-%04d-%010d",
		r.Intn(10000),
		r.Intn(10000),
		r.Int63n(10_000_000_000),
	)
}
