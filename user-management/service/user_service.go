package service

import (
	"user-management/models"
	"user-management/repository"
)

type UserService struct {
	Repo *repository.UserRepository
}

func (s *UserService) GetUsers() ([]models.User, error) {
	return s.Repo.GetAll()
}

func (s *UserService) GetUser(id int) (*models.User, error) {
	return s.Repo.GetByID(id)
}

func (s *UserService) CreateUser(u models.User) error {
	return s.Repo.Create(u)
}

func (s *UserService) UpdateUser(u models.User) error {
	return s.Repo.Update(u)
}

func (s *UserService) DeleteUser(id int) error {
	return s.Repo.DeleteById(id)
}