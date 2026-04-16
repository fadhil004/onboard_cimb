package service

import (
	"context"
	"encoding/json"
	"errors"
	"account-service/config"
	"account-service/dto"
	"account-service/helper"
	"account-service/middleware"
	"account-service/models"
	kafkapkg "account-service/pkg/kafka"
	"account-service/pkg/logger"
	"account-service/repository"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const serviceCodeAccountCreation = "06"

type AccountService struct {
	Repo repository.AccountRepository
	Publisher kafkapkg.Publisher
}

func (s *AccountService) Create(ctx context.Context, idemKey string, req dto.CreateAccountRequest) (dto.CreateAccountResponse, error) {
	ctx, span := middleware.Tracer.Start(ctx, "AccountService.Create")
	defer span.End()

	traceID := helper.GetTraceID(ctx)

	logger.Logger.Info("processing request for creating account",
		zap.String("trace_id", traceID),
		zap.String("idem_key", idemKey),
		zap.String("name", req.Name),
		zap.String("partner_reference_no", req.PartnerReferenceNo),
	)

	cacheKey := "bank:account-creation:" + idemKey
	val, err := config.RDB.Get(ctx, cacheKey).Result()
	if err == nil {
		var cached dto.CreateAccountResponse
		if jsonErr := json.Unmarshal([]byte(val), &cached); jsonErr == nil {
			logger.Logger.Info("idempotency cache hit",
				zap.String("trace_id", traceID),
				zap.String("idem_key", idemKey),
			)
			return cached, nil
		}
	} else if !errors.Is(err, redis.Nil) {
		logger.Logger.Warn("redis get error, proceeding",
			zap.String("trace_id", traceID),
			zap.Error(err),
		)
	}

	accountNumber := helper.GenerateAccountNumber()   
	referenceNo   := uuid.New().String()
	authCode      := helper.GenerateAuthCode(req.PartnerReferenceNo, accountNumber)
	apiKey        := helper.GenerateAPIKey()

	accountHolder := req.Name
	if accountHolder == "" {
		accountHolder = "Anonymous"
	}

	acc := models.Account{
		ID:            uuid.New(),
		AccountNumber: accountNumber,
		AccountHolder: accountHolder,
		Balance:       0, 
	}

	if err := s.Repo.Create(ctx, acc); err != nil {
		logger.Logger.Error("failed to save account",
			zap.String("trace_id", traceID),
			zap.String("account_number", accountNumber),
			zap.Error(err),
		)
		return dto.CreateAccountResponse{}, err
	}

	result := dto.CreateAccountResponse{
		ResponseCode:       helper.SnapResponseCode(200, serviceCodeAccountCreation, "00"),
		ResponseMessage:    "Request has been processed successfully",
		ReferenceNo:        referenceNo,
		PartnerReferenceNo: req.PartnerReferenceNo,
		AuthCode:           authCode,
		APIKey:             apiKey,
		AccountID:          acc.ID.String(),
		AccountNumber: 		accountNumber,
		State:              req.State,
		AdditionalInfo:     req.AdditionalInfo,
	}

	// Cache ke Redis untuk idempotency
	data, err := json.Marshal(result)
	if err == nil {
		if cacheErr := config.RDB.Set(ctx, cacheKey, data, 5*time.Minute).Err(); cacheErr != nil {
			logger.Logger.Warn("failed to cache account creation result (non-fatal)",
				zap.String("trace_id", traceID),
				zap.Error(cacheErr),
			)
		}
	}

	logger.Logger.Info("registration-account-creation success",
		zap.String("trace_id", traceID),
		zap.String("reference_no", referenceNo),
		zap.String("account_number", accountNumber),
		zap.String("name", accountHolder),
	)

	s.Publisher.AccountCreated(ctx, kafkapkg.AccountCreatedEvent{
		AccountID:     acc.ID.String(),
		AccountNumber: accountNumber,
		AccountHolder: accountHolder,
		PartnerRefNo:  req.PartnerReferenceNo,
		ReferenceNo:   referenceNo,
	})

	return result, nil
}

func (s *AccountService) GetAll(ctx context.Context) ([]dto.AccountResponse, error) {
	ctx, span := middleware.Tracer.Start(ctx, "AccountService.GetAll")
	defer span.End()

	logger.Logger.Info("getting all accounts", zap.String("trace_id", helper.GetTraceID(ctx)))
	accounts, err := s.Repo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]dto.AccountResponse, 0 , len(accounts))
	for _, acc := range accounts {
		result = append(result, dto.AccountResponse{
			ID: 		   acc.ID.String(),
			AccountNumber: acc.AccountNumber,
			AccountHolder: acc.AccountHolder,
			Balance:       acc.Balance,
			CreatedAt: 	   acc.CreatedAt,
			UpdatedAt:     acc.UpdatedAt,
		})
	}
	return result, nil
}

func (s *AccountService) GetByID(ctx context.Context, id string) (dto.AccountResponse, error) {
	ctx, span := middleware.Tracer.Start(ctx, "AccountService.GetByID")
	defer span.End()

	logger.Logger.Info("getting account by id", zap.String("trace_id", helper.GetTraceID(ctx)), zap.String("account_id", id))

	if id == "" {
		logger.Logger.Error("id is required")
		return dto.AccountResponse{}, errors.New("id required")
	}

	key := "bank:account:" + id

	val, err := config.RDB.Get(ctx, key).Result()
	if err == nil {
		logger.Logger.Info("cache hit", zap.String("key", key))
		var acc dto.AccountResponse
		if err := json.Unmarshal([]byte(val), &acc); err == nil {
			return acc, nil
		}
	} else if err == redis.Nil {
		logger.Logger.Warn("cache miss", zap.String("key", key))
	}
	
	uid, err := uuid.Parse(id)
	if err != nil {
		logger.Logger.Error("invalid id format", zap.String("id", id))
		return dto.AccountResponse{}, errors.New("invalid id format")
	}

	acc,err := s.Repo.GetByID(ctx,uid)
	if err != nil {
		logger.Logger.Error("failed to get account by id", zap.Error(err))
		return dto.AccountResponse{}, err
	}

	resp := dto.AccountResponse{
		ID:            acc.ID.String(),
		AccountNumber: acc.AccountNumber,
		AccountHolder: acc.AccountHolder,
		Balance:       acc.Balance,
		CreatedAt:     acc.CreatedAt,
		UpdatedAt:     acc.UpdatedAt,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		logger.Logger.Error("error marshaling account", zap.Error(err))
		return dto.AccountResponse{}, err
	}
	err = config.RDB.Set(ctx, key, data, 3*time.Minute).Err()

	if err != nil {
		logger.Logger.Error("error setting to redis", zap.Error(err))
	}

	return resp,nil
}

func (s *AccountService) Update(ctx context.Context, id uuid.UUID, req dto.UpdateAccountRequest) error {
	ctx, span := middleware.Tracer.Start(ctx, "AccountService.Update")
	defer span.End()

	logger.Logger.Info("updating account", zap.String("trace_id", helper.GetTraceID(ctx)), zap.String("account_id", id.String()))

	if id == uuid.Nil {
		logger.Logger.Error("id is required")
		return errors.New("id required")
	}
	if req.AccountHolder == "" {
		logger.Logger.Error("account holder is required")
		return errors.New("account_holder required")
	}
	if req.Balance < 0 {
		logger.Logger.Error("invalid balance", zap.Int64("balance", req.Balance))
		return errors.New("invalid balance")
	}

	acc := models.Account{ID: id, AccountHolder: req.AccountHolder, Balance: req.Balance}

	err := s.Repo.Update(ctx,acc)
	if err != nil {
		logger.Logger.Error("failed to update account", zap.Error(err))
		return err
	}

	key := "bank:account:" + acc.ID.String()
	err = config.RDB.Del(ctx, key).Err()
	if err != nil {
		logger.Logger.Error("error deleting from redis", zap.Error(err))
	}

	return nil
}

func (s *AccountService) Delete(ctx context.Context, id string) error {
	ctx, span := middleware.Tracer.Start(ctx, "AccountService.Delete")
	defer span.End()

	logger.Logger.Info("deleting account", zap.String("trace_id", helper.GetTraceID(ctx)), zap.String("account_id", id))

	if id == "" {
		logger.Logger.Error("id is required")
		return errors.New("id required")
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		logger.Logger.Error("invalid id format", zap.String("id", id))
		return errors.New("invalid id format")
	}

	err = s.Repo.Delete(ctx,uid)
	if err != nil {
		logger.Logger.Error("failed to delete account", zap.Error(err))
		return err
	}

	key := "bank:account:" + id
	err = config.RDB.Del(ctx, key).Err()
	if err != nil {
		logger.Logger.Error("error deleting from redis", zap.Error(err))
	}

	return nil
}

func (s *AccountService) Deposit(ctx context.Context, req dto.BalanceRequest) (dto.BalanceResponse, error) {
	ctx, span := middleware.Tracer.Start(ctx, "AccountService.Deposit")
	defer span.End()

	traceID := helper.GetTraceID(ctx)
	logger.Logger.Info("processing deposit",
		zap.String("trace_id", traceID),
		zap.String("account_number", req.AccountNumber),
		zap.Int64("amount", req.Amount),
	)

	if req.Amount <= 0 {
		return dto.BalanceResponse{}, errors.New("amount must be greater than 0")
	}

	acc, err := s.Repo.GetByAccountNumber(ctx, req.AccountNumber)
	if err != nil {
		logger.Logger.Error("account not found for deposit",
			zap.String("trace_id", traceID),
			zap.String("account_number", req.AccountNumber),
		)
		return dto.BalanceResponse{}, helper.ErrAccountNotFound
	}

	acc.Balance += req.Amount

	if err := s.Repo.Update(ctx, acc); err != nil {
		logger.Logger.Error("failed to update balance on deposit", zap.Error(err))
		return dto.BalanceResponse{}, err
	}

	// Invalidate cache
	config.RDB.Del(ctx, "bank:account:"+acc.ID.String())

	logger.Logger.Info("deposit success",
		zap.String("trace_id", traceID),
		zap.String("account_number", req.AccountNumber),
		zap.Int64("new_balance", acc.Balance),
	)

	s.Publisher.BalanceChanged(ctx, kafkapkg.BalanceChangedEvent{
		EventType:     "account.balance.deposited",
		AccountNumber: req.AccountNumber,
		AmountChanged: req.Amount,
		BalanceAfter:  acc.Balance,
		Remark:        req.Remark,
	})

	return dto.BalanceResponse{
		ResponseCode:    helper.SnapResponseCode(200, serviceCodeAccountCreation, "00"),
		ResponseMessage: "Request has been processed successfully",
		AccountNumber:   req.AccountNumber,
		Balance:         acc.Balance,
		Remark:          req.Remark,
	}, nil
}

func (s *AccountService) Withdraw(ctx context.Context, req dto.BalanceRequest) (dto.BalanceResponse, error) {
	ctx, span := middleware.Tracer.Start(ctx, "AccountService.Withdraw")
	defer span.End()

	traceID := helper.GetTraceID(ctx)
	logger.Logger.Info("processing withdrawal",
		zap.String("trace_id", traceID),
		zap.String("account_number", req.AccountNumber),
		zap.Int64("amount", req.Amount),
	)

	if req.Amount <= 0 {
		return dto.BalanceResponse{}, errors.New("amount must be greater than 0")
	}

	acc, err := s.Repo.GetByAccountNumber(ctx, req.AccountNumber)
	if err != nil {
		logger.Logger.Error("account not found for withdrawal",
			zap.String("trace_id", traceID),
			zap.String("account_number", req.AccountNumber),
		)
		return dto.BalanceResponse{}, helper.ErrAccountNotFound
	}

	if acc.Balance < req.Amount {
		logger.Logger.Error("insufficient balance for withdrawal",
			zap.String("trace_id", traceID),
			zap.Int64("balance", acc.Balance),
			zap.Int64("requested", req.Amount),
		)
		return dto.BalanceResponse{}, helper.ErrInsufficientFunds
	}

	acc.Balance -= req.Amount

	if err := s.Repo.Update(ctx, acc); err != nil {
		logger.Logger.Error("failed to update balance on withdrawal", zap.Error(err))
		return dto.BalanceResponse{}, err
	}

	// Invalidate cache
	config.RDB.Del(ctx, "bank:account:"+acc.ID.String())

	logger.Logger.Info("withdrawal success",
		zap.String("trace_id", traceID),
		zap.String("account_number", req.AccountNumber),
		zap.Int64("new_balance", acc.Balance),
	)

	s.Publisher.BalanceChanged(ctx, kafkapkg.BalanceChangedEvent{
		EventType:     "account.balance.withdrawn",
		AccountNumber: req.AccountNumber,
		AmountChanged: -req.Amount,
		BalanceAfter:  acc.Balance,
		Remark:        req.Remark,
	})

	return dto.BalanceResponse{
		ResponseCode:    helper.SnapResponseCode(200, serviceCodeAccountCreation, "00"),
		ResponseMessage: "Request has been processed successfully",
		AccountNumber:   req.AccountNumber,
		Balance:         acc.Balance,
		Remark:          req.Remark,
	}, nil
}
