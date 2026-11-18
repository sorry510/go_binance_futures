package controllers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/beego/beego/v2/server/web/context"
)

func TestCM4WalletRequiresJWTBeforeAnyDAPIRequest(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/coin-m/wallet", nil)
	ctx := context.NewContext()
	ctx.Reset(recorder, req)
	c := new(CoinMQuantController)
	c.Init(ctx, "CoinMQuantController", "GetWallet", nil)
	c.GetWallet()
	if recorder.Code != http.StatusUnauthorized || !strings.Contains(recorder.Body.String(), "coin_m_auth_required") {
		t.Fatalf("anonymous COIN-M wallet request not rejected: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
