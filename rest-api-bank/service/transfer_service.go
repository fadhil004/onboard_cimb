package service

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"rest-api-bank/config"
	"rest-api-bank/dto"
	"rest-api-bank/helper"
	"rest-api-bank/models"
	"rest-api-bank/repository"
	"time"

	"github.com/google/uuid"
)

type TransferService struct {
	AccountRepo     repository.AccountRepository
	TransactionRepo repository.TransactionRepository
}

func (s *TransferService) Transfer(ctx context.Context, idemKey string, req dto.TransferRequest) (dto.TransferRequest, error) {
	key := "bank:transfer:" + idemKey

	val, err := config.RDB.Get(ctx, key).Result()
	if err == nil {
		var cached dto.TransferRequest
		err := json.Unmarshal([]byte(val), &cached)
		if err == nil {
			cached.Message = "Duplicate request - returning cached result"
			return cached, nil
		}
	}

	if err.Error() != "redis: nil" {
		log.Println("error getting from redis (fallback db):", err)
	}

	err = s.processTransfer(ctx, req)
	if err != nil {
		return dto.TransferRequest{}, err
	}

	result := dto.TransferRequest{
		Status:        "success",
		Message:       "Transfer successful",
	}
	
	data, err := json.Marshal(result)
	if err != nil {
		log.Println("failed to marshal result:", err)
		return dto.TransferRequest{}, err
	}

	err = config.RDB.Set(ctx, key, data, 5*time.Minute).Err() // simpan hasil ke redis, set expired 5 menit
	if err != nil {
		log.Println("error setting to redis:", err)
	}

	return result, nil
}

func (s *TransferService) processTransfer(ctx context.Context, req dto.TransferRequest) error {
	from, err := s.AccountRepo.GetByID(ctx, helper.UuidMustParse(req.FromAccountID))
	if err != nil {
		return err
	}

	to, err := s.AccountRepo.GetByID(ctx, helper.UuidMustParse(req.ToAccountID))
	if err != nil {
		return err
	}

	if from.Balance < req.Amount {
		return errors.New("insufficient balance")
	}

	from.Balance -= req.Amount
	to.Balance += req.Amount

	err = s.AccountRepo.Update(ctx, from)
	if err != nil {
		return err
	}

	err = s.AccountRepo.Update(ctx, to)
	if err != nil {
		return err
	}

	tx := models.Transaction{
		ID:            uuid.New(),
		FromAccountID: from.ID,
		ToAccountID:   to.ID,
		Amount:        req.Amount,
	}

	return s.TransactionRepo.Create(ctx, tx)
}

func (s *TransferService) GetTransaction(ctx context.Context, id string) ([]models.Transaction, error) {
	return s.TransactionRepo.GetByAccountID(ctx, helper.UuidMustParse(id))
}