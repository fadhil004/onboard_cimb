package dto

import "time"

type TransferRequest struct {
	FromAccountID string `json:"from_account_id"`
	ToAccountID   string `json:"to_account_id"`
	Amount        int64  `json:"amount"`
	Status        string `json:"status,omitempty"`
	Message       string `json:"message,omitempty"`
}

// TransactionResponse dipakai untuk GET /accounts/{id}/transactions
type TransactionResponse struct {
	ID            string    `json:"id"`
	FromAccountID string    `json:"fromAccountId"`
	ToAccountID   string    `json:"toAccountId"`
	Amount        int64     `json:"amount"`
	Remark        string    `json:"remark"`
	CreatedAt     time.Time `json:"createdAt"`
}
