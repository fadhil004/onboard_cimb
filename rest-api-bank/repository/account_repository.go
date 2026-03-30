package repository

import (
	"rest-api-bank/models"

	"github.com/google/uuid"
)

type AccountRepository interface {
    Create(acc models.Account) error
    GetAll() ([]models.Account, error)
    GetByID(id uuid.UUID) (models.Account, error)
    Update(acc models.Account) error
    Delete(id uuid.UUID) error
}
