package controllers

import (
	"github.com/beego/beego/v2/client/orm"
	"go_binance_futures/service/coinmquant"
)

// CM-3 local read model only. No Binance order/cancel/reconcile writes.
func (c *CoinMQuantController) ListPositions() {
	if !c.auth() {
		return
	}
	page, err := coinmquant.Positions(orm.NewOrm(), coinmquant.LedgerFilter{
		Symbol: c.GetString("symbol"), Status: c.GetString("status"), Page: c.num("page", 1), Limit: c.num("limit", 20),
	})
	c.reply(400, page, err)
}
func (c *CoinMQuantController) ListOrders() {
	if !c.auth() {
		return
	}
	page, err := coinmquant.Orders(orm.NewOrm(), coinmquant.LedgerFilter{
		Symbol: c.GetString("symbol"), Status: c.GetString("status"), Page: c.num("page", 1), Limit: c.num("limit", 20),
	})
	c.reply(400, page, err)
}
func (c *CoinMQuantController) GetPositionDetail() {
	if !c.auth() {
		return
	}
	id, err := c.id()
	if err != nil {
		c.reply(400, nil, err)
		return
	}
	detail, err := coinmquant.Detail(orm.NewOrm(), id)
	c.reply(400, detail, err)
}
