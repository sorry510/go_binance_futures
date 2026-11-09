package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"go_binance_futures/service/orderhistory"

	"github.com/beego/beego/v2/server/web"
	"github.com/golang-jwt/jwt/v5"
)

// OrderController exposes ONLY read-only v23 Managed/Fill order history.
// The physical legacy `order` table is deliberately never read or deleted.
type OrderController struct{ web.Controller }

func (ctrl *OrderController) respondError(status int, message string) {
	ctrl.Ctx.Output.SetStatus(status)
	// Always respond JSON, regardless of Accept. Context.Resp negotiates
	// YAML/XML/Protobuf and can panic on a non-proto map payload.
	_ = ctrl.JSONResp(map[string]any{"code": status, "data": nil, "msg": message})
}
func (ctrl *OrderController) success(value any) {
	_ = ctrl.JSONResp(map[string]any{"code": 200, "data": value, "msg": "success"})
}

// The product has a single configured operator login, not independent
// main/lead per-user roles. Both accounts are readable only to that operator.
// Defense-in-depth: require validated JWT claims set by JwtMiddleware even if
// a future router accidentally omits its filter.
func (ctrl *OrderController) authorized() bool {
	claims, ok := ctrl.Ctx.Input.GetData("user").(jwt.MapClaims)
	if !ok {
		ctrl.respondError(http.StatusUnauthorized, "order_history_auth_required")
		return false
	}
	username, ok := claims["Username"].(string)
	if !ok || strings.TrimSpace(username) == "" {
		ctrl.respondError(http.StatusUnauthorized, "order_history_auth_required")
		return false
	}
	return true
}

func parseHistoryInt(s string, def int) (int, error) {
	if strings.TrimSpace(s) == "" {
		return def, nil
	}
	return strconv.Atoi(strings.TrimSpace(s))
}
func parseHistoryMillis(s string) (int64, error) {
	if strings.TrimSpace(s) == "" {
		return 0, nil
	}
	return strconv.ParseInt(strings.TrimSpace(s), 10, 64)
}
func (ctrl *OrderController) Get() {
	if !ctrl.authorized() {
		return
	}
	page, err := parseHistoryInt(ctrl.GetString("page"), 1)
	if err != nil {
		ctrl.respondError(400, "invalid page")
		return
	}
	limit, err := parseHistoryInt(ctrl.GetString("limit"), 20)
	if err != nil {
		ctrl.respondError(400, "invalid limit")
		return
	}
	start, err := parseHistoryMillis(ctrl.GetString("start_time"))
	if err != nil {
		ctrl.respondError(400, "invalid start_time")
		return
	}
	end, err := parseHistoryMillis(ctrl.GetString("end_time"))
	if err != nil {
		ctrl.respondError(400, "invalid end_time")
		return
	}
	filters := orderhistory.Filter{
		AccountID: ctrl.GetString("account_id"), Owner: ctrl.GetString("owner"),
		Symbol: ctrl.GetString("symbol"), PositionSide: ctrl.GetString("position_side"),
		Status: ctrl.GetString("status"), Settlement: ctrl.GetString("settlement"),
		StartTime: start, EndTime: end, Page: page, Limit: limit,
	}
	result, err := (orderhistory.Store{}).List(ctrl.Ctx.Request.Context(), filters)
	if err != nil {
		if errors.Is(err, orderhistory.ErrInvalidQuery) {
			ctrl.respondError(400, err.Error())
			return
		}
		ctrl.respondError(500, "managed order history unavailable")
		return
	}
	ctrl.success(result)
}
func (ctrl *OrderController) Details() {
	if !ctrl.authorized() {
		return
	}
	raw := ctrl.Ctx.Input.Param(":position_id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		ctrl.respondError(400, "invalid position_id")
		return
	}
	result, err := (orderhistory.Store{}).Details(ctrl.Ctx.Request.Context(), ctrl.GetString("account_id"), id)
	if err != nil {
		switch {
		case errors.Is(err, orderhistory.ErrInvalidQuery):
			ctrl.respondError(400, err.Error())
		case errors.Is(err, orderhistory.ErrNotFound):
			ctrl.respondError(404, "position cycle not found for this account")
		default:
			ctrl.respondError(500, "managed order details unavailable")
		}
		return
	}
	ctrl.success(result)
}

// Historical Fill records are immutable. No DELETE endpoint exists for them.
func (ctrl *OrderController) Delete() {
	ctrl.respondError(http.StatusGone, "historical exchange fills cannot be deleted")
}
func (ctrl *OrderController) DeleteAll() {
	ctrl.respondError(http.StatusGone, "historical exchange fills cannot be deleted")
}
