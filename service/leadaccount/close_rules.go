package leadaccount

import (
	"context"
	"fmt"
	"strings"

	"github.com/adshao/go-binance/v2/futures"
	"go_binance_futures/service/futuresownership"
)

// ValidateLeadCloseRules checks symbol filters without changing an account's
// leverage or margin mode. Min notional is waived ONLY for an entire remaining
// position; a partial reduction cannot claim the full-close exception.
func ValidateLeadCloseRules(order futuresownership.OrderRequest, symbol futures.Symbol, referencePrice, fullCloseCap, fullLiveQty float64) error {
	invalid := fmt.Errorf("Lead close exchange filters are unverified")
	if symbol.Symbol != strings.ToUpper(strings.TrimSpace(order.Symbol)) || symbol.Status != "TRADING" || symbol.QuoteAsset != "USDT" ||
		!riskFinitePositive(order.Quantity) || !riskFinitePositive(referencePrice) || !riskFinitePositive(fullCloseCap) || !riskFinitePositive(fullLiveQty) ||
		order.Quantity > fullCloseCap+1e-10 {
		return invalid
	}

	marketExecution := order.OrderType == "MARKET" || order.OrderType == "STOP_MARKET" || order.OrderType == "TAKE_PROFIT_MARKET"
	if !marketExecution && order.OrderType != "LIMIT" {
		return invalid
	}
	switch order.OrderType {
	case "MARKET":
		if order.Price != 0 || order.StopPrice != 0 {
			return invalid
		}
	case "LIMIT":
		if !riskFinitePositive(order.Price) || order.StopPrice != 0 {
			return invalid
		}
	case "STOP_MARKET", "TAKE_PROFIT_MARKET":
		if order.Price != 0 || !riskFinitePositive(order.StopPrice) {
			return invalid
		}
	}
	lotFound, priceFound, notionalFound := false, false, false
	minNotional := 0.0
	for _, filter := range symbol.Filters {
		kind, _ := filter["filterType"].(string)
		switch kind {
		case "LOT_SIZE", "MARKET_LOT_SIZE":
			if (kind == "MARKET_LOT_SIZE") != marketExecution {
				continue
			}
			minimum, e1 := parseRuleFloat(filter, "minQty")
			maximum, e2 := parseRuleFloat(filter, "maxQty")
			step, e3 := parseRuleFloat(filter, "stepSize")
			if e1 != nil || e2 != nil || e3 != nil || !riskFinitePositive(step) || maximum < minimum ||
				order.Quantity < minimum || order.Quantity > maximum || !alignedToStep(order.Quantity, minimum, step) {
				return fmt.Errorf("Lead close quantity filter rejected")
			}
			lotFound = true
		case "PRICE_FILTER":
			minimum, e1 := parseRuleFloat(filter, "minPrice")
			maximum, e2 := parseRuleFloat(filter, "maxPrice")
			tick, e3 := parseRuleFloat(filter, "tickSize")
			if e1 != nil || e2 != nil || e3 != nil || !riskFinitePositive(tick) || maximum < minimum {
				return invalid
			}
			requestedPrice := 0.0
			if order.OrderType == "LIMIT" {
				requestedPrice = order.Price
			}
			if order.OrderType == "STOP_MARKET" || order.OrderType == "TAKE_PROFIT_MARKET" {
				requestedPrice = order.StopPrice
			}
			if requestedPrice > 0 && (requestedPrice < minimum || requestedPrice > maximum || !alignedToStep(requestedPrice, minimum, tick)) {
				return fmt.Errorf("Lead close price tick/limits rejected")
			}
			priceFound = true
		case "MIN_NOTIONAL":
			minimum, err := parseRuleFloat(filter, "notional")
			if err != nil {
				return invalid
			}
			if minimum > minNotional {
				minNotional = minimum
			}
			notionalFound = true
		case "NOTIONAL":
			minimum, err := parseRuleFloat(filter, "minNotional")
			if err != nil {
				return invalid
			}
			if minimum > minNotional {
				minNotional = minimum
			}
			notionalFound = true
		}
	}
	if !lotFound || !priceFound || !notionalFound {
		return invalid
	}
	effectivePrice := referencePrice
	if order.OrderType == "LIMIT" {
		effectivePrice = order.Price
	}
	if order.OrderType == "STOP_MARKET" || order.OrderType == "TAKE_PROFIT_MARKET" {
		effectivePrice = order.StopPrice
	}
	if order.Quantity*effectivePrice < minNotional && order.Quantity < fullLiveQty-1e-10 {
		return fmt.Errorf("Lead partial close below min notional")
	}
	return nil
}

func (a *LeadExecutionAdapter) verifyCloseRules(ctx context.Context, order futuresownership.OrderRequest, referencePrice, cap, liveQty float64) error {
	if a.rules == nil || a.rules.AccountID != "lead" || a.rules.Reader == nil {
		return ErrLeadOpenBlocked
	}
	exchange, err := a.rules.Reader.GetLeadExchangeInfoContext(ctx)
	if err != nil || exchange == nil {
		return fmt.Errorf("Lead close exchange rules unavailable")
	}
	for _, symbol := range exchange.Symbols {
		if symbol.Symbol == strings.ToUpper(strings.TrimSpace(order.Symbol)) {
			return ValidateLeadCloseRules(order, symbol, referencePrice, cap, liveQty)
		}
	}
	return fmt.Errorf("Lead close symbol missing from exchangeInfo")
}
