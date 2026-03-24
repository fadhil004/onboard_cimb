package service

import (
	"errors"
	"rest-api-bank/helper"
	"rest-api-bank/models"
	"rest-api-bank/repository"
)

type AccountService struct {
	Repo *repository.AccountRepository
}

func (s *AccountService) Create(acc models.Account) error {

	if acc.AccountNumber == "" {
		return errors.New("account_number required")
	}

	return s.Repo.Create(acc)
}

func (s *AccountService) GetAll() ([]models.Account, error) {
	return s.Repo.GetAll()
}

func (s *AccountService) GetByID(id string) (models.Account, error) {
	return s.Repo.GetByID(helper.UuidMustParse(id))
}

func (s *AccountService) Update(acc models.Account) error {
	return s.Repo.Update(acc)
}

func (s *AccountService) Delete(id string) error {
	return s.Repo.Delete(helper.UuidMustParse(id))
}
