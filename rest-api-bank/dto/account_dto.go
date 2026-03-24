package dto

type CreateAccountRequest struct {
	AccountNumber string `json:"account_number"`
	AccountHolder string `json:"account_holder"`
	Balance       int64  `json:"balance"`
}

type UpdateAccountRequest struct {
	AccountHolder string `json:"account_holder"`
	Balance       int64  `json:"balance"`
}