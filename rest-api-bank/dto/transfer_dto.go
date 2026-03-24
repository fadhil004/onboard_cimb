package dto

type TransferRequest struct {
	FromAccountID string `json:"from_account_id"`
	ToAccountID   string `json:"to_account_id"`
	Amount        int64  `json:"amount"`
}