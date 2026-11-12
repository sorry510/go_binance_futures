package controllers

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go_binance_futures/service/spotquant"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/server/web"
	"github.com/expr-lang/expr"
	"github.com/golang-jwt/jwt/v5"
)

type SpotQuantController struct{ web.Controller }

func (ctrl *SpotQuantController) respond(status int, value interface{}, err error) {
	if err != nil {
		ctrl.Ctx.Output.SetStatus(status)
		_ = ctrl.JSONResp(map[string]interface{}{"code": status, "data": nil, "msg": err.Error()})
		return
	}
	_ = ctrl.JSONResp(map[string]interface{}{"code": 200, "data": value, "msg": "success"})
}

// authorized is a defense-in-depth check beyond JwtMiddleware. None of
// the Spot quant controller routes is public. Require a validated
// operator Username claim even if a future router misses the global filter.
func (ctrl *SpotQuantController) authorized() bool {
	claims, ok := ctrl.Ctx.Input.GetData("user").(jwt.MapClaims)
	if !ok {
		ctrl.respond(http.StatusUnauthorized, nil, fmt.Errorf("spot_quant_auth_required"))
		return false
	}
	username, ok := claims["Username"].(string)
	if !ok || strings.TrimSpace(username) == "" {
		ctrl.respond(http.StatusUnauthorized, nil, fmt.Errorf("spot_quant_auth_required"))
		return false
	}
	return true
}

func (ctrl *SpotQuantController) intParam(name string, fallback int) int {
	value, err := strconv.Atoi(ctrl.GetString(name))
	if err != nil {
		return fallback
	}
	return value
}
func (ctrl *SpotQuantController) id() (int64, error) {
	id, err := strconv.ParseInt(ctrl.Ctx.Input.Param(":id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid record id")
	}
	return id, nil
}
func (ctrl *SpotQuantController) symbols() {
	page, err := spotquant.ListSymbols(orm.NewOrm(), spotquant.SymbolFilters{
		Symbol: ctrl.GetString("symbol"), Quote: ctrl.GetString("symbol_type"),
		Enabled: ctrl.GetString("enable"), Pin: ctrl.GetString("pin") == "1",
		Sort: ctrl.GetString("sort"), Page: ctrl.intParam("page", 1),
		Limit: ctrl.intParam("limit", 20),
	})
	ctrl.respond(http.StatusBadRequest, page, err)
}

func (ctrl *SpotQuantController) ListSymbols() {
	if !ctrl.authorized() {
		return
	}
	ctrl.symbols()
}
func (ctrl *SpotQuantController) GetSymbol() {
	if !ctrl.authorized() {
		return
	}
	id, err := ctrl.id()
	if err != nil {
		ctrl.respond(400, nil, err)
		return
	}
	row, err := spotquant.GetSymbol(orm.NewOrm(), id)
	if err != nil {
		ctrl.respond(404, nil, err)
		return
	}
	ctrl.respond(200, row, nil)
}
func (ctrl *SpotQuantController) CreateSymbol() {
	if !ctrl.authorized() {
		return
	}
	var request struct {
		Symbol string `json:"symbol"`
	}
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &request); err != nil {
		ctrl.respond(400, nil, err)
		return
	}
	row, err := spotquant.CreateSymbol(orm.NewOrm(), request.Symbol)
	ctrl.respond(400, row, err)
}
func (ctrl *SpotQuantController) EditSymbol() {
	if !ctrl.authorized() {
		return
	}
	id, err := ctrl.id()
	if err != nil {
		ctrl.respond(400, nil, err)
		return
	}
	row, err := spotquant.ApplySymbolPatch(orm.NewOrm(), id, ctrl.Ctx.Input.RequestBody)
	ctrl.respond(400, row, err)
}
func (ctrl *SpotQuantController) BatchSymbols() {
	if !ctrl.authorized() {
		return
	}
	var body struct {
		IDs                []int64         `json:"ids"`
		SpotQuantEnabled   *int            `json:"spot_quant_enabled"`
		Usdt               json.RawMessage `json:"usdt"`
		Profit             json.RawMessage `json:"profit"`
		Loss               json.RawMessage `json:"loss"`
		StrategyTemplateID int64           `json:"strategy_template_id"`
	}
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &body); err != nil {
		ctrl.respond(400, nil, err)
		return
	}
	patch := map[string]interface{}{}
	if body.SpotQuantEnabled != nil {
		patch["spot_quant_enabled"] = *body.SpotQuantEnabled
	}
	for name, value := range map[string]json.RawMessage{"usdt": body.Usdt, "profit": body.Profit, "loss": body.Loss} {
		if len(value) != 0 {
			patch[name] = value
		}
	}
	// Apply a template by copying its Spot-specific JSON into each selected
	// symbol; never persist a reference to the Futures template table.
	if body.StrategyTemplateID > 0 {
		template, err := spotquant.GetTemplate(orm.NewOrm(), body.StrategyTemplateID)
		if err != nil {
			ctrl.respond(400, nil, err)
			return
		}
		patch["technology"] = template.Technology
		patch["strategy"] = template.Strategy
	}
	if len(patch) == 0 {
		ctrl.respond(400, nil, fmt.Errorf("empty batch update"))
		return
	}
	raw, err := json.Marshal(patch)
	if err != nil {
		ctrl.respond(400, nil, err)
		return
	}
	n, err := spotquant.BatchSymbolPatch(orm.NewOrm(), body.IDs, raw)
	ctrl.respond(400, map[string]interface{}{"updated": n}, err)
}
func (ctrl *SpotQuantController) EnableSymbols() {
	if !ctrl.authorized() {
		return
	}
	flag, err := strconv.Atoi(ctrl.Ctx.Input.Param(":flag"))
	if err != nil {
		ctrl.respond(400, nil, fmt.Errorf("invalid flag"))
		return
	}
	n, err := spotquant.UpdateSymbolsEnabled(orm.NewOrm(), flag)
	ctrl.respond(400, map[string]interface{}{"updated": n}, err)
}

func (ctrl *SpotQuantController) ListTemplates() {
	if !ctrl.authorized() {
		return
	}
	page, err := spotquant.ListTemplates(orm.NewOrm(), ctrl.GetString("name"), ctrl.intParam("page", 1), ctrl.intParam("limit", 20))
	ctrl.respond(400, page, err)
}
func (ctrl *SpotQuantController) GetTemplate() {
	if !ctrl.authorized() {
		return
	}
	id, err := ctrl.id()
	if err != nil {
		ctrl.respond(400, nil, err)
		return
	}
	row, err := spotquant.GetTemplate(orm.NewOrm(), id)
	if err != nil {
		ctrl.respond(404, nil, err)
		return
	}
	ctrl.respond(200, row, nil)
}
func (ctrl *SpotQuantController) CreateTemplate() {
	if !ctrl.authorized() {
		return
	}
	row, err := spotquant.CreateTemplate(orm.NewOrm(), ctrl.Ctx.Input.RequestBody)
	ctrl.respond(400, row, err)
}
func (ctrl *SpotQuantController) EditTemplate() {
	if !ctrl.authorized() {
		return
	}
	id, err := ctrl.id()
	if err != nil {
		ctrl.respond(400, nil, err)
		return
	}
	row, err := spotquant.EditTemplate(orm.NewOrm(), id, ctrl.Ctx.Input.RequestBody)
	ctrl.respond(400, row, err)
}
func (ctrl *SpotQuantController) DeleteTemplate() {
	if !ctrl.authorized() {
		return
	}
	id, err := ctrl.id()
	if err != nil {
		ctrl.respond(400, nil, err)
		return
	}
	err = spotquant.DeleteTemplate(orm.NewOrm(), id)
	ctrl.respond(400, nil, err)
}
func (ctrl *SpotQuantController) ImportTemplate() {
	if !ctrl.authorized() {
		return
	}
	row, action, err := spotquant.ImportTemplate(orm.NewOrm(), ctrl.Ctx.Input.RequestBody)
	ctrl.respond(400, map[string]interface{}{"template": row, "action": action}, err)
}

// SpotRuleTest is a synthetic preview, never a Binance REST/WS order.
// Expressions involving historical indicators may only pass syntax validation
// until the Spot-only indicator runtime is implemented.
func (ctrl *SpotQuantController) SpotRuleTest() {
	if !ctrl.authorized() {
		return
	}
	var payload struct {
		Strategy   json.RawMessage `json:"strategy"`
		Technology json.RawMessage `json:"technology"`
		Price      json.Number     `json:"price"`
	}
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &payload); err != nil {
		ctrl.respond(400, nil, err)
		return
	}
	price := 100.0
	priceSource := "synthetic_default"
	var savedPriceTime int64
	// The symbol endpoint uses the LOCAL Spot ticker, not the Futures
	// ticker or Futures PositionRisk. Template previews use sample price.
	if ctrl.Ctx.Input.Param(":id") != "" {
		id, err := ctrl.id()
		if err != nil {
			ctrl.respond(400, nil, err)
			return
		}
		row, err := spotquant.GetSymbol(orm.NewOrm(), id)
		if err != nil {
			ctrl.respond(404, nil, err)
			return
		}
		if len(payload.Strategy) == 0 && row.Strategy != "" {
			payload.Strategy, _ = json.Marshal(row.Strategy)
		}
		if len(payload.Technology) == 0 && row.Technology != "" {
			payload.Technology, _ = json.Marshal(row.Technology)
		}
		if n, err := strconv.ParseFloat(row.Close, 64); err == nil && n > 0 && !math.IsNaN(n) && !math.IsInf(n, 0) {
			price = n
			priceSource = "spot_symbols.close"
			savedPriceTime = row.UpdateTime
		}
	}
	strategy, err := spotquant.ParseJSONField(payload.Strategy, false)
	if err != nil {
		ctrl.respond(400, nil, err)
		return
	}
	technology, err := spotquant.ParseJSONField(payload.Technology, true)
	if err != nil {
		ctrl.respond(400, nil, err)
		return
	}
	if err := spotquant.ValidateTechnology(technology); err != nil {
		ctrl.respond(400, nil, err)
		return
	}
	rules, err := spotquant.ParseRules(strategy)
	if err != nil {
		ctrl.respond(400, nil, err)
		return
	}
	var rule *spotquant.Rule
	for i := range rules {
		if rules[i].Enable {
			rule = &rules[i]
			break
		}
	}
	if rule == nil {
		ctrl.respond(400, nil, fmt.Errorf("enable a LONG or CLOSE_LONG rule to preview"))
		return
	}
	if payload.Price != "" {
		if price, err = payload.Price.Float64(); err != nil || price <= 0 || math.IsNaN(price) || math.IsInf(price, 0) {
			ctrl.respond(400, nil, fmt.Errorf("price must be a positive finite number"))
			return
		}
		priceSource = "user_input"
		savedPriceTime = 0
	}
	env := map[string]interface{}{
		"NowPrice": price, "NowSymbolClose": price, "NowSymbolOpen": price,
		"NowSymbolLow": price, "NowSymbolHigh": price, "NowSymbolPercentChange": 0.0,
		"NowTime": time.Now().UnixMilli(),
	}
	program, err := expr.Compile(strings.TrimSpace(rule.Code), expr.Env(env), expr.AllowUndefinedVariables(), expr.AsBool())
	if err != nil {
		ctrl.respond(400, nil, fmt.Errorf("rule compilation: %w", err))
		return
	}
	out, err := expr.Run(program, env)
	if err != nil {
		// Indicators require actual Spot candles. A syntactically valid
		// expression is NOT a live trading decision.
		ctrl.respond(200, map[string]interface{}{
			"valid": true, "pass": nil, "type": rule.Type, "mode": "syntax_only",
			"price_source": priceSource, "spot_price_time": savedPriceTime,
			"detail": err.Error(),
		}, nil)
		return
	}
	ctrl.respond(200, map[string]interface{}{
		"valid": true, "pass": out, "type": rule.Type, "mode": "synthetic",
		"price": price, "price_source": priceSource, "spot_price_time": savedPriceTime,
	}, nil)
}
