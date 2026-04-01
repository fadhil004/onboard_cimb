package repository

import (
	"context"
	"rest-api-bank/models"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type transactionRepository struct {
	db *sqlx.DB
}

func NewTransactionRepository(db *sqlx.DB) TransactionRepository {
	return &transactionRepository{db}
}

func (r *transactionRepository) Create(ctx context.Context, tx models.Transaction) error {
	_, err := r.db.ExecContext(ctx, `
	INSERT INTO transactions (id, from_account_id, to_account_id, amount)
	VALUES ($1,$2,$3,$4)
	`, tx.ID, tx.FromAccountID, tx.ToAccountID, tx.Amount)

	return err
}

func (r *transactionRepository) GetByAccountID(ctx context.Context, id uuid.UUID) ([]models.Transaction, error) {
	var txs []models.Transaction

	err := r.db.SelectContext(ctx, &txs, `
	SELECT * FROM transactions 
	WHERE from_account_id=$1 OR to_account_id=$1
	`, id)

	return txs, err
}