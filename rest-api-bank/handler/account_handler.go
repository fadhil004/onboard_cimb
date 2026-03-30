package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"rest-api-bank/dto"
	"rest-api-bank/helper"
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
	h.mux.HandleFunc(helper.NewAPIPath(http.MethodPost, "/accounts"), h.Create())
	h.mux.HandleFunc(helper.NewAPIPath(http.MethodGet, "/accounts"), h.GetAll())
	// h.mux.HandleFunc(helper.NewAPIPath(http.MethodGet, "/accounts/"), h.GetByID())
	h.mux.HandleFunc(helper.NewAPIPath(http.MethodPut, "/accounts/"), h.Update())
	h.mux.HandleFunc(helper.NewAPIPath(http.MethodDelete, "/accounts/"), h.Delete())
	h.mux.HandleFunc(helper.NewAPIPath(http.MethodGet, "/accounts/"), h.HandleAccountsAdvanced())
}

func (h *AccountHandler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

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

		err := h.Service.Create(acc)
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

		data, err := h.Service.GetAll()
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

		id := helper.GetIDFromPath(r.URL.Path)
		data, err := h.Service.GetByID(id)
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

		err := h.Service.Update(acc)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(dto.BaseResponse{
				ResponseCode: "404",
				ResponseDesc: "Account not found",
			})
			return
		}

		updatedData, _ := h.Service.GetByID(id.String())
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

		id := helper.GetIDFromTransactionPath(r.URL.Path)
		data, err := h.TransferService.GetTransaction(id)
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

		id := helper.GetIDFromPath(r.URL.Path)

		err := h.Service.Delete(id)
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

func (h *AccountHandler) HandleAccountsAdvanced() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		path := r.URL.Path

		//accounts/{id}/transactions
		if strings.Contains(path, "/transactions") {
			h.GetTransaction()(w, r)
			return
		}

		//default /accounts/{id}
		h.GetByID()(w, r)
	}
}