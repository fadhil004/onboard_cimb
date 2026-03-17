package repository

import (
	"database-exercise/models"

	"gorm.io/gorm"
)

type UserGORM struct {
	db *gorm.DB
}

func NewUserGORM(db *gorm.DB) *UserGORM {
	return &UserGORM{db: db}
}

func (r *UserGORM) GetAll() ([]models.User, error) {
	var users []models.User
	err := r.db.Find(&users).Error
	return users, err
}

func (r *UserGORM) GetByID(id int) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, id).Error
	return &user, err
}

func (r *UserGORM) Create(user models.User) error {
	return r.db.Create(&user).Error
}