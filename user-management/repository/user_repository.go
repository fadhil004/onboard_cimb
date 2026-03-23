package repository

import (
	"user-management/models"

	"gorm.io/gorm"
)

type UserRepository struct {
	DB *gorm.DB
}

func (r *UserRepository) GetAll() ([]models.User, error) {
	var users []models.User
	err := r.DB.Find(&users).Error
	return users, err
}

func (r *UserRepository) GetByID(id int) (*models.User, error) {
	var user models.User
	err := r.DB.First(&user, id).Error
	return &user, err
}

func (r *UserRepository) Create(u models.User) error {
	return r.DB.Create(&u).Error
}

func (r *UserRepository) Update(u models.User) error {
	return r.DB.Save(&u).Error
}

func (r *UserRepository) DeleteById(id int) error {
	return r.DB.Delete(&models.User{}, id).Error
}