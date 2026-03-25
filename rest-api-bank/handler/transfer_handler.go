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
	h.mux.HandleFunc(helper.NewAPIPath(http.MethodPost, "/transfers"), h.Transfer())
}

func (h *TransferHandler) Transfer() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var req dto.TransferRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(dto.BaseResponse{
				ResponseCode: "400",
				ResponseDesc: "Invalid Body",
			})
			return
		}

		fromID, err := uuid.Parse(req.FromAccountID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(dto.BaseResponse{
				ResponseCode: "400",
				ResponseDesc: "Invalid from_account_id",
			})
			return
		}

		toID, err := uuid.Parse(req.ToAccountID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(dto.BaseResponse{
				ResponseCode: "400",
				ResponseDesc: "Invalid to_account_id",
			})
			return
		}

		err = h.Service.Transfer(fromID, toID, req.Amount)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(dto.BaseResponse{
				ResponseCode: "400",
				ResponseDesc: err.Error(),
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(dto.BaseResponse{
			ResponseCode: "200",
			ResponseDesc: "Transfer success",
		})
	}
}

// func (h *TransferHandler) GetTransaction() http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {

// 		id := helper.GetIDFromTransactionPath(r.URL.Path)
// 		data, err := h.Service.GetTransaction(id)
// 		if err != nil {
// 			w.WriteHeader(http.StatusNotFound)
// 			json.NewEncoder(w).Encode(dto.BaseResponse{
// 				ResponseCode: "404",
// 				ResponseDesc: "Account not found",
// 			})
// 			return
// 		}

// 		w.WriteHeader(http.StatusOK)
// 		json.NewEncoder(w).Encode(dto.BaseResponse{
// 			ResponseCode: "200",
// 			ResponseDesc: "Success",
// 			Data:         data,
// 		})
// 	}
// }