package repository

import (
	"rest-api-bank/models"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type TransactionRepository struct {
	DB *sqlx.DB
}

func (r *TransactionRepository) Create(tx models.Transaction) error {
	_, err := r.DB.Exec(`
	INSERT INTO transactions (id, from_account_id, to_account_id, amount)
	VALUES ($1,$2,$3,$4)
	`, tx.ID, tx.FromAccountID, tx.ToAccountID, tx.Amount)

	return err
}

func (r *TransactionRepository) GetByAccountID(id uuid.UUID) ([]models.Transaction, error) {
	var txs []models.Transaction

	err := r.DB.Select(&txs, `
	SELECT * FROM transactions 
	WHERE from_account_id=$1 OR to_account_id=$1
	`, id)

	return txs, err
}