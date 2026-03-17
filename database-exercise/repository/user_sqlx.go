package repository

import (
	"database-exercise/models"

	"github.com/jmoiron/sqlx"
)

type UserSQLX struct {
	db *sqlx.DB
}

func NewUserSQLX(db *sqlx.DB) *UserSQLX {
	return &UserSQLX{db: db}
}

func (r *UserSQLX) GetAll() ([]models.User, error) {
	var users []models.User
	err := r.db.Select(&users, "SELECT * FROM users")
	return users, err
}

func (r *UserSQLX) GetByID(id int) (*models.User, error) {
	var user models.User
	err := r.db.Get(&user, "SELECT * FROM users WHERE id=$1", id)
	return &user, err
}

func (r *UserSQLX) Create(user models.User) error {
	_, err := r.db.Exec(
		"INSERT INTO users (name, email) VALUES ($1, $2)",
		user.Name, user.Email,
	)
	return err
}