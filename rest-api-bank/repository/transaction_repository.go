package repository

import (
	"context"
	"rest-api-bank/models"

	"github.com/google/uuid"
)

type TransactionRepository interface {
	Create(ctx context.Context, tx models.Transaction) error
	GetByAccountID(ctx context.Context, id uuid.UUID) ([]models.Transaction, error)
}