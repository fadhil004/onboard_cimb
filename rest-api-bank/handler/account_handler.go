package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"rest-api-bank/dto"
	"rest-api-bank/helper"
	"rest-api-bank/middleware"
	"rest-api-bank/models"
	"rest-api-bank/pkg/logger"
	"rest-api-bank/service"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var(
	NormalizePath = helper.NormalizePath
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
		ctx, span := middleware.Tracer.Start(r.Context(), "AccountHandler.Create")
		defer span.End()

		logger.Logger.Info("handling create account request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
		)
		
		var req dto.CreateAccountRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Logger.Error("failed to decode create account request", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(dto.BaseResponse{
				ResponseCode: "400",
				ResponseDesc: "Invalid Body",
			})
			return
		}

		if req.AccountNumber == "" {
			logger.Logger.Error("account number is required")
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
			logger.Logger.Error("failed to create account", zap.Error(err))
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
		ctx, span := middleware.Tracer.Start(r.Context(), "AccountHandler.GetAll")
		defer span.End()

		logger.Logger.Info("handling get all account request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
		)

		data, err := h.Service.GetAll(ctx)
		if err != nil {
			logger.Logger.Error("failed to get all accounts", zap.Error(err))
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
		ctx,span := middleware.Tracer.Start(r.Context(), "AccountHandler.GetByID")
		defer span.End()

		logger.Logger.Info("handling get account by id request",
			zap.String("trace_id", helper.GetTraceID(ctx)),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
		)

		id := helper.GetIDFromPath(r.URL.Path)
		data, err := h.Service.GetByID(ctx, id)
		if err != nil {
			logger.Logger.Error("failed to get account by id", zap.Error(err))
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
		ctx, span := middleware.Tracer.Start(r.Context(), "AccountHandler.Update")
		defer span.End()

		logger.Logger.Info("handling update account request",
			zap.String("trace_id", helper.GetTraceID(ctx)),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
		)

		id, err := uuid.Parse(helper.GetIDFromPath(r.URL.Path))
		if err != nil {
			logger.Logger.Error("invalid id format", zap.String("id", id.String()))
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(dto.BaseResponse{
				ResponseCode: "400",
				ResponseDesc: "Invalid ID format",
			})
			return
		}

		var req dto.UpdateAccountRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Logger.Error("failed to decode update account request", zap.Error(err))
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

		err = h.Service.Update(ctx, acc)
		if err != nil {
			logger.Logger.Error("failed to update account", zap.Error(err))
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
		ctx, span := middleware.Tracer.Start(r.Context(), "AccountHandler.GetTransaction")
		defer span.End()

		logger.Logger.Info("handling get transaction request",
			zap.String("trace_id", helper.GetTraceID(ctx)),
			zap.String("method", r.Method),
			zap.String("path", NormalizePath(r.URL.Path)),
		)

		id := helper.GetIDFromTransactionPath(r.URL.Path)
		data, err := h.TransferService.GetTransaction(ctx, id)
		if err != nil {
			logger.Logger.Error("failed to get transaction", zap.Error(err))
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
		ctx,span := middleware.Tracer.Start(r.Context(), "AccountHandler.Delete")
		defer span.End()

		logger.Logger.Info("handling delete account request",
			zap.String("trace_id", helper.GetTraceID(ctx)),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
		)

		id := helper.GetIDFromPath(r.URL.Path)

		err := h.Service.Delete(ctx, id)
		if err != nil {
			logger.Logger.Error("failed to delete account", zap.Error(err))
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
