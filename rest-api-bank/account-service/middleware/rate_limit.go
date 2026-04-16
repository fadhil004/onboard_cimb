package middleware

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"account-service/config"
	"account-service/dto"
	"time"
)

func RateLimit(next http.Handler, domain string, feature string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		key := "bank:" + domain + ":rate_limit:" + feature+ ":ip:" + ip

		ctx := r.Context()

		count, err := config.RDB.Incr(ctx, key).Result()
		if err != nil {
			log.Println("Redis error:", err)
			next.ServeHTTP(w, r)
			return
		}

		if count == 1 {
			err := config.RDB.Expire(ctx, key, 5*time.Second).Err()
			if err != nil {
				log.Println("Redis expire error:", err)
			}
		}

		if count > 5 {
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(dto.BaseResponse{
				ResponseCode: "429",
				ResponseDesc: "Too Many Requests",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}
