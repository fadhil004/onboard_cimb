package repository

import (
	"rest-api-bank/models"

	"github.com/google/uuid"
)

type TransactionRepository interface {
	Create(tx models.Transaction) error
	GetByAccountID(id uuid.UUID) ([]models.Transaction, error)
}