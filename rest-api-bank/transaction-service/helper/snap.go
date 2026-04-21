package helper

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidField      = errors.New("invalid field")
	ErrMandatoryField    = errors.New("mandatory field missing")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrInvalidToken      = errors.New("invalid token")
	ErrInsufficientFunds = errors.New("insufficient balance")
	ErrAccountNotFound   = errors.New("account not found")
	ErrDuplicate         = errors.New("duplicate transaction")
	ErrSupectedFraud     = errors.New("suspected fraud")
	ErrCardBlocked 		 = errors.New("card blocked")
	ErrBeneficiaryBlocked = errors.New("beneficiary blocked")
)

func SnapResponseCode(httpcode int, serviceCode string, caseCode string) string {
	return fmt.Sprintf("%03d%s%s", httpcode, serviceCode, caseCode)
}

func MapSnapError(err error, serviceCode string) (string, string, int) {
	switch err {
	case ErrMandatoryField:
		return SnapResponseCode(400, serviceCode, "02"), "Invalid Mandatory Field", 400
	case ErrInvalidField:
		return SnapResponseCode(400, serviceCode, "01"), "Invalid Field Format", 400
	case ErrUnauthorized:
		return SnapResponseCode(401, serviceCode, "00"), "Unauthorized", 401
	case ErrInvalidToken:
		return SnapResponseCode(401, serviceCode, "01"), "Invalid Token (B2B)", 401
	case ErrInsufficientFunds:
		return SnapResponseCode(403, serviceCode, "14"), "Insufficient Funds", 403
	case ErrAccountNotFound:
		return SnapResponseCode(404, serviceCode, "11"), "Invalid Account", 404
	case ErrDuplicate:
		return SnapResponseCode(409, serviceCode, "01"), "Duplicate partnerReferenceNo", 409
	case ErrSupectedFraud:
		return SnapResponseCode(403, serviceCode, "03"), "Suspected Fraud", 403
	case ErrCardBlocked:
		return SnapResponseCode(403, serviceCode, "07"), "Card Blocked", 403
	case ErrBeneficiaryBlocked:
		return SnapResponseCode(403, serviceCode, "08"), "Beneficiary Blocked", 403
	default:
		return SnapResponseCode(500, serviceCode, "00"), "General Error", 500
	}
}
