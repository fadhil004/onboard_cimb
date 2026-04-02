package service

import (
	"context"
	"encoding/json"
	"errors"
	"rest-api-bank/config"
	"rest-api-bank/helper"
	"rest-api-bank/middleware"
	"rest-api-bank/models"
	"rest-api-bank/pkg/logger"
	"rest-api-bank/repository"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type AccountService struct {
	Repo repository.AccountRepository
}

func (s *AccountService) Create(ctx context.Context, acc models.Account) error {
	ctx, span := middleware.Tracer.Start(ctx, "AccountService.Create")
	defer span.End()

	logger.Logger.Info("creating account",
		zap.String("trace_id", helper.GetTraceID(ctx)),
		zap.String("account_number", acc.AccountNumber),
		zap.String("account_holder", acc.AccountHolder),
		zap.Int64("balance", acc.Balance),
	)

	if acc.AccountNumber == "" {
		err := errors.New("account_number required")
		logger.Logger.Error("validation failed", zap.Error(err))
		return err
	}
	if acc.AccountHolder == "" {
		err := errors.New("account_holder required")
		logger.Logger.Error("validation failed", zap.Error(err))
		return err
	}
	if acc.Balance < 0 {
		err := errors.New("invalid balance")
		logger.Logger.Error("validation failed", zap.Error(err))
		return err
	}

	return s.Repo.Create(ctx,acc)
}

func (s *AccountService) GetAll(ctx context.Context) ([]models.Account, error) {
	ctx, span := middleware.Tracer.Start(ctx, "AccountService.GetAll")
	defer span.End()

	logger.Logger.Info("getting all accounts", zap.String("trace_id", helper.GetTraceID(ctx)))
	return s.Repo.GetAll(ctx)
}

func (s *AccountService) GetByID(ctx context.Context, id string) (models.Account, error) {
	ctx, span := middleware.Tracer.Start(ctx, "AccountService.GetByID")
	defer span.End()

	logger.Logger.Info("getting account by id", zap.String("trace_id", helper.GetTraceID(ctx)), zap.String("account_id", id))

	if id == "" {
		logger.Logger.Error("id is required")
		return models.Account{}, errors.New("id required")
	}

	key := "bank:account:" + id

	val, err := config.RDB.Get(ctx, key).Result()
	if err == nil {
		logger.Logger.Info("cache hit", zap.String("key", key))
		var acc models.Account
		if err := json.Unmarshal([]byte(val), &acc); err == nil {
			return acc, nil
		}
	} else if err == redis.Nil {
		logger.Logger.Warn("cache miss", zap.String("key", key))
		// log.Println("error getting from redis (fallback db):", err)
	}
	
	uid, err := uuid.Parse(id)
	if err != nil {
		logger.Logger.Error("invalid id format", zap.String("id", id))
		return models.Account{}, errors.New("invalid id format")
	}

	acc,err := s.Repo.GetByID(ctx,uid)
	if err != nil {
		logger.Logger.Error("failed to get account by id", zap.Error(err))
		return acc, err
	}

	data, err := json.Marshal(acc)
	if err != nil {
		logger.Logger.Error("error marshaling account", zap.Error(err))
		return acc, err
	}
	// simpan ke redis, set expired 3 menit
	err = config.RDB.Set(ctx, key, data, 3*time.Minute).Err()

	if err != nil {
		logger.Logger.Error("error setting to redis", zap.Error(err))
		// log.Println("error setting to redis:", err)
	}

	return acc,nil
}

func (s *AccountService) Update(ctx context.Context, acc models.Account) error {
	ctx, span := middleware.Tracer.Start(ctx, "AccountService.Update")
	defer span.End()

	logger.Logger.Info("updating account", zap.String("trace_id", helper.GetTraceID(ctx)), zap.String("account_id", acc.ID.String()))

	if acc.ID == uuid.Nil {
		logger.Logger.Error("id is required")
		return errors.New("id required")
	}
	if acc.AccountHolder == "" {
		logger.Logger.Error("account holder is required")
		return errors.New("account_holder required")
	}
	if acc.Balance < 0 {
		logger.Logger.Error("invalid balance", zap.Int64("balance", acc.Balance))
		return errors.New("invalid balance")
	}

	err := s.Repo.Update(ctx,acc)
	if err != nil {
		logger.Logger.Error("failed to update account", zap.Error(err))
		return err
	}

	key := "bank:account:" + acc.ID.String()
	err = config.RDB.Del(ctx, key).Err()
	if err != nil {
		logger.Logger.Error("error deleting from redis", zap.Error(err))
		// log.Println("error deleting from redis:", err)
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
		// log.Println("error deleting from redis:", err)
	}

	return nil
}