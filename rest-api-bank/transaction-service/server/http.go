package server

import (
	"encoding/json"
	"microservices-bank/transaction-service/dto"
	"net/http"
)

func ApplicationMiddlewareResponse(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		h.ServeHTTP(w, r)
	}
}

func HandleRouteNotFound(mux *http.ServeMux) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h, pattern := mux.Handler(r)
		if pattern == "" {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(dto.BaseResponse{ResponseCode: "404", ResponseDesc: "Not found"})
			return
		}
		h.ServeHTTP(w, r)
	}
}
