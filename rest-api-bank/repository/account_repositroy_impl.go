package repository

import (
	"errors"
	"rest-api-bank/models"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type accountRepository struct {
	db *sqlx.DB
}

func NewAccountRepository(db *sqlx.DB) AccountRepository {
    return &accountRepository{db}
}

func (r *accountRepository) Create(acc models.Account) error {
	query := `
	INSERT INTO accounts (id, account_number, account_holder, balance)
	VALUES ($1,$2,$3,$4)
	`
	_, err := r.db.Exec(query, acc.ID, acc.AccountNumber, acc.AccountHolder, acc.Balance)
	return err
}

func (r *accountRepository) GetAll() ([]models.Account, error) {
	var accounts []models.Account
	err := r.db.Select(&accounts, "SELECT * FROM accounts")
	return accounts, err
}

func (r *accountRepository) GetByID(id uuid.UUID) (models.Account, error) {
	var acc models.Account
	err := r.db.Get(&acc, "SELECT * FROM accounts WHERE id=$1", id)

	if err != nil {
		return acc, errors.New("account not found")
	}

	return acc, nil
}

func (r *accountRepository) Update(acc models.Account) error {
	result, err := r.db.Exec(`
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

func (r *accountRepository) Delete(id uuid.UUID) error {
	result, err := r.db.Exec("DELETE FROM accounts WHERE id=$1", id)

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
