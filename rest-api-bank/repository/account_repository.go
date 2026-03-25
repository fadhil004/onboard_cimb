package repository

import (
	"errors"
	"rest-api-bank/models"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type AccountRepository struct {
	DB *sqlx.DB
}

func (r *AccountRepository) Create(acc models.Account) error {
	query := `
	INSERT INTO accounts (id, account_number, account_holder, balance)
	VALUES ($1,$2,$3,$4)
	`
	_, err := r.DB.Exec(query, acc.ID, acc.AccountNumber, acc.AccountHolder, acc.Balance)
	return err
}

func (r *AccountRepository) GetAll() ([]models.Account, error) {
	var accounts []models.Account
	err := r.DB.Select(&accounts, "SELECT * FROM accounts")
	return accounts, err
}

func (r *AccountRepository) GetByID(id uuid.UUID) (models.Account, error) {
	var acc models.Account
	err := r.DB.Get(&acc, "SELECT * FROM accounts WHERE id=$1", id)

	if err != nil {
		return acc, errors.New("account not found")
	}

	return acc, nil
}

func (r *AccountRepository) Update(acc models.Account) error {
	result, err := r.DB.Exec(`
	UPDATE accounts SET account_holder=$1, balance=$2 WHERE id=$3
	`, acc.AccountHolder, acc.Balance, acc.ID)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return errors.New("Account not found")
	}

	return err
}

func (r *AccountRepository) Delete(id uuid.UUID) error {
	_, err := r.DB.Exec("DELETE FROM accounts WHERE id=$1", id)
	return err
}
