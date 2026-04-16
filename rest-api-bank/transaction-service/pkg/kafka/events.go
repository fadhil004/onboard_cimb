package kafka

import "time"

// Transaction event
type TransactionCreatedEvent struct {
	EventID              string    `json:"eventId"`
	EventType            string    `json:"eventType"`
	TransactionID        string    `json:"transactionId"`
	SourceAccountNo      string    `json:"sourceAccountNo"`
	BeneficiaryAccountNo string    `json:"beneficiaryAccountNo"`
	AmountValue          string    `json:"amountValue"`
	Currency             string    `json:"currency"`
	Remark               string    `json:"remark"`
	Status               string    `json:"status"`
	PartnerRefNo         string    `json:"partnerRefNo"`
	ReferenceNo          string    `json:"referenceNo"`
	OccurredAt           time.Time `json:"occurredAt"`
}
