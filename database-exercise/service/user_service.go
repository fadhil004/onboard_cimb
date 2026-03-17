package service

import (
	"database-exercise/models"
	"database-exercise/repository"
	"fmt"
)

type UserService struct {
	repo repository.UserRepository
}

func NewUserService(r repository.UserRepository) *UserService {
	return &UserService{repo: r}
}

func (s *UserService) GetAll() ([]models.User, error) {
	return s.repo.GetAll()
}

func (s *UserService) GetByID(id int) (*models.User, error) {
	user, err := s.repo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return user, nil
}

func (s *UserService) Create(name, email string) error {

	if name == "" || email == "" {
		return fmt.Errorf("name and email required")
	}

	user := models.User{
		Name:  name,
		Email: email,
	}

	return s.repo.Create(user)
}