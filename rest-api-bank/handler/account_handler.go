package handler

import (
	"encoding/json"
	"net/http"

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
	h.mux.HandleFunc(helper.NewAPIPath("POST", "/accounts"), h.Create())
	h.mux.HandleFunc(helper.NewAPIPath("GET", "/accounts"), h.GetAll())
	h.mux.HandleFunc(helper.NewAPIPath("GET", "/accounts/"), h.GetByID())
	h.mux.HandleFunc(helper.NewAPIPath("PUT", "/accounts/"), h.Update())
	h.mux.HandleFunc(helper.NewAPIPath("DELETE", "/accounts/"), h.Delete())
}

func (h *AccountHandler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var req dto.CreateAccountRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid body", 400)
			return
		}

		if req.AccountNumber == "" {
			http.Error(w, "account_number required", 400)
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
			http.Error(w, err.Error(), 400)
			return
		}

		json.NewEncoder(w).Encode(dto.BaseResponse{
			Message: "Account created",
		})
	}
}

func (h *AccountHandler) GetAll() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		data, err := h.Service.GetAll()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		json.NewEncoder(w).Encode(dto.BaseResponse{
			Message: "Success",
			Data:    data,
		})
	}
}

func (h *AccountHandler) GetByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		id := helper.GetIDFromPath(r.URL.Path)
		data, err := h.Service.GetByID(id)
		if err != nil {
			http.Error(w, "account not found", 404)
			return
		}

		json.NewEncoder(w).Encode(dto.BaseResponse{
			Message: "Success",
			Data:    data,
		})
	}
}

func (h *AccountHandler) Update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		id := helper.UuidMustParse(helper.GetIDFromPath(r.URL.Path))

		var req dto.UpdateAccountRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid body", 400)
			return
		}

		acc := models.Account{
			ID:            id,
			AccountHolder: req.AccountHolder,
			Balance:       req.Balance,
		}

		err := h.Service.Update(acc)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		json.NewEncoder(w).Encode(dto.BaseResponse{
			Message: "Account updated",
		})
	}
}

func (h *AccountHandler) Delete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		id := helper.GetIDFromPath(r.URL.Path)

		err := h.Service.Delete(id)
		if err != nil {
			http.Error(w, "account not found", 404)
			return
		}

		json.NewEncoder(w).Encode(dto.BaseResponse{
			Message: "Account deleted successfully",
		})
	}
}

func (h *AccountHandler) GetTransaction() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		id := helper.GetIDFromTransactionPath(r.URL.Path)
		data, err := h.TransferService.GetTransaction(id)
		if err != nil {
			http.Error(w, "account not found", 404)
			return
		}

		json.NewEncoder(w).Encode(dto.BaseResponse{
			Message: "Success",
			Data:    data,
		})
	}
}