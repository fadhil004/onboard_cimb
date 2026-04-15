package models

import (
	"time"

	"github.com/google/uuid"
)

type Account struct {
	ID            uuid.UUID `db:"id"             json:"id"`
	AccountNumber string    `db:"account_number" json:"account_number"`
	AccountHolder string    `db:"account_holder" json:"account_holder"`
	Balance       int64     `db:"balance"        json:"balance"`
	CreatedAt     time.Time `db:"created_at"     json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"     json:"updated_at"`
}