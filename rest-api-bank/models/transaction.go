package models

import "github.com/google/uuid"

type Transaction struct {
	ID            uuid.UUID `db:"id"`
	FromAccountID uuid.UUID `db:"from_account_id"`
	ToAccountID   uuid.UUID `db:"to_account_id"`
	Amount        int64     `db:"amount"`
}