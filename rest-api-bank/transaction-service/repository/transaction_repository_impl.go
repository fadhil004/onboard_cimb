package repository

import (
	"context"
	"transaction-service/helper"
	"transaction-service/middleware"
	"transaction-service/models"
	"transaction-service/pkg/logger"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type transactionRepository struct {
	db *sqlx.DB
}

func NewTransactionRepository(db *sqlx.DB) TransactionRepository {
	return &transactionRepository{db}
}

func (r *transactionRepository) Create(ctx context.Context, tx models.Transaction) error {
	ctx, span := middleware.Tracer.Start(ctx, "TransactionRepository.Create")
	defer span.End()

	logger.Logger.Info("inserting transaction into database",
		zap.String("trace_id", helper.GetTraceID(ctx)),
		zap.String("from_account_id", tx.FromAccountID.String()),
		zap.String("to_account_id", tx.ToAccountID.String()),
		zap.Int64("amount", tx.Amount),
	)

	_, err := r.db.ExecContext(ctx, `
	INSERT INTO transactions (id, from_account_id, to_account_id, amount, remark, status)
	VALUES ($1,$2,$3,$4,$5,$6)
	`, tx.ID, tx.FromAccountID, tx.ToAccountID, tx.Amount, tx.Remark, tx.Status)

	if err != nil {
		logger.Logger.Error("failed to insert transaction", zap.Error(err))
	}

	return err
}

func (r *transactionRepository) GetByAccountID(ctx context.Context, id uuid.UUID) ([]models.Transaction, error) {
	ctx, span := middleware.Tracer.Start(ctx, "TransactionRepository.GetByAccountID")
	defer span.End()

	logger.Logger.Info("getting transactions for account",
		zap.String("trace_id", helper.GetTraceID(ctx)),
		zap.String("account_id", id.String()),
	)

	var txs []models.Transaction

	err := r.db.SelectContext(ctx, &txs, `
	SELECT * FROM transactions 
	WHERE from_account_id=$1 OR to_account_id=$1
	ORDER BY created_at DESC
	`, id)

	if err != nil {
		logger.Logger.Error("failed to get transactions", zap.Error(err))
	}

	return txs, err
}
