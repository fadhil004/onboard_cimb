package middleware

import (
	"context"
	"encoding/json"
	"microservices-bank/transaction-service/helper"
	"net/http"
	"time"
)

type SnapContext struct{ ServiceCode string }

type SnapResponseWriter struct {
	http.ResponseWriter
	ServiceCode string
	written     bool
}

type contextKey string

const snapKey contextKey = "snap"

func GetSnap(ctx context.Context) SnapContext {
	val := ctx.Value(snapKey)
	if val == nil {
		return SnapContext{}
	}
	return val.(SnapContext)
}

func SnapMiddleware(serviceCode string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			requiredHeaders := []string{"Authorization", "X-SIGNATURE", "X-PARTNER-ID", "X-EXTERNAL-ID", "CHANNEL-ID"}
			for _, header := range requiredHeaders {
				if r.Header.Get(header) == "" {
					WriteSnapError(w, serviceCode, helper.ErrMandatoryField, "Invalid Mandatory Field "+header)
					return
				}
			}
			ctx := context.WithValue(r.Context(), snapKey, SnapContext{ServiceCode: serviceCode})
			sw := &SnapResponseWriter{ResponseWriter: w, ServiceCode: serviceCode}
			next(sw, r.WithContext(ctx))
		}
	}
}

func WriteSnapError(w http.ResponseWriter, serviceCode string, err error, customMsg string) {
	code, msg, httpCode := helper.MapSnapError(err, serviceCode)
	if customMsg != "" {
		msg = customMsg
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-TIMESTAMP", time.Now().Format(time.RFC3339))
	w.WriteHeader(httpCode)
	json.NewEncoder(w).Encode(map[string]string{"responseCode": code, "responseMessage": msg})
}

func (sw *SnapResponseWriter) WriteJSON(data interface{}) {
	sw.Header().Set("Content-Type", "application/json")
	sw.Header().Set("X-TIMESTAMP", time.Now().Format(time.RFC3339))
	sw.WriteHeader(http.StatusOK)
	json.NewEncoder(sw).Encode(data)
	sw.written = true
}
