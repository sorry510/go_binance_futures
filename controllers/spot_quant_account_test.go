package controllers

import (
	"fmt"
	"go_binance_futures/service/spotledger"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/beego/beego/v2/client/orm"
	beegocontext "github.com/beego/beego/v2/server/web/context"
	"github.com/golang-jwt/jwt/v5"
)

func spotAccountControllerTest(method, route, body, id string) (*SpotQuantAccountController, *httptest.ResponseRecorder) {
	request := httptest.NewRequest(method, route, strings.NewReader(body))
	request.Header.Set("Accept", "application/xml")
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	ctx := beegocontext.NewContext()
	ctx.Reset(recorder, request)
	ctx.Input.SetData("user", jwt.MapClaims{"Username": "single_operator"})
	ctx.Input.RequestBody = []byte(body)
	ctx.Input.SetParam(":id", id)
	ctrl := new(SpotQuantAccountController)
	ctrl.Init(ctx, "SpotQuantAccountController", "", nil)
	return ctrl, recorder
}
func TestStage065SpotAccountInvalidIdentifiersAreRejectedBeforeDB(t *testing.T) {
	for _, invalid := range []string{"-1", "not-id", "0", "1 OR 1=1"} {
		ctrl, rec := spotAccountControllerTest(http.MethodGet, "/spot-quant/positions/bad", "", invalid)
		ctrl.PositionDetail()
		if rec.Code != 400 || !strings.Contains(rec.Body.String(), "invalid Spot position id") {
			t.Fatalf("invalid id %s returned %d %s", invalid, rec.Code, rec.Body.String())
		}
	}
}
func TestStage065SpotAccountRebuildRequiresExplicitConfirmation(t *testing.T) {
	for _, body := range []string{"{}", "{\"confirm\":false}", "{\"confirm\":null}", "{", ""} {
		ctrl, rec := spotAccountControllerTest(http.MethodPost, "/spot-quant/positions/1/rebuild", body, "1")
		ctrl.LocalRebuild()
		if rec.Code != 400 || !strings.Contains(rec.Body.String(), "confirm=true") {
			t.Fatalf("unsafe local mutation request %q returned %d %s", body, rec.Code, rec.Body.String())
		}
	}
}

func TestReview065SpotAccountRejectsMissingOperatorClaim(t *testing.T) {
	for _, tc := range []struct {
		name   string
		method string
		run    func(*SpotQuantAccountController)
	}{
		{"positions", http.MethodGet, func(c *SpotQuantAccountController) { c.Positions() }},
		{"detail", http.MethodGet, func(c *SpotQuantAccountController) { c.PositionDetail() }},
		{"orders", http.MethodGet, func(c *SpotQuantAccountController) { c.Orders() }},
		{"rebuild", http.MethodPost, func(c *SpotQuantAccountController) { c.LocalRebuild() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, rec := spotAccountControllerTest(tc.method, "/spot-quant/positions/1", "{}", "1")
			c.Ctx.Input.SetData("user", nil)
			tc.run(c)
			if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "spot_quant_auth_required") {
				t.Fatalf("unauthenticated %s leaked Spot info: %d %s", tc.name, rec.Code, rec.Body.String())
			}
		})
	}
}
func TestReview065SummaryTooLargeReturnsActionable422(t *testing.T) {
	c, rec := spotAccountControllerTest(http.MethodGet, "/spot-quant/positions", "", "1")
	c.respondRead(nil, spotledger.ErrSummaryTooLarge)
	if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), "safe limit") && !strings.Contains(rec.Body.String(), "summary exceeds") {
		t.Fatalf("unbounded summary not conveyed to client: %d %s", rec.Code, rec.Body.String())
	}
}

// 06-5-F5: distinguish operator-fixable, missing and internal errors without
// leaking database-driver internals in a JSON response.
func TestReview065SpotAccountHTTPErrorClassification(t *testing.T) {
	for _, tc := range []struct {
		err     error
		status  int
		message string
	}{
		{orm.ErrNoRows, http.StatusNotFound, "not found"},
		{spotledger.ErrNeedsReview, http.StatusConflict, "reconciliation"},
		{fmt.Errorf("mysql connection details should remain private"), http.StatusInternalServerError, "spot ledger unavailable"},
	} {
		c, rec := spotAccountControllerTest(http.MethodGet, "/spot-quant/positions", "", "1")
		c.respondRead(nil, tc.err)
		if rec.Code != tc.status || !strings.Contains(rec.Body.String(), tc.message) {
			t.Fatalf("error=%v => status=%d response=%s", tc.err, rec.Code, rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), "mysql connection details") {
			t.Fatalf("internal database details leaked to client: %s", rec.Body.String())
		}
	}
}
