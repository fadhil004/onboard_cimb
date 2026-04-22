package middleware

import (
	"encoding/json"
	"log"
	"microservices-bank/transaction-service/config"
	"microservices-bank/transaction-service/dto"
	"net"
	"net/http"
	"time"
)

func RateLimit(next http.Handler, domain string, feature string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identifier := r.Header.Get("X-PARTNER-ID")
		if identifier == "" {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}
			identifier = ip
		}

		key := "bank:" + domain + ":rate_limit:" + feature + ":partner:" + identifier

		ctx := r.Context()

		count, err := config.RDB.Incr(ctx, key).Result()
		if err != nil {
			log.Println("Redis error:", err)
			next.ServeHTTP(w, r)
			return
		}
		if count == 1 {
			config.RDB.Expire(ctx, key, 5*time.Second)
		}
		if count > 5 {
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(dto.BaseResponse{ResponseCode: "429", ResponseDesc: "Too Many Requests"})
			return
		}
		next.ServeHTTP(w, r)
	})
}