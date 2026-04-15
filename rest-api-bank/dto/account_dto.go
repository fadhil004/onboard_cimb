package dto

import "time"

type DeviceInfo struct {
	OS           string `json:"os"`
	OSVersion    string `json:"osVersion"`
	Model        string `json:"model"`
	Manufacturer string `json:"manufacture"`
}

type CreateAccountRequest struct {
	PartnerReferenceNo string                 `json:"partnerReferenceNo"`
	CountryCode        string                 `json:"countryCode"`
	CustomerID         string                 `json:"customerId"`
	DeviceInfo         DeviceInfo             `json:"deviceInfo"`
	Email              string                 `json:"email"`
	Lang               string                 `json:"lang"`
	Locale             string                 `json:"locale"`
	Name               string                 `json:"name"`
	OnboardingPartner  string                 `json:"onboardingPartner"`
	PhoneNo            string                 `json:"phoneNo"`
	RedirectURL        string                 `json:"redirectUrl"`
	Scopes             string                 `json:"scopes"`
	SeamlessData       string                 `json:"seamlessData"`
	SeamlessSign       string                 `json:"seamlessSign"`
	State              string                 `json:"state"`
	MerchantID         string                 `json:"merchantId"`
	SubMerchantID      string                 `json:"subMerchantId"`
	TerminalType       string                 `json:"terminalType"`
	AdditionalInfo     map[string]interface{} `json:"additionalInfo"`
}

type CreateAccountResponse struct {
	ResponseCode       string                 `json:"responseCode"`
	ResponseMessage    string                 `json:"responseMessage"`
	ReferenceNo        string                 `json:"referenceNo,omitempty"`
	PartnerReferenceNo string                 `json:"partnerReferenceNo,omitempty"`
	AuthCode           string                 `json:"authCode,omitempty"`
	APIKey             string                 `json:"apiKey,omitempty"`
	AccountID          string                 `json:"accountId,omitempty"`
	AccountNumber      string                 `json:"accountNumber,omitempty"`
	State              string                 `json:"state,omitempty"`
	AdditionalInfo     map[string]interface{} `json:"additionalInfo,omitempty"`
}

type UpdateAccountRequest struct {
	AccountHolder string `json:"account_holder"`
	Balance       int64  `json:"balance"`
}

type BalanceRequest struct {
	AccountNumber string `json:"accountNumber"`
	Amount        int64  `json:"amount"`
	Remark        string `json:"remark"`
}

type BalanceResponse struct {
	ResponseCode    string `json:"responseCode"`
	ResponseMessage string `json:"responseMessage"`
	AccountNumber   string `json:"accountNumber"`
	Balance         int64  `json:"balance"`
	Remark          string `json:"remark,omitempty"`
}

type AccountResponse struct {
	ID            string    `json:"id"`
	AccountNumber string    `json:"accountNumber"`
	AccountHolder string    `json:"accountHolder"`
	Balance       int64     `json:"balance"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}