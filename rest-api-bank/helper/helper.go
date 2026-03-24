package helper

import (
	"strings"

	"github.com/google/uuid"
)

func GetIDFromPath(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

func UuidMustParse(id string) uuid.UUID {
	u, _ := uuid.Parse(id)
	return u
}

func GetIDFromTransactionPath(path string) string {
	parts := strings.Split(path, "/")
	// /accounts/{id}/transactions → id di index 2
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}