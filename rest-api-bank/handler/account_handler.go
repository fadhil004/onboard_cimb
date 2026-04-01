package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"rest-api-bank/dto"
	"rest-api-bank/helper"
	"rest-api-bank/middleware"
	"rest-api-bank/models"
	"rest-api-bank/service"

	"github.com/google/uuid"
)

type AccountHandler struct {
	mux *http.ServeMux
	Service *service.AccountService
	TransferService *service.TransferService
}

func NewAccountHandler(mux *http.ServeMux, service *service.AccountService, transferService *service.TransferService) *AccountHandler {
	return &AccountHandler{mux: mux, Service: service, TransferService: transferService}
}

func (h *AccountHandler) MapRoutes() {
	h.mux.HandleFunc(helper.NewAPIPath(http.MethodPost, "/accounts"), middleware.RateLimit(h.Create(),"auth", "create"))
	h.mux.HandleFunc(helper.NewAPIPath(http.MethodGet, "/accounts"), middleware.RateLimit(h.GetAll(),"auth", "get_all"))
	h.mux.HandleFunc(helper.NewAPIPath(http.MethodGet, "/accounts/{id}"), middleware.RateLimit(h.GetByID(), "account", "get_by_id"))
	h.mux.HandleFunc(helper.NewAPIPath(http.MethodPut, "/accounts/{id}"), h.Update())
	h.mux.HandleFunc(helper.NewAPIPath(http.MethodDelete, "/accounts/{id}"), h.Delete())
	h.mux.HandleFunc(helper.NewAPIPath(http.MethodGet, "/accounts/{id}/transactions"), middleware.RateLimit(h.GetTransaction(), "account", "get_transaction"))
}

func (h *AccountHandler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		
		var req dto.CreateAccountRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(dto.BaseResponse{
				ResponseCode: "400",
				ResponseDesc: "Invalid Body",
			})
			return
		}

		if req.AccountNumber == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(dto.BaseResponse{
				ResponseCode: "400",
				ResponseDesc: "Account Number Required",
			})
			return
		}

		acc := models.Account{
			ID:            uuid.New(),
			AccountNumber: req.AccountNumber,
			AccountHolder: req.AccountHolder,
			Balance:       req.Balance,
		}

		err := h.Service.Create(ctx,acc)
		if err != nil {
			if strings.Contains(err.Error(), "duplicate key") {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(dto.BaseResponse{
					ResponseCode: "400",
					ResponseDesc: "Account number already exists",
				})
				return
			}
			
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(dto.BaseResponse{
				ResponseCode: "500",
				ResponseDesc: "Internal Server Error",
			})
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(dto.BaseResponse{
			ResponseCode: "201",
			ResponseDesc: "Account Created",
			Data:         acc,
		})
	}
}

func (h *AccountHandler) GetAll() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		data, err := h.Service.GetAll(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(dto.BaseResponse{
				ResponseCode: "500",
				ResponseDesc: "Internal Server Error",
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(dto.BaseResponse{
			ResponseCode: "200",
			ResponseDesc: "Success",
			Data:         data,
		})
	}
}

func (h *AccountHandler) GetByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id := helper.GetIDFromPath(r.URL.Path)
		data, err := h.Service.GetByID(ctx, id)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(dto.BaseResponse{
				ResponseCode: "404",
				ResponseDesc: "Account not found",
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(dto.BaseResponse{
			ResponseCode: "200",
			ResponseDesc: "Success",
			Data:         data,
		})
	}
}

func (h *AccountHandler) Update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id := helper.UuidMustParse(helper.GetIDFromPath(r.URL.Path))

		var req dto.UpdateAccountRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(dto.BaseResponse{
				ResponseCode: "400",
				ResponseDesc: "Invalid Body",
			})
			return
		}

		acc := models.Account{
			ID:            id,
			AccountHolder: req.AccountHolder,
			Balance:       req.Balance,
		}

		err := h.Service.Update(ctx, acc)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(dto.BaseResponse{
				ResponseCode: "404",
				ResponseDesc: "Account not found",
			})
			return
		}

		updatedData, _ := h.Service.GetByID(ctx, id.String())
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(dto.BaseResponse{
			ResponseCode: "200",
			ResponseDesc: "Account updated",
			Data:         updatedData,
		})
	}
}

func (h *AccountHandler) GetTransaction() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id := helper.GetIDFromTransactionPath(r.URL.Path)
		data, err := h.TransferService.GetTransaction(ctx, id)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(dto.BaseResponse{
				ResponseCode: "404",
				ResponseDesc: "Account not found",
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(dto.BaseResponse{
			ResponseCode: "200",
			ResponseDesc: "Success",
			Data:         data,
		})
	}
}

func (h *AccountHandler) Delete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id := helper.GetIDFromPath(r.URL.Path)

		err := h.Service.Delete(ctx, id)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(dto.BaseResponse{
				ResponseCode: "404",
				ResponseDesc: "Account not found",
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(dto.BaseResponse{
			ResponseCode: "200",
			ResponseDesc: "Account deleted successfully",
		})
	}
}
