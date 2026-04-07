package repository

import (
	"context"
	"errors"
	"rest-api-bank/middleware"
	"rest-api-bank/models"
	"rest-api-bank/pkg/logger"
	"rest-api-bank/pkg/metrics"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type accountRepository struct {
	db *sqlx.DB
}

func NewAccountRepository(db *sqlx.DB) AccountRepository {
    return &accountRepository{db}
}

func (r *accountRepository) Create(ctx context.Context, acc models.Account) error {
	start := time.Now()

	ctx, span := middleware.Tracer.Start(ctx, "AccountRepository.Create")
	defer span.End()

	logger.Logger.Info("inserting account into database",
		zap.String("account_number", acc.AccountNumber),
		zap.String("account_holder", acc.AccountHolder),
		zap.Int64("balance", acc.Balance),
	)

	query := `
	INSERT INTO accounts (id, account_number, account_holder, balance)
	VALUES ($1,$2,$3,$4)
	`
	_, err := r.db.ExecContext(ctx, query, acc.ID, acc.AccountNumber, acc.AccountHolder, acc.Balance)

	metrics.DBQueryDuration.
			WithLabelValues("AccountRepository.Create").
			Observe(time.Since(start).Seconds())

	if err != nil {
		logger.Logger.Error("failed to insert account", zap.Error(err))
	}

	return err
}

func (r *accountRepository) GetAll(ctx context.Context) ([]models.Account, error) {
	start := time.Now()

	ctx, span := middleware.Tracer.Start(ctx, "AccountRepository.GetAll")
	defer span.End()

	logger.Logger.Info("getting all accounts")

	var accounts []models.Account
	err := r.db.SelectContext(ctx, &accounts, "SELECT * FROM accounts")

	if err != nil {
		logger.Logger.Error("failed to get accounts", zap.Error(err))
	}

	metrics.DBQueryDuration.
		WithLabelValues("AccountRepository.GetAll").
		Observe(time.Since(start).Seconds())

	return accounts, err
}

func (r *accountRepository) GetByID(ctx context.Context, id uuid.UUID) (models.Account, error) {
	start := time.Now()

	ctx, span := middleware.Tracer.Start(ctx, "AccountRepository.GetByID")
	defer span.End()

	logger.Logger.Info("getting account by id", zap.String("id", id.String()))

	var acc models.Account
	err := r.db.GetContext(ctx, &acc, "SELECT * FROM accounts WHERE id=$1", id)

	metrics.DBQueryDuration.
		WithLabelValues("AccountRepository.GetByID").
		Observe(time.Since(start).Seconds())

	if err != nil {
		logger.Logger.Error("failed to get account by id", zap.Error(err))

		return acc, errors.New("account not found")
	}

	return acc, nil
}

func (r *accountRepository) Update(ctx context.Context, acc models.Account) error {
	start := time.Now()
	
	ctx, span := middleware.Tracer.Start(ctx, "AccountRepository.Update")
	defer span.End()

	logger.Logger.Info("updating account",
		zap.String("id", acc.ID.String()),
		zap.String("account_holder", acc.AccountHolder),
		zap.Int64("balance", acc.Balance),
	)

	result, err := r.db.ExecContext(ctx, `
	UPDATE accounts SET account_holder=$1, balance=$2 WHERE id=$3
	`, acc.AccountHolder, acc.Balance, acc.ID)

	metrics.DBQueryDuration.
		WithLabelValues("AccountRepository.Update").
		Observe(time.Since(start).Seconds())

	if err != nil {
		logger.Logger.Error("failed to update account", zap.Error(err))
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		logger.Logger.Error("failed to get rows affected", zap.Error(err))
		return err
	}

	if rows == 0 {
		logger.Logger.Error("account not found", zap.String("id", acc.ID.String()))
		return errors.New("Account not found")
	}

	return err
}

func (r *accountRepository) Delete(ctx context.Context, id uuid.UUID) error {
	start := time.Now()

	ctx, span := middleware.Tracer.Start(ctx, "AccountRepository.Delete")
	defer span.End()

	logger.Logger.Info("deleting account", zap.String("id", id.String()))
	result, err := r.db.ExecContext(ctx, "DELETE FROM accounts WHERE id=$1", id)

	metrics.DBQueryDuration.
		WithLabelValues("AccountRepository.Delete").
		Observe(time.Since(start).Seconds())

	if err != nil {
		logger.Logger.Error("failed to delete account", zap.Error(err))
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		logger.Logger.Error("failed to get rows affected", zap.Error(err))
		return err
	}

	if rows == 0 {
		logger.Logger.Error("account not found", zap.String("id", id.String()))
		return errors.New("Account not found")
	}
	return err
}
