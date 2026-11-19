package controllers

import (
	"net/http"
	"strings"

	"go_binance_futures/feature"

	"github.com/beego/beego/v2/server/web"
	"github.com/golang-jwt/jwt/v5"
)

type MainEntryDiagnosticsController struct{ web.Controller }

// GET only. This returns a bounded in-process snapshot of the Main runner's
// previous decisions. It must not access DB, Binance or strategy evaluation.
func (c *MainEntryDiagnosticsController) Get() {
	claims, ok := c.Ctx.Input.GetData("user").(jwt.MapClaims)
	if !ok {
		c.reply(http.StatusUnauthorized, "entry_diagnostics_auth_required", nil)
		return
	}
	name, ok := claims["Username"].(string)
	if !ok || strings.TrimSpace(name) == "" {
		c.reply(http.StatusUnauthorized, "entry_diagnostics_auth_required", nil)
		return
	}
	symbol := strings.ToUpper(strings.TrimSpace(c.GetString("symbol")))
	side := strings.ToUpper(strings.TrimSpace(c.GetString("position_side")))
	if !validEntryDiagnosticSymbol(symbol) || (side != "" && side != "LONG" && side != "SHORT") {
		c.reply(http.StatusBadRequest, "invalid entry diagnostics filters", nil)
		return
	}
	c.reply(http.StatusOK, "success", feature.MainEntryDiagnostics(symbol, side))
}
func validEntryDiagnosticSymbol(symbol string) bool {
	if len(symbol) > 32 {
		return false
	}
	for _, r := range symbol {
		if (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}
func (c *MainEntryDiagnosticsController) reply(status int, message string, data any) {
	c.Ctx.Output.SetStatus(status)
	_ = c.JSONResp(map[string]any{"code": status, "msg": message, "data": data})
}
