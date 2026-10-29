package controllers

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	futuresownership "go_binance_futures/service/futuresownership"
)

func TestManualClaimErrorCodeDoesNotExposeInternalDetails(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{futuresownership.ErrManualPositionNotFound, "manual_claim_position_not_found"},
		{futuresownership.ErrManualAccountQuantityChanged, "manual_claim_account_quantity_changed"},
		{futuresownership.ErrManualDuplicateExchangePosition, "manual_claim_duplicate_exchange_position"},
		{futuresownership.ErrManualInvalidEntryPrice, "manual_claim_invalid_entry_price"},
		{fmt.Errorf("wrapped: %w", futuresownership.ErrManualPositionNotFound), "manual_claim_position_not_found"},
		{errors.New("mysql access denied: super-secret sql"), "manual_claim_rejected"},
		{errors.New("Binance upstream internal error"), "manual_claim_rejected"},
	}
	for _, tc := range cases {
		if got := manualClaimErrorCode(tc.err); got != tc.want {
			t.Fatalf("code(%v)=%s, want %s", tc.err, got, tc.want)
		}
	}
}

func TestManualClaimRequestLengthAndSyntaxLimits(t *testing.T) {
	valid := []byte(`{"symbol":"HYPEUSDT","position_side":"SHORT","owner":"auto_strategy","quantity":1.04,"expected_account_qty":1.04,"reason":"reviewed trading fills"}`)
	r, err := decodeManualClaimRequest(valid)
	if err != nil || r.Symbol != "HYPEUSDT" || r.Quantity != 1.04 {
		t.Fatalf("valid request rejected: %+v err=%v", r, err)
	}
	for _, body := range [][]byte{nil, []byte{}, []byte("{"), []byte("nullx"), []byte(strings.Repeat(" ", manualClaimRequestMaxBytes+1))} {
		if _, err := decodeManualClaimRequest(body); err == nil {
			t.Fatalf("invalid request size=%d accepted", len(body))
		}
	}
}
