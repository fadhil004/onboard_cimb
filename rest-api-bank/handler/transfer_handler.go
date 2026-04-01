package handler

import (
	"encoding/json"
	"net/http"

	"rest-api-bank/dto"
	"rest-api-bank/helper"
	"rest-api-bank/middleware"
	"rest-api-bank/service"

	"github.com/google/uuid"
)

type TransferHandler struct {
	mux *http.ServeMux
	Service *service.TransferService
}

func NewTransferHandler(mux *http.ServeMux, service *service.TransferService) *TransferHandler {
	return &TransferHandler{mux: mux, Service: service}
}

func (h *TransferHandler) MapRoutes() {
	h.mux.HandleFunc(helper.NewAPIPath(http.MethodPost, "/transfers"), middleware.RateLimit(h.Transfer(), "account", "transfer"))
}

func (h *TransferHandler) Transfer() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req dto.TransferRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(dto.BaseResponse{
				ResponseCode: "400",
				ResponseDesc: "Invalid Body",
			})
			return
		}

		idemKey := r.Header.Get("Idempotency-Key")
		if idemKey == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(dto.BaseResponse{
				ResponseCode: "400",
				ResponseDesc: "Idempotency-Key header required",
			})
			return
		}

		_, err := uuid.Parse(req.FromAccountID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(dto.BaseResponse{
				ResponseCode: "400",
				ResponseDesc: "Invalid from_account_id",
			})
			return
		}

		_, err = uuid.Parse(req.ToAccountID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(dto.BaseResponse{
				ResponseCode: "400",
				ResponseDesc: "Invalid to_account_id",
			})
			return
		}

		result, err := h.Service.Transfer(ctx, idemKey, req)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(dto.BaseResponse{
				ResponseCode: "500",
				ResponseDesc: err.Error(),
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(dto.BaseResponse{
			ResponseCode: "200",
			ResponseDesc: result.Message,
		})
	}
}