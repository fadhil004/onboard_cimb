package middleware

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"rest-api-bank/config"
	"rest-api-bank/dto"
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

		// INCREASE
		count, err := config.RDB.Incr(ctx, key).Result()
		if err != nil {
			// Redis error bukan berarti error aplikasi, jadi log saja dan lanjutkan request
			log.Println("Redis error:", err)
			next.ServeHTTP(w, r)
			return
		}

		// Set expire (TTL) saat pertama kali key dibuat
		if count == 1 {
			err := config.RDB.Expire(ctx, key, 5*time.Second).Err()
			if err != nil {
				log.Println("Redis expire error:", err)
			}
		}

		// Limit 5 request per 5 seconds
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