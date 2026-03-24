package models

import "github.com/google/uuid"

type Account struct {
	ID            uuid.UUID `db:"id"`
	AccountNumber string    `db:"account_number"`
	AccountHolder string    `db:"account_holder"`
	Balance       int64     `db:"balance"`
}