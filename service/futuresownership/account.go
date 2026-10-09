package futuresownership

import (
	"fmt"
	"strings"

	"go_binance_futures/feature/api/binance"
)

// BindAccount intentionally rejects unknown account IDs. Legacy zero-value
// Service and DefaultService remain explicitly main-only for compatibility.
func BindAccount(id binance.AccountID) (Service, error) {
	if id != binance.MainAccountID && id != binance.LeadAccountID {
		return Service{}, fmt.Errorf("unsupported ownership account %q", id)
	}
	return Service{AccountID: string(id)}, nil
}

func (s Service) accountID() (string, error) {
	id := strings.TrimSpace(s.AccountID)
	if id == "" {
		return string(binance.MainAccountID), nil
	}
	if id != string(binance.MainAccountID) && id != string(binance.LeadAccountID) {
		return "", fmt.Errorf("invalid ownership account %q", id)
	}
	return id, nil
}
