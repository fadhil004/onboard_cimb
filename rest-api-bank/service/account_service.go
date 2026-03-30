package service

import (
	"errors"
	"rest-api-bank/helper"
	"rest-api-bank/models"
	"rest-api-bank/repository"

	"github.com/google/uuid"
)

type AccountService struct {
	Repo repository.AccountRepository
}

func (s *AccountService) Create(acc models.Account) error {
	if acc.AccountNumber == "" {
		return errors.New("account_number required")
	}
	if acc.AccountHolder == "" {
		return errors.New("account_holder required")
	}
	if acc.Balance < 0 {
		return errors.New("invalid balance")
	}

	return s.Repo.Create(acc)
}

func (s *AccountService) GetAll() ([]models.Account, error) {
	return s.Repo.GetAll()
}

func (s *AccountService) GetByID(id string) (models.Account, error) {
	if id == "" {
		return models.Account{}, errors.New("id required")
	}

	uid := helper.UuidMustParse(id)
	return s.Repo.GetByID(uid)
}

func (s *AccountService) Update(acc models.Account) error {
	if acc.ID == uuid.Nil {
		return errors.New("id required")
	}
	if acc.AccountHolder == "" {
		return errors.New("account_holder required")
	}
	if acc.Balance < 0 {
		return errors.New("invalid balance")
	}

	return s.Repo.Update(acc)
}

func (s *AccountService) Delete(id string) error {
	if id == "" {
		return errors.New("id required")
	}

	uid := helper.UuidMustParse(id)
	return s.Repo.Delete(uid)
}