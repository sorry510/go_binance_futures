package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"go_binance_futures/service/spotledger"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

// SpotQuantAccountController is strictly independent from Futures, old Spot
// Rush and Binance account balances. Only local Spot-managed Fill data is
// exposed. No live order submission/cancel endpoints in 06-5.
type SpotQuantAccountController struct{ SpotQuantController }

func spotQuantRead() (*spotledger.SpotReadService, error) {
	return spotledger.NewSpotReadService(orm.NewOrm(), "mysql")
}
func (c *SpotQuantAccountController) respondRead(value interface{}, err error) {
	if err == nil {
		c.respond(http.StatusOK, value, nil)
		return
	}
	switch {
	case errors.Is(err, spotledger.ErrSummaryTooLarge):
		c.respond(http.StatusUnprocessableEntity, nil, err)
	case errors.Is(err, spotledger.ErrNeedsReview):
		c.respond(http.StatusConflict, nil, fmt.Errorf("spot ledger needs reconciliation"))
	case errors.Is(err, orm.ErrNoRows):
		c.respond(http.StatusNotFound, nil, fmt.Errorf("spot record not found"))
	case strings.HasPrefix(err.Error(), "invalid ") || strings.Contains(err.Error(), "confirm=true"):
		c.respond(http.StatusBadRequest, nil, err)
	default:
		logs.Error("spot quant read/rebuild failure: %v", err)
		c.respond(http.StatusInternalServerError, nil, fmt.Errorf("spot ledger unavailable"))
	}
}
func (c *SpotQuantAccountController) positionID() (int64, error) {
	id, err := strconv.ParseInt(c.Ctx.Input.Param(":id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid Spot position id")
	}
	return id, nil
}
func (c *SpotQuantAccountController) Positions() {
	if !c.authorized() {
		return
	}
	read, err := spotQuantRead()
	if err != nil {
		c.respondRead(nil, err)
		return
	}
	value, err := read.Positions(spotledger.PositionFilter{
		Symbol: c.GetString("symbol"), SymbolLike: true, Status: c.GetString("status"),
		Page: c.intParam("page", 1), Limit: c.intParam("limit", 20),
	})
	c.respondRead(value, err)
}
func (c *SpotQuantAccountController) PositionDetail() {
	if !c.authorized() {
		return
	}
	id, err := c.positionID()
	if err != nil {
		c.respondRead(nil, err)
		return
	}
	read, err := spotQuantRead()
	if err != nil {
		c.respondRead(nil, err)
		return
	}
	view, err := read.Position(id)
	if err != nil {
		c.respondRead(nil, err)
		return
	}
	orders, err := read.Ledger().ListOrders(id)
	if err != nil {
		c.respondRead(nil, err)
		return
	}
	fills, err := read.Ledger().ListFills(id)
	if err != nil {
		c.respondRead(nil, err)
		return
	}
	issues, err := read.Ledger().ReviewPosition(id)
	if err != nil {
		if !errors.Is(err, spotledger.ErrNeedsReview) {
			c.respondRead(nil, err)
			return
		}
		// Show immutable orders/fills for operator diagnosis instead of
		// hiding the details behind a review-required projection error.
		issues = []string{err.Error()}
	}
	c.respondRead(map[string]interface{}{
		"view": view, "orders": orders, "fills": fills, "review_issues": issues,
		"manual_trading_enabled": false, "binance_wallet_owned": false,
	}, nil)
}
func (c *SpotQuantAccountController) Orders() {
	if !c.authorized() {
		return
	}
	read, err := spotQuantRead()
	if err != nil {
		c.respondRead(nil, err)
		return
	}
	positionID := int64(0)
	if value := c.GetString("position_id"); value != "" {
		positionID, err = strconv.ParseInt(value, 10, 64)
		if err != nil || positionID <= 0 {
			c.respondRead(nil, fmt.Errorf("invalid position_id"))
			return
		}
	}
	value, err := read.Orders(spotledger.OrderFilter{
		PositionID: positionID, Symbol: c.GetString("symbol"),
		Status: c.GetString("status"), Page: c.intParam("page", 1), Limit: c.intParam("limit", 20),
	})
	c.respondRead(value, err)
}

// LocalRebuild uses only already-persisted immutable Spot fills. It does
// NOT edit fill amounts, claim wallet assets or call Binance APIs.
// The UI requires a second confirmation before invoking it.
func (c *SpotQuantAccountController) LocalRebuild() {
	if !c.authorized() {
		return
	}
	var body struct {
		Confirm bool `json:"confirm"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &body); err != nil || !body.Confirm {
		c.respondRead(nil, fmt.Errorf("confirm=true is required for local ledger rebuild"))
		return
	}
	id, err := c.positionID()
	if err != nil {
		c.respondRead(nil, err)
		return
	}
	read, err := spotQuantRead()
	if err != nil {
		c.respondRead(nil, err)
		return
	}
	issues, err := read.Ledger().ReviewPosition(id)
	if err != nil {
		c.respondRead(nil, err)
		return
	}
	if len(issues) > 0 {
		c.respondRead(nil, fmt.Errorf("%w: order exchange state requires reconciliation before local rebuild: %v", spotledger.ErrNeedsReview, issues))
		return
	}
	row, err := read.Ledger().RebuildPosition(id)
	c.respondRead(row, err)
}
