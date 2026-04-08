package service

import (
	"context"
	"encoding/json"
	"errors"
	"rest-api-bank/config"
	"rest-api-bank/dto"
	"rest-api-bank/helper"
	"rest-api-bank/middleware"
	"rest-api-bank/models"
	"rest-api-bank/pkg/logger"
	"rest-api-bank/pkg/metrics"
	"rest-api-bank/repository"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type TransferService struct {
	AccountRepo     repository.AccountRepository
	TransactionRepo repository.TransactionRepository
}

func (s *TransferService) Transfer(ctx context.Context, idemKey string, req dto.TransferRequest) (dto.TransferRequest, error) {
	ctx, span := middleware.Tracer.Start(ctx, "TransferService.Transfer")
	defer span.End()

	logger.Logger.Info("processing transfer",
		zap.String("trace_id", helper.GetTraceID(ctx)),
		zap.String("from_account_id", req.FromAccountID),
		zap.String("to_account_id", req.ToAccountID),
		zap.Int64("amount", req.Amount),
		zap.String("idempotency_key", idemKey),
	)

	key := "bank:transfer:" + idemKey

	val, err := config.RDB.Get(ctx, key).Result()
	if err == nil {
		var cached dto.TransferRequest
		err := json.Unmarshal([]byte(val), &cached)
		if err == nil {
			logger.Logger.Info("cache hit", zap.String("key", key))
			cached.Message = "Duplicate request - returning cached result"
			return cached, nil
		}
	}

	if err.Error() != "redis: nil" {
		logger.Logger.Error("redis error", zap.String("key", key), zap.Error(err))
		// log.Println("error getting from redis (fallback db):", err)
	}

	err = s.processTransfer(ctx, req)
	if err != nil {
		logger.Logger.Error("failed to process transfer", zap.Error(err))
		return dto.TransferRequest{}, err
	}

	result := dto.TransferRequest{
		Status:        "success",
		Message:       "Transfer successful",
	}
	
	data, err := json.Marshal(result)
	if err != nil {
		logger.Logger.Error("failed to marshal result", zap.Error(err))
		// log.Println("failed to marshal result:", err)
		return dto.TransferRequest{}, err
	}

	err = config.RDB.Set(ctx, key, data, 5*time.Minute).Err() // simpan hasil ke redis, set expired 5 menit
	if err != nil {
		logger.Logger.Error("failed to set result to redis", zap.Error(err))
		// log.Println("error setting to redis:", err)
	}

	return result, nil
}

func (s *TransferService) processTransfer(ctx context.Context, req dto.TransferRequest) error {
	ctx, span := middleware.Tracer.Start(ctx, "TransferService.processTransfer")
	defer span.End()

	logger.Logger.Info("processing transfer",
		zap.String("trace_id", helper.GetTraceID(ctx)),
		zap.String("from_account_id", req.FromAccountID),
		zap.String("to_account_id", req.ToAccountID),
		zap.Int64("amount", req.Amount),
	)

	from, err := s.AccountRepo.GetByID(ctx, uuid.MustParse(req.FromAccountID))
	if err != nil {
		logger.Logger.Error("failed to get from account", zap.Error(err))
		return err
	}

	to, err := s.AccountRepo.GetByID(ctx, uuid.MustParse(req.ToAccountID))
	if err != nil {
		logger.Logger.Error("failed to get to account", zap.Error(err))
		return err
	}

	if from.Balance < req.Amount {
		logger.Logger.Error("insufficient balance", zap.Int64("balance", from.Balance))
		return errors.New("insufficient balance")
	}

	from.Balance -= req.Amount
	to.Balance += req.Amount

	err = s.AccountRepo.Update(ctx, from)
	if err != nil {	
		logger.Logger.Error("failed to update from account", zap.Error(err))
		return err
	}

	err = s.AccountRepo.Update(ctx, to)
	if err != nil {
		logger.Logger.Error("failed to update to account", zap.Error(err))
		return err
	}

	tx := models.Transaction{
		ID:            uuid.New(),
		FromAccountID: from.ID,
		ToAccountID:   to.ID,
		Amount:        req.Amount,
	}

	err = s.TransactionRepo.Create(ctx, tx)
	if err != nil {
		metrics.TransferFailed.Inc()
		logger.Logger.Error("failed to create transaction", zap.Error(err))
		return err
	}

	metrics.TransferTotal.Inc()
	metrics.TransferAmount.Observe(float64(req.Amount))

	return nil
}

func (s *TransferService) GetTransaction(ctx context.Context, id string) ([]models.Transaction, error) {
	ctx, span := middleware.Tracer.Start(ctx, "TransferService.GetTransaction")
	defer span.End()

	logger.Logger.Info("getting transactions for account", zap.String("account_id", id))
	
	if id == "" {
		logger.Logger.Error("id is required")
		return nil, errors.New("id required")
	}

	return s.TransactionRepo.GetByAccountID(ctx, uuid.MustParse(id))
}