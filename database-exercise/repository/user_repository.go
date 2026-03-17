package repository

import (
	"database-exercise/models"
)

type UserRepository interface {
	GetAll() ([]models.User, error)
	GetByID(id int) (*models.User, error)
	Create(user models.User) error
}