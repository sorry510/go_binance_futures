package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/service/coinmquant"
)

// CM-4 still exposes only read-only local ledgers and separately requested
// DAPI balance/mark reads. Nothing here calls an order/claim/reconcile write.
func (c *CoinMQuantController) ListPositions() {
	if !c.auth() {
		return
	}
	o := orm.NewOrm()
	filter := coinmquant.LedgerFilter{
		Symbol: c.GetString("symbol"), Status: c.GetString("status"),
		Page: c.num("page", 1), Limit: c.num("limit", 20),
	}
	need, err := coinmquant.NeedMarks(o, filter)
	if err != nil {
		c.reply(400, nil, err)
		return
	}
	marks := map[string]coinmquant.Mark{}
	priceStatus := "not_required"
	if need {
		priceStatus = "unavailable"
		ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 8*time.Second)
		prices, readErr := binance.GetCoinMMarkPricesContext(ctx)
		cancel()
		if readErr == nil {
			now := time.Now().UnixMilli()
			for _, mark := range prices {
				if !binance.CoinMMarkTimeValid(mark.Time, now) {
					continue
				}
				value := coinmquant.Mark{Symbol: mark.Symbol, Price: mark.MarkPrice, Time: mark.Time}
				if value.Check(mark.Symbol) == nil {
					marks[mark.Symbol] = value
				}
			}
			if len(marks) > 0 {
				priceStatus = "available"
			}
		}
	}
	result, err := coinmquant.PositionSummaries(o, filter, marks)
	if err != nil {
		c.reply(400, nil, err)
		return
	}
	// A valid mark for another symbol must not mask an unpriced holding
	// within this filtered result.
	if need && priceStatus == "available" {
		for _, asset := range result.Totals {
			if asset.Unpriced > 0 {
				priceStatus = "unavailable"
				break
			}
		}
	}
	result.PriceStatus = priceStatus
	c.reply(200, result, nil)
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

type coinMWalletBalance struct {
	Asset              string `json:"asset"`
	WalletBalance      string `json:"wallet_balance"`
	AvailableBalance   string `json:"available_balance"`
	CrossWalletBalance string `json:"cross_wallet_balance"`
	CrossUnrealizedPnL string `json:"cross_unrealized_pnl"`
	WithdrawAvailable  string `json:"withdraw_available"`
	UpdatedAt          int64  `json:"updated_at"`
}

func (c *CoinMQuantController) GetWallet() {
	if !c.auth() {
		return
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 8*time.Second)
	defer cancel()
	balances, err := binance.GetDeliveryBalanceContext(ctx)
	if err != nil {
		// No silent mainnet fallback, no fabricated zero balances. In
		// particular, an unconfigured Demo key is NOT a blank wallet.
		c.reply(http.StatusServiceUnavailable, nil, err)
		return
	}
	wallet := make([]coinMWalletBalance, 0, len(balances))
	for _, b := range balances {
		if b == nil || b.Asset == "" {
			continue
		}
		wallet = append(wallet, coinMWalletBalance{
			Asset: b.Asset, WalletBalance: b.Balance, AvailableBalance: b.AvailableBalance,
			CrossWalletBalance: b.CrossWalletBalance, CrossUnrealizedPnL: b.CrossUnPnl,
			WithdrawAvailable: b.WithdrawAvailable, UpdatedAt: b.UpdateTime,
		})
	}
	c.reply(200, map[string]interface{}{
		"environment": binance.CoinMEnvironment(),
		"balances":    wallet,
		"read_only":   true,
	}, nil)
}
