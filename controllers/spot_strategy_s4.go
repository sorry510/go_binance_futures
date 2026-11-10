package controllers

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"go_binance_futures/service/spotstrategy"
)

func (c *SpotStrategyController) s4ReadDB() (*sql.DB, error) {
	if c.readDB != nil {
		return c.readDB, nil
	}
	return orm.GetDB("default")
}

func (c *SpotStrategyController) s4ReadScope() bool {
	if _, ok := c.authorized(); !ok {
		return false
	}
	// The only supported Spot ownership scope is fixed server-side.
	// An old/explicit caller may repeat it, but must never select another
	// scope; ordinary read-only pages need not send redundant account fields.
	account, owner := c.GetString("account_id"), c.GetString("owner")
	if account != "" || owner != "" {
		if err := spotstrategy.ValidateScope(account, owner); err != nil {
			c.respond(http.StatusBadRequest, nil, "spot_strategy_scope_invalid")
			return false
		}
	}
	return true
}
func (c *SpotStrategyController) s4ReadPage() (spotstrategy.PageRequest, bool) {
	page := spotstrategy.PageRequest{Symbol: c.GetString("symbol"), Status: c.GetString("status")}
	if raw := c.GetString("page"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 {
			c.respond(http.StatusBadRequest, nil, "spot_strategy_page_invalid")
			return page, false
		}
		page.Page = v
	}
	if raw := c.GetString("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 {
			c.respond(http.StatusBadRequest, nil, "spot_strategy_limit_invalid")
			return page, false
		}
		page.Limit = v
	}
	return page, true
}

func (c *SpotStrategyController) s4HandleError(err error) {
	if err == nil {
		return
	}
	switch {
	case errors.Is(err, spotstrategy.ErrInvalidS4Filter):
		c.respond(http.StatusBadRequest, nil, "spot_strategy_filter_invalid")
		return
	case errors.Is(err, spotstrategy.ErrUnmanagedOrder):
		c.respond(http.StatusNotFound, nil, "spot_strategy_resource_not_found")
	default:
		// The service can contain raw DB or state messages. Never reflect those,
		// SQL fragments or signed REST URLs to callers.
		c.respond(http.StatusServiceUnavailable, nil, "spot_strategy_read_unavailable")
	}
}
func (c *SpotStrategyController) ListSymbols() {
	if _, ok := c.authorized(); !ok {
		return
	}
	page, ok := c.s4ReadPage()
	if !ok {
		return
	}
	page.QuoteAsset = c.GetString("quote_asset")
	db, err := c.s4ReadDB()
	if err != nil {
		c.s4HandleError(err)
		return
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 6*time.Second)
	defer cancel()
	result, err := spotstrategy.ListSettings(ctx, db, page)
	if err != nil {
		c.s4HandleError(err)
		return
	}
	c.respond(http.StatusOK, result, "success")
}
func (c *SpotStrategyController) ListHoldings() {
	if !c.s4ReadScope() {
		return
	}
	page, ok := c.s4ReadPage()
	if !ok {
		return
	}
	db, err := c.s4ReadDB()
	if err != nil {
		c.s4HandleError(err)
		return
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 6*time.Second)
	defer cancel()
	result, err := spotstrategy.ListHoldings(ctx, db, page)
	if err != nil {
		c.s4HandleError(err)
		return
	}
	c.respond(http.StatusOK, result, "success")
}
func (c *SpotStrategyController) ListOrders() {
	if !c.s4ReadScope() {
		return
	}
	page, ok := c.s4ReadPage()
	if !ok {
		return
	}
	holdingID := int64(0)
	if raw := c.GetString("holding_id"); raw != "" {
		v, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil || v <= 0 {
			c.respond(http.StatusBadRequest, nil, "spot_strategy_holding_id_invalid")
			return
		}
		holdingID = v
	}
	db, err := c.s4ReadDB()
	if err != nil {
		c.s4HandleError(err)
		return
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 6*time.Second)
	defer cancel()
	result, err := spotstrategy.ListOrders(ctx, db, holdingID, page)
	if err != nil {
		c.s4HandleError(err)
		return
	}
	c.respond(http.StatusOK, result, "success")
}
func (c *SpotStrategyController) ListFills() {
	if !c.s4ReadScope() {
		return
	}
	page, ok := c.s4ReadPage()
	if !ok {
		return
	}
	raw := c.GetString("order_id")
	localID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || localID <= 0 {
		c.respond(http.StatusBadRequest, nil, "spot_strategy_order_id_invalid")
		return
	}
	db, err := c.s4ReadDB()
	if err != nil {
		c.s4HandleError(err)
		return
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 6*time.Second)
	defer cancel()
	result, err := spotstrategy.ListOrderFills(ctx, db, localID, page)
	if err != nil {
		c.s4HandleError(err)
		return
	}
	c.respond(http.StatusOK, result, "success")
}
