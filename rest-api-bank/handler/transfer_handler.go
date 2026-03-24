package handler

import (
	"encoding/json"
	"net/http"

	"rest-api-bank/dto"
	"rest-api-bank/helper"
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
	h.mux.HandleFunc(helper.NewAPIPath("POST", "/transfers"), h.Transfer())
}

func (h *TransferHandler) Transfer() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var req dto.TransferRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid body", 400)
			return
		}

		fromID, err := uuid.Parse(req.FromAccountID)
		if err != nil {
			http.Error(w, "invalid from_account_id", 400)
			return
		}

		toID, err := uuid.Parse(req.ToAccountID)
		if err != nil {
			http.Error(w, "invalid to_account_id", 400)
			return
		}

		err = h.Service.Transfer(fromID, toID, req.Amount)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		json.NewEncoder(w).Encode(dto.BaseResponse{
			Message: "Transfer success",
		})
	}
}