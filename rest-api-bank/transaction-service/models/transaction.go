package models

import (
	"time"

	"github.com/google/uuid"
)

type Transaction struct {
	ID            uuid.UUID `db:"id"              json:"id"`
	FromAccountID uuid.UUID `db:"from_account_id" json:"from_account_id"`
	ToAccountID   uuid.UUID `db:"to_account_id"   json:"to_account_id"`
	Amount        int64     `db:"amount"          json:"amount"`
	Remark        string    `db:"remark"          json:"remark"`
	Status        string    `db:"status"          json:"status"`
	CreatedAt     time.Time `db:"created_at"      json:"created_at"`
}
