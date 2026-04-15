package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"rest-api-bank/config"
	"rest-api-bank/dto"
	"rest-api-bank/helper"
	"rest-api-bank/middleware"
	"rest-api-bank/models"
	kafkapkg "rest-api-bank/pkg/kafka"
	"rest-api-bank/pkg/logger"
	"rest-api-bank/pkg/metrics"
	"rest-api-bank/repository"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type TransferService struct {
	AccountRepo     repository.AccountRepository
	TransactionRepo repository.TransactionRepository
	Publisher 		kafkapkg.Publisher
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
		// Cache hit — return idempotent response
		var cached dto.SnapTransferResponse
		if jsonErr := json.Unmarshal([]byte(val), &cached); jsonErr == nil {
			logger.Logger.Info("idempotency cache hit", zap.String("key", key))
			// FIX: per SNAP BI, duplicate request still returns the original response unchanged
			return cached, nil
		}
	} else if !errors.Is(err, redis.Nil) {
		// Real Redis error — log and fallthrough to process normally
		logger.Logger.Error("redis get error (fallback to db)", zap.String("key", key), zap.Error(err))
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
		ResponseCode:    helper.SnapResponseCode(200, snap.ServiceCode, "00"),
		ResponseMessage: "Request has been processed successfully",
		ReferenceNo:     referenceNo,

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
		// log.Println("failed to marshal result:", err)
		return dto.SnapTransferResponse{}, err
	}

	if err := config.RDB.Set(ctx, key, data, 5*time.Minute).Err(); err != nil {
		// Non-fatal: log but don't fail the request
		logger.Logger.Warn("failed to cache transfer result in redis", zap.String("key", key), zap.Error(err))
	}

	s.Publisher.TransactionCreated(ctx, kafkapkg.TransactionCreatedEvent{
		TransactionID:        txID,
		SourceAccountNo:      req.SourceAccountNo,
		BeneficiaryAccountNo: req.BeneficiaryAccountNo,
		AmountValue:          req.Amount.Value,
		Currency:             req.Currency,
		Remark:               req.Remark,
		PartnerRefNo:         req.PartnerReferenceNo,
		ReferenceNo:          referenceNo,
	})

	return result, nil
}

func (s *TransferService) processTransfer(ctx context.Context, req dto.SnapTransferRequest) (txID string, err	error) {
	ctx, span := middleware.Tracer.Start(ctx, "TransferService.processTransfer")
	defer span.End()

	logger.Logger.Info("processing transfer internal",
		zap.String("trace_id", helper.GetTraceID(ctx)),
		zap.String("from_account_id", req.SourceAccountNo),
		zap.String("to_account_id", req.BeneficiaryAccountNo),
		zap.String("amount", req.Amount.Value),
	)

	from, err := s.AccountRepo.GetByAccountNumber(ctx, req.SourceAccountNo)
	if err != nil {
		logger.Logger.Error("source account not found", zap.String("source_account_no", req.SourceAccountNo))
		return "",  helper.ErrAccountNotFound
	}

	to, err := s.AccountRepo.GetByAccountNumber(ctx, req.BeneficiaryAccountNo)
	if err != nil {
		logger.Logger.Error("beneficiary account not found", zap.String("beneficiary_account_no", req.BeneficiaryAccountNo))
		return "",helper.ErrAccountNotFound
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

	if from.Balance < amountInt {
		logger.Logger.Error("insufficient balance", 
			zap.Int64("balance", from.Balance),
			zap.Int64("required", amountInt),
		)
		return "",helper.ErrInsufficientFunds
	}

	from.Balance -= amountInt
	to.Balance += amountInt

	err = s.AccountRepo.Update(ctx, from)
	if err != nil {	
		logger.Logger.Error("failed to update source account balance", zap.Error(err))
		return "", err
	}

	err = s.AccountRepo.Update(ctx, to)
	if err != nil {
		logger.Logger.Error("failed to update beneficiary account balance", zap.Error(err))
		return "", err
	}

	newTxID := uuid.New()
	tx := models.Transaction{
		ID:            newTxID,
		FromAccountID: from.ID,
		ToAccountID:   to.ID,
		Amount:        amountInt,
	}

	err = s.TransactionRepo.Create(ctx, tx)
	if err != nil {
		metrics.TransferFailed.Inc()
		logger.Logger.Error("failed to create transaction", zap.Error(err))
		return "", err
	}

	metrics.TransferTotal.Inc()
	metrics.TransferAmount.Observe(float64(amountInt))

	return newTxID.String(),nil
}

func (s *TransferService) GetTransaction(ctx context.Context, id string) ([]dto.TransactionResponse, error) {
	ctx, span := middleware.Tracer.Start(ctx, "TransferService.GetTransaction")
	defer span.End()

	logger.Logger.Info("getting transactions for account", zap.String("account_id", id))
	
	if id == "" {
		logger.Logger.Error("id is required")
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
			CreatedAt:     tx.CreatedAt,
		})
	}

	return result,nil
}