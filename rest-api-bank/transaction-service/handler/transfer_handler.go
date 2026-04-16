package handler

import (
	"encoding/json"
	"net/http"

	"transaction-service/dto"
	"transaction-service/helper"
	"transaction-service/middleware"
	"transaction-service/pkg/logger"
	"transaction-service/service"

	"go.uber.org/zap"
)

type TransferHandler struct {
	mux     *http.ServeMux
	Service *service.TransferService
}

func NewTransferHandler(mux *http.ServeMux, service *service.TransferService) *TransferHandler {
	return &TransferHandler{mux: mux, Service: service}
}

func (h *TransferHandler) MapRoutes() {
	h.mux.HandleFunc(helper.NewAPIPath(http.MethodPost, "/transfers-intrabank"),
		middleware.SnapMiddleware("17")(middleware.RateLimit(h.Transfer(), "account", "transfer")))
	h.mux.HandleFunc(helper.NewAPIPath(http.MethodGet, "/accounts/{id}/transactions"),
		middleware.RateLimit(h.GetTransaction(), "account", "get_transaction"))
}

func (h *TransferHandler) Transfer() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := middleware.Tracer.Start(r.Context(), "TransferHandler.Transfer")
		defer span.End()

		snap := middleware.GetSnap(ctx)
		traceID := helper.GetTraceID(ctx)

		logger.Logger.Info("handling transfer request",
			zap.String("trace_id", traceID),
			zap.String("method", r.Method),
			zap.String("path", helper.NormalizePath(r.URL.Path)),
		)

		idemKey := r.Header.Get("X-EXTERNAL-ID")
		if idemKey == "" {
			logger.Logger.Error("X-EXTERNAL-ID header is empty")
			middleware.WriteSnapError(w, snap.ServiceCode, helper.ErrMandatoryField, "Invalid Mandatory Field X-EXTERNAL-ID")
			return
		}

		var req dto.SnapTransferRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Logger.Error("failed to decode transfer request", zap.Error(err))
			middleware.WriteSnapError(w, snap.ServiceCode, helper.ErrInvalidField, "")
			return
		}

		if req.PartnerReferenceNo == "" {
			middleware.WriteSnapError(w, snap.ServiceCode, helper.ErrMandatoryField, "Invalid Mandatory Field partnerReferenceNo")
			return
		}
		if req.SourceAccountNo == "" {
			middleware.WriteSnapError(w, snap.ServiceCode, helper.ErrMandatoryField, "Invalid Mandatory Field sourceAccountNo")
			return
		}
		if req.BeneficiaryAccountNo == "" {
			middleware.WriteSnapError(w, snap.ServiceCode, helper.ErrMandatoryField, "Invalid Mandatory Field beneficiaryAccountNo")
			return
		}
		if req.Amount.Value == "" || req.Amount.Currency == "" {
			middleware.WriteSnapError(w, snap.ServiceCode, helper.ErrMandatoryField, "Invalid Mandatory Field amount")
			return
		}

		result, err := h.Service.Transfer(ctx, idemKey, req)
		if err != nil {
			logger.Logger.Error("failed to process transfer", zap.Error(err), zap.String("trace_id", traceID))
			middleware.WriteSnapError(w, snap.ServiceCode, err, "")
			return
		}

		sw := w.(*middleware.SnapResponseWriter)
		sw.WriteJSON(result)
	}
}

func (h *TransferHandler) GetTransaction() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := middleware.Tracer.Start(r.Context(), "TransferHandler.GetTransaction")
		defer span.End()

		logger.Logger.Info("handling get transaction request",
			zap.String("trace_id", helper.GetTraceID(ctx)),
			zap.String("method", r.Method),
			zap.String("path", helper.NormalizePath(r.URL.Path)),
		)

		id := helper.GetIDFromTransactionPath(r.URL.Path)
		data, err := h.Service.GetTransaction(ctx, id)
		if err != nil {
			logger.Logger.Error("failed to get transaction", zap.Error(err))
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(dto.BaseResponse{ResponseCode: "404", ResponseDesc: "Account not found"})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(dto.BaseResponse{ResponseCode: "200", ResponseDesc: "Success", Data: data})
	}
}
