package controllers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/beego/beego/v2/server/web/context"
)

func TestCM7RuntimeSummaryRequiresAuthenticatedUser(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/coin-m/runtime-status", nil)
	ctx := context.NewContext()
	ctx.Reset(recorder, req)
	controller := new(CoinMQuantController)
	controller.Init(ctx, "CoinMQuantController", "GetRuntimeSummary", nil)
	controller.GetRuntimeSummary()
	if recorder.Code != http.StatusUnauthorized ||
		!strings.Contains(recorder.Body.String(), "coin_m_auth_required") {
		t.Fatalf("anonymous COIN-M runtime summary leaked local account status: %d %s", recorder.Code, recorder.Body.String())
	}
}
