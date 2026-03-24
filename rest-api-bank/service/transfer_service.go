package service

import (
	"errors"
	"rest-api-bank/helper"
	"rest-api-bank/models"
	"rest-api-bank/repository"

	"github.com/google/uuid"
)

type TransferService struct {
	AccountRepo     *repository.AccountRepository
	TransactionRepo *repository.TransactionRepository
}

func (s *TransferService) Transfer(fromID, toID uuid.UUID, amount int64) error {

	from, err := s.AccountRepo.GetByID(fromID)
	if err != nil {
		return err
	}

	to, err := s.AccountRepo.GetByID(toID)
	if err != nil {
		return err
	}

	if from.Balance < amount {
		return errors.New("insufficient balance")
	}

	from.Balance -= amount
	to.Balance += amount

	err = s.AccountRepo.Update(from)
	if err != nil {
		return err
	}

	err = s.AccountRepo.Update(to)
	if err != nil {
		return err
	}

	tx := models.Transaction{
		ID:            uuid.New(),
		FromAccountID: fromID,
		ToAccountID:   toID,
		Amount:        amount,
	}

	return s.TransactionRepo.Create(tx)
}

func (s *TransferService) GetTransaction(id string) ([]models.Transaction, error) {
	return s.TransactionRepo.GetByAccountID(helper.UuidMustParse(id))
}