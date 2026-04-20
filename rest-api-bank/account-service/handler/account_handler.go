package handler

import (
	"encoding/json"
	"net/http"

	"microservices-bank/account-service/dto"
	"microservices-bank/account-service/helper"
	"microservices-bank/account-service/middleware"
	"microservices-bank/account-service/pkg/logger"
	"microservices-bank/account-service/service"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var(
	NormalizePath = helper.NormalizePath
)

type AccountHandler struct {
	mux *http.ServeMux
	Service *service.AccountService
}

func NewAccountHandler(mux *http.ServeMux, service *service.AccountService) *AccountHandler {
	return &AccountHandler{mux: mux, Service: service}
}

func (h *AccountHandler) MapRoutes() {
	h.mux.HandleFunc(helper.NewAPIPath(http.MethodPost, "/registration-account-creation"), 
		middleware.SnapMiddleware("06")(middleware.RateLimit(h.Create(), "auth", "create")))
	h.mux.HandleFunc(helper.NewAPIPath(http.MethodPost, "/balance/deposit"),
		middleware.SnapMiddleware("06")(middleware.RateLimit(h.Deposit(), "balance", "deposit")))
	h.mux.HandleFunc(helper.NewAPIPath(http.MethodPost, "/balance/withdraw"),
		middleware.SnapMiddleware("06")(middleware.RateLimit(h.Withdraw(), "balance", "withdraw")))
	h.mux.HandleFunc(helper.NewAPIPath(http.MethodGet, "/accounts"), middleware.RateLimit(h.GetAll(),"auth", "get_all"))
	h.mux.HandleFunc(helper.NewAPIPath(http.MethodGet, "/accounts/{id}"), middleware.RateLimit(h.GetByID(), "account", "get_by_id"))
	h.mux.HandleFunc(helper.NewAPIPath(http.MethodPut, "/accounts/{id}"), h.Update())
	h.mux.HandleFunc(helper.NewAPIPath(http.MethodDelete, "/accounts/{id}"), h.Delete())
}

func (h *AccountHandler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := middleware.Tracer.Start(r.Context(), "AccountHandler.Create")
		defer span.End()

		snap := middleware.GetSnap(ctx)
		traceID := helper.GetTraceID(ctx)

		logger.Logger.Info("handling create account request",
			zap.String("trace_id", traceID),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
		)
		
		idemKey := r.Header.Get("X-EXTERNAL-ID")
		if idemKey == "" {
			logger.Logger.Error("X-EXTERNAL-ID is empty", zap.String("trace_id", traceID))
			middleware.WriteSnapError(w, snap.ServiceCode, helper.ErrMandatoryField, "Invalid Mandatory Field X-EXTERNAL-ID")
			return
		}

		var req dto.CreateAccountRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Logger.Error("failed to decode create account request", zap.String("trace_id", traceID), zap.Error(err))
			middleware.WriteSnapError(w, snap.ServiceCode, helper.ErrInvalidField, "")
			return
		}

		if len(req.PartnerReferenceNo) > 64 {
			middleware.WriteSnapError(w, snap.ServiceCode, helper.ErrInvalidField, "partnerReferenceNo exceeds max length 64")
			return
		}
		if len(req.PhoneNo) > 16 {
			middleware.WriteSnapError(w, snap.ServiceCode, helper.ErrInvalidField, "phoneNo exceeds max length 16")
			return
		}
		if len(req.Name) > 128 {
			middleware.WriteSnapError(w, snap.ServiceCode, helper.ErrInvalidField, "name exceeds max length 128")
			return
		}
		
		result, err := h.Service.Create(ctx, idemKey, req)
		if err != nil {
			logger.Logger.Error("registration-account-creation failed",
				zap.String("trace_id", traceID), zap.Error(err))
			middleware.WriteSnapError(w, snap.ServiceCode, err, "")
			return
		}

		logger.Logger.Info("registration-account-creation success",
			zap.String("trace_id", traceID),
			zap.String("account_id", result.AccountID),
		)

		sw := w.(*middleware.SnapResponseWriter)
		sw.WriteJSON(result)
	}
}

func (h *AccountHandler) Deposit() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := middleware.Tracer.Start(r.Context(), "AccountHandler.Deposit")
		defer span.End()

		snap := middleware.GetSnap(ctx)
		traceID := helper.GetTraceID(ctx)

		logger.Logger.Info("handling deposit", zap.String("trace_id", traceID))

		idemKey := r.Header.Get("X-EXTERNAL-ID")
		if idemKey == "" {
			middleware.WriteSnapError(w, snap.ServiceCode, helper.ErrMandatoryField, "Invalid Mandatory Field X-EXTERNAL-ID")
			return
		}

		var req dto.BalanceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Logger.Error("failed to decode deposit request", zap.String("trace_id", traceID), zap.Error(err))
			middleware.WriteSnapError(w, snap.ServiceCode, helper.ErrInvalidField, "")
			return
		}

		if req.AccountNumber == "" {
			middleware.WriteSnapError(w, snap.ServiceCode, helper.ErrMandatoryField, "Invalid Mandatory Field accountNumber")
			return
		}
		if req.Amount <= 0 {
			middleware.WriteSnapError(w, snap.ServiceCode, helper.ErrInvalidField, "amount must be greater than 0")
			return
		}

		result, err := h.Service.Deposit(ctx, req)
		if err != nil {
			logger.Logger.Error("deposit failed", zap.String("trace_id", traceID), zap.Error(err))
			middleware.WriteSnapError(w, snap.ServiceCode, err, "")
			return
		}

		sw := w.(*middleware.SnapResponseWriter)
		sw.WriteJSON(result)
	}
}

func (h *AccountHandler) Withdraw() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := middleware.Tracer.Start(r.Context(), "AccountHandler.Withdraw")
		defer span.End()

		snap := middleware.GetSnap(ctx)
		traceID := helper.GetTraceID(ctx)

		logger.Logger.Info("handling withdrawal", zap.String("trace_id", traceID))

		idemKey := r.Header.Get("X-EXTERNAL-ID")
		if idemKey == "" {
			middleware.WriteSnapError(w, snap.ServiceCode, helper.ErrMandatoryField, "Invalid Mandatory Field X-EXTERNAL-ID")
			return
		}

		var req dto.BalanceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Logger.Error("failed to decode withdrawal request", zap.String("trace_id", traceID), zap.Error(err))
			middleware.WriteSnapError(w, snap.ServiceCode, helper.ErrInvalidField, "")
			return
		}

		if req.AccountNumber == "" {
			middleware.WriteSnapError(w, snap.ServiceCode, helper.ErrMandatoryField, "Invalid Mandatory Field accountNumber")
			return
		}
		if req.Amount <= 0 {
			middleware.WriteSnapError(w, snap.ServiceCode, helper.ErrInvalidField, "amount must be greater than 0")
			return
		}

		result, err := h.Service.Withdraw(ctx, req)
		if err != nil {
			logger.Logger.Error("withdrawal failed", zap.String("trace_id", traceID), zap.Error(err))
			middleware.WriteSnapError(w, snap.ServiceCode, err, "")
			return
		}

		sw := w.(*middleware.SnapResponseWriter)
		sw.WriteJSON(result)
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

		id, err := uuid.Parse(helper.GetIDFromPath(r.URL.Path))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(dto.BaseResponse{ResponseCode: "400", ResponseDesc: "Invalid ID format"})
			return
		}

		var req dto.UpdateAccountRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(dto.BaseResponse{ResponseCode: "400", ResponseDesc: "Invalid Body"})
			return
		}

		if err := h.Service.Update(ctx, id, req); err != nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(dto.BaseResponse{ResponseCode: "404", ResponseDesc: "Account not found"})
			return
		}

		updated, _ := h.Service.GetByID(ctx, id.String())
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(dto.BaseResponse{ResponseCode: "200", ResponseDesc: "Account updated", Data: updated})
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
