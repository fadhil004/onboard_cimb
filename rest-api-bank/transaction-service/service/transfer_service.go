package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	pb "microservices-bank/proto/accountpb"
	fraudpb "microservices-bank/proto/fraudpb"
	"microservices-bank/transaction-service/config"
	"microservices-bank/transaction-service/dto"
	"microservices-bank/transaction-service/helper"
	"microservices-bank/transaction-service/middleware"
	"microservices-bank/transaction-service/models"
	kafkapkg "microservices-bank/transaction-service/pkg/kafka"
	"microservices-bank/transaction-service/pkg/logger"
	"microservices-bank/transaction-service/pkg/metrics"
	"microservices-bank/transaction-service/repository"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// ErrFraudDetected is returned when FDS rejects a transaction.
var ErrFraudDetected = errors.New("transaction rejected by fraud detection")

type TransferService struct {
	TransactionRepo repository.TransactionRepository
	Publisher       kafkapkg.Publisher
	AccountClient   pb.AccountServiceClient
	FraudClient     fraudpb.FraudDetectionServiceClient
}

func (s *TransferService) Transfer(ctx context.Context, idemKey string, req dto.SnapTransferRequest) (dto.SnapTransferResponse, error) {
	ctx, span := middleware.Tracer.Start(ctx, "TransferService.Transfer")
	defer span.End()

	snap := middleware.GetSnap(ctx)

	logger.Logger.Info("processing transfer",
		zap.String("trace_id", helper.GetTraceID(ctx)),
		zap.String("from_account_id", req.SourceAccountNo),
		zap.String("to_account_id", req.BeneficiaryAccountNo),
		zap.String("amount", req.Amount.Value),
		zap.String("idempotency_key", idemKey),
	)

	key := "bank:transfer:" + idemKey

	val, err := config.RDB.Get(ctx, key).Result()
	if err == nil {
		var cached dto.SnapTransferResponse
		if jsonErr := json.Unmarshal([]byte(val), &cached); jsonErr == nil {
			logger.Logger.Info("idempotency cache hit", zap.String("key", key))
			return cached, nil
		}
	} else if !errors.Is(err, redis.Nil) {
		logger.Logger.Error("redis get error (fallback to db)", zap.String("key", key), zap.Error(err))
	}

	// Fraud Detection — called before any balance operation
	if err := s.runFraudCheck(ctx, req); err != nil {
		return dto.SnapTransferResponse{}, err
	}

	txID, err := s.processTransfer(ctx, req)
	if err != nil {
		logger.Logger.Error("transfer failed",
			zap.String("trace_id", helper.GetTraceID(ctx)),
			zap.Error(err),
		)
		return dto.SnapTransferResponse{}, err
	}

	referenceNo := uuid.New().String()

	result := dto.SnapTransferResponse{
		ResponseCode:         helper.SnapResponseCode(200, snap.ServiceCode, "00"),
		ResponseMessage:      "Request has been processed successfully",
		ReferenceNo:          referenceNo,
		PartnerReferenceNo:   req.PartnerReferenceNo,
		Amount:               req.Amount,
		BeneficiaryAccountNo: req.BeneficiaryAccountNo,
		Currency:             req.Currency,
		CustomerReference:    req.CustomerReference,
		SourceAccount:        req.SourceAccountNo,
		TransactionDate:      req.TransactionDate,
		OriginatorInfos:      req.OriginatorInfos,
		AdditionalInfo:       req.AdditionalInfo,
	}

	data, err := json.Marshal(result)
	if err != nil {
		logger.Logger.Error("failed to marshal result", zap.Error(err))
		return dto.SnapTransferResponse{}, err
	}

	if err := config.RDB.Set(ctx, key, data, 5*time.Minute).Err(); err != nil {
		logger.Logger.Warn("failed to cache transfer result in redis", zap.String("key", key), zap.Error(err))
	}

	s.Publisher.TransactionCreated(ctx, kafkapkg.TransactionCreatedEvent{
		TransactionID:        txID,
		SourceAccountNo:      req.SourceAccountNo,
		BeneficiaryAccountNo: req.BeneficiaryAccountNo,
		AmountValue:          req.Amount.Value,
		Currency:             req.Currency,
		Remark:               req.Remark,
		Status:               "SUCCESS",
		PartnerRefNo:         req.PartnerReferenceNo,
		ReferenceNo:          referenceNo,
	})

	return result, nil
}

// runFraudCheck calls fraud-detection-service (Node.js) via gRPC.
// Fail-open policy: if FDS unreachable, log error and allow the transaction.
func (s *TransferService) runFraudCheck(ctx context.Context, req dto.SnapTransferRequest) error {
	ctx, span := middleware.Tracer.Start(ctx, "TransferService.runFraudCheck")
	defer span.End()

	if s.FraudClient == nil {
		logger.Logger.Warn("FraudClient not configured — skipping fraud check")
		return nil
	}

	amountFloat, _ := strconv.ParseFloat(req.Amount.Value, 64)

	fraudResp, err := s.FraudClient.CheckTransaction(ctx, &fraudpb.FraudCheckRequest{
		SourceAccountNo:      req.SourceAccountNo,
		BeneficiaryAccountNo: req.BeneficiaryAccountNo,
		Amount:               amountFloat,
		Currency:             req.Currency,
		PartnerReferenceNo:   req.PartnerReferenceNo,
		TransactionDate:      req.TransactionDate,
	})
	if err != nil {
		logger.Logger.Error("fraud-detection-service unreachable — failing open",
			zap.String("trace_id", helper.GetTraceID(ctx)),
			zap.Error(err),
		)
		return nil
	}

	logger.Logger.Info("fraud check result",
		zap.String("source", req.SourceAccountNo),
		zap.String("beneficiary", req.BeneficiaryAccountNo),
		zap.Bool("allowed", fraudResp.Allowed),
		zap.String("fraud_code", fraudResp.FraudCode),
		zap.String("risk_level", fraudResp.RiskLevel),
		zap.Int32("score", fraudResp.Score),
		zap.String("decision", fraudResp.Decision),
	)

	if fraudResp.Decision == "REVIEW" {
		metrics.TransferFailed.Inc()
		logger.Logger.Warn("transaction needs review",
			zap.String("source", req.SourceAccountNo),
			zap.Int32("score", fraudResp.Score),
		)
		return helper.ErrNeedReview
	}
	
	if fraudResp.FraudCode == "ACCOUNT_RESTRICTED" {
		metrics.TransferFailed.Inc()
		logger.Logger.Warn("account restricted",
			zap.String("source", req.SourceAccountNo),
			zap.Int32("score", fraudResp.Score),
		)
		return helper.ErrAccountRestricted
	}

	if !fraudResp.Allowed{
		metrics.TransferFailed.Inc()
		return helper.ErrSupectedFraud
	}

	return nil
}

func (s *TransferService) processTransfer(ctx context.Context, req dto.SnapTransferRequest) (txID string, err error) {
	ctx, span := middleware.Tracer.Start(ctx, "TransferService.processTransfer")
	defer span.End()

	logger.Logger.Info("processing transfer via gRPC",
		zap.String("trace_id", helper.GetTraceID(ctx)),
		zap.String("from", req.SourceAccountNo),
		zap.String("to", req.BeneficiaryAccountNo),
		zap.String("amount", req.Amount.Value),
	)

	fromAcc, err := s.AccountClient.GetByAccountNumber(ctx, &pb.GetByAccountNumberRequest{
		AccountNumber: req.SourceAccountNo,
	})
	if err != nil {
		logger.Logger.Error("source account not found via gRPC", zap.String("source_account_no", req.SourceAccountNo), zap.Error(err))
		return "", helper.ErrAccountNotFound
	}

	toAcc, err := s.AccountClient.GetByAccountNumber(ctx, &pb.GetByAccountNumberRequest{
		AccountNumber: req.BeneficiaryAccountNo,
	})
	if err != nil {
		logger.Logger.Error("beneficiary account not found via gRPC", zap.String("beneficiary_account_no", req.BeneficiaryAccountNo), zap.Error(err))
		return "", helper.ErrAccountNotFound
	}

	amountFloat, err := strconv.ParseFloat(req.Amount.Value, 64)
	if err != nil {
		logger.Logger.Error("invalid amount value", zap.String("amount", req.Amount.Value))
		return "", helper.ErrInvalidField
	}
	amountInt := int64(amountFloat)
	if amountInt <= 0 {
		logger.Logger.Error("amount must be positive", zap.String("amount", req.Amount.Value))
		return "", fmt.Errorf("%w: amount must be greater than 0", helper.ErrInvalidField)
	}

	if fromAcc.Balance < amountInt {
		logger.Logger.Error("insufficient balance",
			zap.Int64("balance", fromAcc.Balance),
			zap.Int64("required", amountInt),
		)
		return "", helper.ErrInsufficientFunds
	}

	debitResp, err := s.AccountClient.UpdateBalance(ctx, &pb.UpdateBalanceRequest{
		AccountNumber: req.SourceAccountNo,
		Amount:        -amountInt,
	})
	if err != nil || !debitResp.Success {
		logger.Logger.Error("failed to debit source account", zap.Error(err))
		metrics.TransferFailed.Inc()
		return "", helper.ErrInsufficientFunds
	}

	creditResp, err := s.AccountClient.UpdateBalance(ctx, &pb.UpdateBalanceRequest{
		AccountNumber: req.BeneficiaryAccountNo,
		Amount:        amountInt,
	})
	if err != nil || !creditResp.Success {
		logger.Logger.Error("failed to credit beneficiary account, rolling back debit", zap.Error(err))
		s.AccountClient.UpdateBalance(ctx, &pb.UpdateBalanceRequest{
			AccountNumber: req.SourceAccountNo,
			Amount:        amountInt,
		})
		metrics.TransferFailed.Inc()
		return "", fmt.Errorf("failed to credit beneficiary account")
	}

	fromID, _ := uuid.Parse(fromAcc.Id)
	toID, _ := uuid.Parse(toAcc.Id)
	newTxID := uuid.New()

	tx := models.Transaction{
		ID:            newTxID,
		FromAccountID: fromID,
		ToAccountID:   toID,
		Amount:        amountInt,
		Remark:        req.Remark,
		Status:        "SUCCESS",
	}

	err = s.TransactionRepo.Create(ctx, tx)
	if err != nil {
		metrics.TransferFailed.Inc()
		logger.Logger.Error("failed to create transaction record", zap.Error(err))
		return "", err
	}

	metrics.TransferTotal.Inc()
	metrics.TransferAmount.Observe(float64(amountInt))

	return newTxID.String(), nil
}

func (s *TransferService) GetTransaction(ctx context.Context, id string) ([]dto.TransactionResponse, error) {
	ctx, span := middleware.Tracer.Start(ctx, "TransferService.GetTransaction")
	defer span.End()

	logger.Logger.Info("getting transactions for account", zap.String("account_id", id))

	if id == "" {
		return nil, errors.New("id required")
	}

	transactions, err := s.TransactionRepo.GetByAccountID(ctx, uuid.MustParse(id))
	if err != nil {
		return nil, err
	}

	result := make([]dto.TransactionResponse, 0, len(transactions))
	for _, tx := range transactions {
		result = append(result, dto.TransactionResponse{
			ID:            tx.ID.String(),
			FromAccountID: tx.FromAccountID.String(),
			ToAccountID:   tx.ToAccountID.String(),
			Amount:        tx.Amount,
			Remark:        tx.Remark,
			Status:        tx.Status,
			CreatedAt:     tx.CreatedAt,
		})
	}

	return result, nil
}
