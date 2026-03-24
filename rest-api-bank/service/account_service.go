package service

import (
	"errors"
	"rest-api-bank/models"
	"rest-api-bank/repository"

	"github.com/google/uuid"
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
	return s.Repo.GetByID(uuidMustParse(id))
}

func uuidMustParse(id string) uuid.UUID {
	u, _ := uuid.Parse(id)
	return u
}