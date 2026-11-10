package spotstrategy

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/beego/beego/v2/client/orm"
)

// Live orders need TWO independent permissions: the previously audited
// strategy mode (unavailable for switching in S1) and an operator-set,
// exact-match process approval. Neither SpotSymbols.Enable nor legacy
// automatic Spot switches authorize them.
const liveApprovalVariable = "SPOT_STRATEGY_LIVE_TRADING_APPROVAL"
const liveApprovalValue = "SPOT-S3-REAL-ORDERS-EXPLICITLY-APPROVED"

// S3 review F1: an environment variable plus an editable DB mode is NOT
// auditable approval. Until an operator-scoped, expiring, revocable grant,
// its immutable event history and legacy exit reconciliation are implemented,
// this build intentionally cannot authorize any real Spot POST/DELETE.
// A DB UPDATE to strategy plus the previously accepted environment string
// is therefore insufficient to enable live trading.
const spotRealOrderReleaseApproved = false

func LiveOrdersApproved() bool {
	return spotRealOrderReleaseApproved &&
		os.Getenv(liveApprovalVariable) == liveApprovalValue
}

// LiveStatus is the single fail-closed display projection of the real
// release gate, selected mode and signed Fill evidence. Do not hard-code
// "approved" in controllers or UI: when policy changes, all callers see
// the same decision. A release flag alone is NOT production readiness.
func LiveStatus(mode string) (execution string, liveApproved bool, reason string) {
	if !LiveOrdersApproved() {
		return "live_disabled", false, "spot_live_authorization_missing"
	}
	if mode != ModeStrategy {
		return "live_disabled", false, "spot_strategy_mode_not_active"
	}
	if !canAttestLiveFills(nil) {
		return "live_disabled", false, "spot_fill_provenance_unverified"
	}
	return "live_ready", true, ""
}

func AuthorizeLiveMode() error {
	if !LiveOrdersApproved() {
		return fmt.Errorf("%w: independent live Spot approval absent", ErrLiveWriteDisabled)
	}
	mode, err := CurrentMode()
	if err != nil {
		return fmt.Errorf("%w: Spot trade mode unavailable", ErrLiveWriteDisabled)
	}
	if mode != ModeStrategy {
		return fmt.Errorf("%w: Spot mode %s not strategy", ErrLiveWriteDisabled, mode)
	}
	return nil
}

// AuthorizeLiveWrite performs a second fail-closed check at the actual
// signed-HTTP boundary. For submission the order must have been reserved
// and claimed in the database BEFORE the POST, with immutable ClientOrderID.
// For cancellation, no arbitrary/manual/old order can be addressed.
func AuthorizeLiveWrite(ctx context.Context, req OrderRequest, cancelling bool) error {
	if err := AuthorizeLiveMode(); err != nil {
		return err
	}
	return validatePersistedSpotWrite(ctx, req, cancelling)
}

// validatePersistedSpotWrite verifies order ownership independently from
// release approval; production ALWAYS invokes it after AuthorizeLiveMode.
// Unit tests can exercise the strict order constraints with real SQL while
// the audit-based production grant is deliberately not implemented.
func validatePersistedSpotWrite(ctx context.Context, req OrderRequest, cancelling bool) error {
	if err := ValidateScope(req.AccountID, req.Owner); err != nil {
		return ErrUnmanagedOrder
	}
	if req.Symbol == "" || strings.ToUpper(req.Symbol) != req.Symbol || !clientIDPattern.MatchString(req.ClientOrderID) {
		return ErrUnmanagedOrder
	}
	o := orm.NewOrm()
	var enabled int
	if err := o.Raw("SELECT strategy_trade_enabled FROM spot_symbols WHERE symbol=?", req.Symbol).QueryRow(&enabled); err != nil || enabled != 1 {
		return fmt.Errorf("%w: Spot strategy symbol disabled", ErrLiveWriteDisabled)
	}
	var side, kind, status, qty string
	err := o.Raw(`SELECT side,order_type,status,CAST(requested_qty AS CHAR) FROM spot_strategy_orders
 WHERE account_id=? AND owner=? AND symbol=? AND client_order_id=?`, req.AccountID, req.Owner, req.Symbol, req.ClientOrderID).
		QueryRow(&side, &kind, &status, &qty)
	if err != nil {
		return ErrUnmanagedOrder
	}
	if cancelling {
		if status != StateNew && status != StatePartial {
			return ErrUnmanagedOrder
		}
	} else {
		if status != StateSubmitting {
			return fmt.Errorf("%w: order not submitting", ErrLiveWriteDisabled)
		}
		if req.Side != side || req.Type != kind {
			return ErrUnmanagedOrder
		}
		stored, err := parseAmount(qty, false)
		if err != nil {
			return ErrUnmanagedOrder
		}
		passed, err := parseAmount(req.Quantity, false)
		if err != nil || !passed.Equal(stored) {
			return ErrUnmanagedOrder
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

// A client configured to send signed requests is NOT evidence that any
// particular HTTP response was authenticated and reconciled. In particular,
// a BaseURL string, SDK constructor or caller-provided Fake cannot attest
// trade provenance. S3 deliberately leaves this closed until S4 implements
// per-response source, TLS/request correlation and audit persistence.
type signedRESTTradeEvidence interface {
	SpotSignedRESTProvenance() bool
}

func canAttestLiveFills(_ *Engine) bool {
	return false
}
