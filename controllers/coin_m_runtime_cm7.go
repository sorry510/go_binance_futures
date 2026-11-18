package controllers

import (
	"net/http"

	"github.com/beego/beego/v2/client/orm"
	"go_binance_futures/service/coinmexecution"
)

// GetRuntimeSummary exposes local, authenticated COIN-M state only.
// It never contacts signed DAPI or invokes the trade processor.
func (c *CoinMQuantController) GetRuntimeSummary() {
	if !c.auth() {
		return
	}
	data, err := coinmexecution.ReadRuntimeSummary(orm.NewOrm())
	if err != nil {
		c.reply(http.StatusServiceUnavailable, nil, err)
		return
	}
	c.reply(http.StatusOK, data, nil)
}
