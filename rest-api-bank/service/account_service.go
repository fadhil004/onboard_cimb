package service

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"rest-api-bank/config"
	"rest-api-bank/helper"
	"rest-api-bank/models"
	"rest-api-bank/repository"
	"time"

	"github.com/google/uuid"
)

type AccountService struct {
	Repo repository.AccountRepository
}

func (s *AccountService) Create(ctx context.Context, acc models.Account) error {
	if acc.AccountNumber == "" {
		return errors.New("account_number required")
	}
	if acc.AccountHolder == "" {
		return errors.New("account_holder required")
	}
	if acc.Balance < 0 {
		return errors.New("invalid balance")
	}

	return s.Repo.Create(ctx,acc)
}

func (s *AccountService) GetAll(ctx context.Context) ([]models.Account, error) {
	return s.Repo.GetAll(ctx)
}

func (s *AccountService) GetByID(ctx context.Context, id string) (models.Account, error) {
	if id == "" {
		return models.Account{}, errors.New("id required")
	}

	key := "bank:account:" + id

	val, err := config.RDB.Get(ctx, key).Result()
	if err == nil {
		var acc models.Account
		if err := json.Unmarshal([]byte(val), &acc); err == nil {
			return acc, nil
		}
	}

	if err != nil {
		log.Println("error getting from redis (fallback db):", err)
	}
	
	uid := helper.UuidMustParse(id)

	acc,err := s.Repo.GetByID(ctx,uid)
	if err != nil {
		return acc, err
	}

	data,_ := json.Marshal(acc)
	// simpan ke redis, set expired 3 menit
	err = config.RDB.Set(ctx, key, data, 3*time.Minute).Err()

	if err != nil {
		log.Println("error setting to redis:", err)
	}

	return acc,nil
}

func (s *AccountService) Update(ctx context.Context, acc models.Account) error {
	if acc.ID == uuid.Nil {
		return errors.New("id required")
	}
	if acc.AccountHolder == "" {
		return errors.New("account_holder required")
	}
	if acc.Balance < 0 {
		return errors.New("invalid balance")
	}

	err := s.Repo.Update(ctx,acc)
	if err != nil {
		return err
	}

	key := "bank:account:" + acc.ID.String()
	err = config.RDB.Del(ctx, key).Err()
	if err != nil {
		log.Println("error deleting from redis:", err)
	}

	return nil
}

func (s *AccountService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("id required")
	}
	uid := helper.UuidMustParse(id)

	err := s.Repo.Delete(ctx,uid)
	if err != nil {
		return err
	}

	key := "bank:account:" + id
	err = config.RDB.Del(ctx, key).Err()
	if err != nil {
		log.Println("error deleting from redis:", err)
	}

	return nil
}