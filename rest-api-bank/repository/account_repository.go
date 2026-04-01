package repository

import (
	"context"
	"rest-api-bank/models"

	"github.com/google/uuid"
)

type AccountRepository interface {
    Create(ctx context.Context, acc models.Account) error
    GetAll(ctx context.Context) ([]models.Account, error)
    GetByID(ctx context.Context, id uuid.UUID) (models.Account, error)
    Update(ctx context.Context, acc models.Account) error
    Delete(ctx context.Context, id uuid.UUID) error
}
