package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	beegocontext "github.com/beego/beego/v2/server/web/context"
	"github.com/golang-jwt/jwt/v5"
)

func spotQuantTestController(method, route, body, accept string) (*SpotQuantController, *httptest.ResponseRecorder) {
	request := httptest.NewRequest(method, route, strings.NewReader(body))
	request.Header.Set("Accept", accept)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	ctx := beegocontext.NewContext()
	ctx.Reset(recorder, request)
	ctx.Input.SetData("user", jwt.MapClaims{"Username": "single_operator"})
	ctx.Input.RequestBody = []byte(body)
	ctrl := new(SpotQuantController)
	ctrl.Init(ctx, "SpotQuantController", "", nil)
	return ctrl, recorder
}

func TestStage062SpotRuleSyntheticPreviewNeverTrading(t *testing.T) {
	body := `{"strategy":[{"name":"buy","type":"long","enable":true,"code":"NowPrice > 10"}],"price":100}`
	ctrl, rec := spotQuantTestController(http.MethodPost, "/spot-quant/strategy-templates/rule-test", body, "application/x-protobuf")
	ctrl.SpotRuleTest()
	if rec.Code != 200 || !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("preview returned status=%d type=%q: %s", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
	}
	var result struct {
		Data struct {
			Pass        bool   `json:"pass"`
			Mode        string `json:"mode"`
			PriceSource string `json:"price_source"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.Data.Pass || result.Data.Mode != "synthetic" || result.Data.PriceSource != "user_input" {
		t.Fatalf("preview falsely claimed live pricing: %+v", result.Data)
	}
}

func TestStage062SpotRuleBadDirectionsRejectedEvenDisabled(t *testing.T) {
	for _, kind := range []string{"short", "close_short"} {
		body := `{"strategy":[{"name":"no","type":"` + kind + `","enable":false,"code":"true"}]}`
		ctrl, rec := spotQuantTestController(http.MethodPost, "/spot-quant/strategy-templates/rule-test", body, "application/json")
		ctrl.SpotRuleTest()
		if rec.Code != 400 || !strings.Contains(rec.Body.String(), "not allowed for Spot") {
			t.Fatalf("direction %s unexpectedly allowed: %d %s", kind, rec.Code, rec.Body.String())
		}
	}
}

func TestStage062SpotRuleUnknownIndicatorSyntaxOnly(t *testing.T) {
	body := `{"strategy":[{"name":"buy","type":"long","enable":true,"code":"NonexistentIndicator > 10"}]}`
	ctrl, rec := spotQuantTestController(http.MethodPost, "/spot-quant/strategy-templates/rule-test", body, "application/json")
	ctrl.SpotRuleTest()
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "syntax_only") || !strings.Contains(rec.Body.String(), "\"pass\":null") {
		t.Fatalf("undefined indicator presented as a live signal: %d %s", rec.Code, rec.Body.String())
	}
}

func TestStage062SpotInvalidEndpointsDoNotAccessDatabase(t *testing.T) {
	t.Run("create malformed", func(t *testing.T) {
		ctrl, rec := spotQuantTestController(http.MethodPost, "/spot-quant/symbols", "{", "application/xml")
		ctrl.CreateSymbol()
		if rec.Code != 400 || !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
			t.Fatalf("malformed body did not produce fixed JSON error: %d %s", rec.Code, rec.Body.String())
		}
	})
	t.Run("invalid global flag", func(t *testing.T) {
		ctrl, rec := spotQuantTestController(http.MethodPut, "/spot-quant/symbols/enable/not-a-number", "", "application/json")
		ctrl.Ctx.Input.SetParam(":flag", "not-a-number")
		ctrl.EnableSymbols()
		if rec.Code != 400 {
			t.Fatalf("invalid flag: %d %s", rec.Code, rec.Body.String())
		}
	})
	t.Run("invalid id", func(t *testing.T) {
		ctrl, rec := spotQuantTestController(http.MethodGet, "/spot-quant/symbols/-1", "", "application/json")
		ctrl.Ctx.Input.SetParam(":id", "-1")
		ctrl.GetSymbol()
		if rec.Code != 400 {
			t.Fatalf("invalid symbol id: %d %s", rec.Code, rec.Body.String())
		}
	})
}

func TestStage062SpotPreviewRejectsFuturesOnlyVariables(t *testing.T) {
	body := `{"strategy":[{"name":"buy","type":"long","enable":true,"code":"ROI > 1"}]}`
	ctrl, rec := spotQuantTestController(http.MethodPost, "/spot-quant/strategy-templates/rule-test", body, "application/json")
	ctrl.SpotRuleTest()
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), "futures-only") {
		t.Fatalf("Futures-only preview should be rejected: %d %s", rec.Code, rec.Body.String())
	}
}

func TestReview065SpotQuantControllerRejectsMissingOperatorClaim(t *testing.T) {
	for _, tc := range []struct {
		name string
		run  func(*SpotQuantController)
	}{
		{"symbols", func(c *SpotQuantController) { c.ListSymbols() }},
		{"create", func(c *SpotQuantController) { c.CreateSymbol() }},
		{"rule preview", func(c *SpotQuantController) { c.SpotRuleTest() }},
		{"import", func(c *SpotQuantController) { c.ImportTemplate() }},
		{"batch", func(c *SpotQuantController) { c.BatchSymbols() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, rec := spotQuantTestController(http.MethodPost, "/spot-quant/test", "{}", "application/json")
			c.Ctx.Input.SetData("user", nil)
			tc.run(c)
			if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "spot_quant_auth_required") {
				t.Fatalf("unauthenticated %s accessed Spot API: status=%d response=%s", tc.name, rec.Code, rec.Body.String())
			}
			c, rec = spotQuantTestController(http.MethodPost, "/spot-quant/test", "{}", "application/json")
			c.Ctx.Input.SetData("user", jwt.MapClaims{"Username": " "})
			tc.run(c)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("blank username authorized %s: %d", tc.name, rec.Code)
			}
		})
	}
}
