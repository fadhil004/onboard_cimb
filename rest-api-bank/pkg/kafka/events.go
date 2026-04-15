package kafka

import "time"

// Account Creation
type AccountCreatedEvent struct {
	EventID        string    `json:"eventId"`        	
	EventType      string    `json:"eventType"`      	
	AccountID      string    `json:"accountId"`      
	AccountNumber  string    `json:"accountNumber"`  
	AccountHolder  string    `json:"accountHolder"`  
	PartnerRefNo   string    `json:"partnerRefNo"`   
	ReferenceNo    string    `json:"referenceNo"`    
	OccurredAt     time.Time `json:"occurredAt"`     
}

// Transaction 
type TransactionCreatedEvent struct {
	EventID              string    `json:"eventId"`
	EventType            string    `json:"eventType"`            
	TransactionID        string    `json:"transactionId"`        
	SourceAccountNo      string    `json:"sourceAccountNo"`
	BeneficiaryAccountNo string    `json:"beneficiaryAccountNo"`
	AmountValue          string    `json:"amountValue"`          
	Currency             string    `json:"currency"`
	Remark               string    `json:"remark"`
	PartnerRefNo         string    `json:"partnerRefNo"`
	ReferenceNo          string    `json:"referenceNo"`
	OccurredAt           time.Time `json:"occurredAt"`
}

// Balance Change 

type BalanceChangedEvent struct {
	EventID       string    `json:"eventId"`
	EventType     string    `json:"eventType"`     // "account.balance.deposited" | "account.balance.withdrawn"
	AccountNumber string    `json:"accountNumber"`
	AmountChanged int64     `json:"amountChanged"` // positif=deposit, negatif=withdraw
	BalanceAfter  int64     `json:"balanceAfter"`
	Remark        string    `json:"remark"`
	OccurredAt    time.Time `json:"occurredAt"`
}
