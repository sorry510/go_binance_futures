package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/server/web"
	"github.com/golang-jwt/jwt/v5"
	"go_binance_futures/service/coinmquant"
)

// CM-3 exposes symbol/template configuration and local-only ledger reads.
// There are deliberately NO DAPI order, cancel, position-claim or mode writes.
type CoinMQuantController struct{ web.Controller }

func (c *CoinMQuantController) reply(status int, data interface{}, err error) {
	if err != nil {
		c.Ctx.Output.SetStatus(status)
		_ = c.JSONResp(map[string]interface{}{"code": status, "data": nil, "msg": err.Error()})
		return
	}
	_ = c.JSONResp(map[string]interface{}{"code": 200, "data": data, "msg": "success"})
}
func (c *CoinMQuantController) auth() bool {
	claim, ok := c.Ctx.Input.GetData("user").(jwt.MapClaims)
	if !ok || strings.TrimSpace(fmt.Sprint(claim["Username"])) == "" || claim["Username"] == nil {
		c.reply(http.StatusUnauthorized, nil, fmt.Errorf("coin_m_auth_required"))
		return false
	}
	if _, ok := claim["Username"].(string); !ok {
		c.reply(http.StatusUnauthorized, nil, fmt.Errorf("coin_m_auth_required"))
		return false
	}
	return true
}
func (c *CoinMQuantController) num(name string, def int) int {
	n, err := strconv.Atoi(c.GetString(name))
	if err != nil {
		return def
	}
	return n
}
func (c *CoinMQuantController) id() (int64, error) {
	id, err := strconv.ParseInt(c.Ctx.Input.Param(":id"), 10, 64)
	if err != nil || id < 1 {
		return 0, fmt.Errorf("invalid CM record ID")
	}
	return id, nil
}
func (c *CoinMQuantController) ListSymbols() {
	if !c.auth() {
		return
	}
	page, err := coinmquant.Symbols(orm.NewOrm(), coinmquant.SymbolFilters{
		Symbol: c.GetString("symbol"), Enabled: c.GetString("enable"),
		Sort: c.GetString("sort"), Pin: c.GetString("pin") == "1",
		Page: c.num("page", 1), Limit: c.num("limit", 20),
	})
	c.reply(400, page, err)
}
func (c *CoinMQuantController) GetSymbol() {
	if !c.auth() {
		return
	}
	id, err := c.id()
	if err != nil {
		c.reply(400, nil, err)
		return
	}
	row, err := coinmquant.Symbol(orm.NewOrm(), id)
	c.reply(400, row, err)
}
func (c *CoinMQuantController) EditSymbol() {
	if !c.auth() {
		return
	}
	id, err := c.id()
	if err != nil {
		c.reply(400, nil, err)
		return
	}
	row, err := coinmquant.PatchSymbol(orm.NewOrm(), id, c.Ctx.Input.RequestBody)
	c.reply(400, row, err)
}
func (c *CoinMQuantController) BatchSymbols() {
	if !c.auth() {
		return
	}
	var req struct {
		IDs        []int64         `json:"ids"`
		Enable     *int            `json:"enable"`
		Budget     json.RawMessage `json:"budget_usd"`
		Profit     json.RawMessage `json:"profit"`
		Loss       json.RawMessage `json:"loss"`
		TemplateID int64           `json:"strategy_template_id"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.reply(400, nil, err)
		return
	}
	patch := map[string]interface{}{}
	if req.Enable != nil {
		patch["enable"] = *req.Enable
	}
	for k, v := range map[string]json.RawMessage{"budget_usd": req.Budget, "profit": req.Profit, "loss": req.Loss} {
		if len(v) > 0 {
			patch[k] = v
		}
	}
	if req.TemplateID > 0 {
		template, err := coinmquant.Template(orm.NewOrm(), req.TemplateID)
		if err != nil {
			c.reply(400, nil, err)
			return
		}
		patch["technology"] = template.Technology
		patch["strategy"] = template.Strategy
	}
	if len(patch) == 0 {
		c.reply(400, nil, fmt.Errorf("empty batch edit"))
		return
	}
	payload, _ := json.Marshal(patch)
	n, err := coinmquant.BatchSymbols(orm.NewOrm(), req.IDs, payload)
	c.reply(400, map[string]int{"updated": n}, err)
}
func (c *CoinMQuantController) EnableSymbols() {
	if !c.auth() {
		return
	}
	flag, err := strconv.Atoi(c.Ctx.Input.Param(":flag"))
	if err != nil {
		c.reply(400, nil, fmt.Errorf("invalid enable flag"))
		return
	}
	n, err := coinmquant.EnableAll(orm.NewOrm(), flag)
	c.reply(400, map[string]int64{"updated": n}, err)
}
func (c *CoinMQuantController) ListTemplates() {
	if !c.auth() {
		return
	}
	page, err := coinmquant.Templates(orm.NewOrm(), c.GetString("name"), c.num("page", 1), c.num("limit", 20))
	c.reply(400, page, err)
}
func (c *CoinMQuantController) GetTemplate() {
	if !c.auth() {
		return
	}
	id, err := c.id()
	if err != nil {
		c.reply(400, nil, err)
		return
	}
	row, err := coinmquant.Template(orm.NewOrm(), id)
	c.reply(400, row, err)
}
func (c *CoinMQuantController) CreateTemplate() {
	if !c.auth() {
		return
	}
	row, err := coinmquant.CreateTemplate(orm.NewOrm(), c.Ctx.Input.RequestBody)
	c.reply(400, row, err)
}
func (c *CoinMQuantController) EditTemplate() {
	if !c.auth() {
		return
	}
	id, err := c.id()
	if err != nil {
		c.reply(400, nil, err)
		return
	}
	row, err := coinmquant.EditTemplate(orm.NewOrm(), id, c.Ctx.Input.RequestBody)
	c.reply(400, row, err)
}
func (c *CoinMQuantController) DeleteTemplate() {
	if !c.auth() {
		return
	}
	id, err := c.id()
	if err != nil {
		c.reply(400, nil, err)
		return
	}
	c.reply(400, nil, coinmquant.DeleteTemplate(orm.NewOrm(), id))
}
func (c *CoinMQuantController) ImportTemplate() {
	if !c.auth() {
		return
	}
	row, action, err := coinmquant.ImportTemplate(orm.NewOrm(), c.Ctx.Input.RequestBody)
	c.reply(400, map[string]interface{}{"template": row, "action": action}, err)
}
func (c *CoinMQuantController) RuleTest() {
	if !c.auth() {
		return
	}
	var body struct {
		Strategy json.RawMessage `json:"strategy"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &body); err != nil {
		c.reply(400, nil, err)
		return
	}
	var strategy string
	if err := json.Unmarshal(body.Strategy, &strategy); err != nil {
		strategy = string(body.Strategy)
	}
	rules, err := coinmquant.Rules(strategy)
	if err != nil {
		c.reply(400, nil, err)
		return
	}
	c.reply(200, map[string]interface{}{"mode": "syntax_only", "pass": false, "rules": len(rules)}, nil)
}
