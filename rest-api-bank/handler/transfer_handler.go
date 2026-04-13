package handler

import (
	"encoding/json"
	"net/http"

	"rest-api-bank/dto"
	"rest-api-bank/helper"
	"rest-api-bank/middleware"
	"rest-api-bank/pkg/logger"
	"rest-api-bank/service"

	"go.uber.org/zap"
)

type TransferHandler struct {
	mux *http.ServeMux
	Service *service.TransferService
}

func NewTransferHandler(mux *http.ServeMux, service *service.TransferService) *TransferHandler {
	return &TransferHandler{mux: mux, Service: service}
}

func (h *TransferHandler) MapRoutes() {
	h.mux.HandleFunc(helper.NewAPIPath(http.MethodPost, "/transfers-intrabank"), 
					middleware.SnapMiddleware("17")(middleware.RateLimit(h.Transfer(), "account", "transfer")))
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
			zap.String("path", NormalizePath(r.URL.Path)),
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
			logger.Logger.Error("partnerReferenceNo is required")
			middleware.WriteSnapError(w, snap.ServiceCode, helper.ErrMandatoryField, "Invalid Mandatory Field partnerReferenceNo")
			return
		}

		if req.SourceAccountNo == "" {
			logger.Logger.Error("sourceAccountNo is required")
			middleware.WriteSnapError(w, snap.ServiceCode, helper.ErrMandatoryField, "Invalid Mandatory Field sourceAccountNo")
			return
		}

		if req.BeneficiaryAccountNo == "" {
			logger.Logger.Error("beneficiaryAccountNo is required")
			middleware.WriteSnapError(w, snap.ServiceCode, helper.ErrMandatoryField, "Invalid Mandatory Field beneficiaryAccountNo")
			return
		}

		if req.Amount.Value == "" || req.Amount.Currency == "" {
			logger.Logger.Error("amount value and currency are required")
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