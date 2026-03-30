package service

import (
	"errors"
	"rest-api-bank/helper"
	"rest-api-bank/models"
	"rest-api-bank/repository"

	"github.com/google/uuid"
)

type TransferService struct {
	AccountRepo     repository.AccountRepository
	TransactionRepo repository.TransactionRepository
}

func (s *TransferService) Transfer(fromID, toID uuid.UUID, amount int64) error {

	// VALIDATION
	if fromID == toID {
		return errors.New("cannot transfer to same account")
	}

	if amount <= 0 {
		return errors.New("invalid amount")
	}

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

	// SIMPAN STATE AWAL (rollback manual sederhana)
	originalFrom := from
	originalTo := to

	from.Balance -= amount
	to.Balance += amount

	// update from
	if err := s.AccountRepo.Update(from); err != nil {
		return err
	}

	// update to
	if err := s.AccountRepo.Update(to); err != nil {
		// rollback manual
		_ = s.AccountRepo.Update(originalFrom)
		return err
	}

	tx := models.Transaction{
		ID:            uuid.New(),
		FromAccountID: fromID,
		ToAccountID:   toID,
		Amount:        amount,
	}

	if err := s.TransactionRepo.Create(tx); err != nil {
		// rollback
		_ = s.AccountRepo.Update(originalFrom)
		_ = s.AccountRepo.Update(originalTo)
		return err
	}

	return nil
}

func (s *TransferService) GetTransaction(id string) ([]models.Transaction, error) {
	return s.TransactionRepo.GetByAccountID(helper.UuidMustParse(id))
}