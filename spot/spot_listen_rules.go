package spot

import (
	"encoding/json"
	"time"

	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/lang"
	"go_binance_futures/models"
	"go_binance_futures/notify"
	"go_binance_futures/technology"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	"github.com/expr-lang/expr"
)

func spotListenNoticeDue(coin models.ListenSymbols) bool {
	return time.Now().UnixMilli()-coin.LastNoticeTime >= coin.NoticeLimitMin*60*1000
}

func saveSpotListenNotice(coin models.ListenSymbols, direction string) bool {
	coin.LastNoticeTime = time.Now().UnixMilli()
	coin.LastNoticeType = direction
	if _, err := orm.NewOrm().Update(&coin, "LastNoticeTime", "LastNoticeType"); err != nil {
		logs.Error("update spot listen notice failed:", coin.Symbol, err)
		return false
	}
	return true
}

// spotKlineCustomListen evaluates spot indicators and spot market prices only.
// A true rule triggers a notification; it does not submit an order.
// spotKlineCustomListen reports why an enabled custom listener did or did not
// dispatch an alert. The caller rate-limits routine outcome logs.
func spotKlineCustomListen(coin models.ListenSymbols) spotListenOutcome {
	if !spotListenNoticeDue(coin) {
		return spotListenCooldown
	}
	if coin.Strategy == "" {
		return spotListenNoStrategy
	}
	var strategies technology.StrategyConfig
	if err := json.Unmarshal([]byte(coin.Strategy), &strategies); err != nil {
		logSpotListenProblem(coin, spotListenInvalidStrategy, "", err)
		return spotListenInvalidStrategy
	}
	// Do not request spot klines/indicator data when there is nothing enabled
	// to execute; invalid configurations are reported explicitly.
	active := 0
	for _, strategy := range strategies {
		if strategy.Enable && strategy.Code != "" && (strategy.Type == "long" || strategy.Type == "short") {
			active++
		}
	}
	if active == 0 {
		return spotListenNoActiveRules
	}

	env := line.InitSpotParseEnv(coin.Symbol, coin.Technology)
	price, ok := env["NowPrice"].(float64)
	if !ok || price <= 0 {
		logSpotListenProblem(coin, spotListenMissingPrice, "", nil)
		return spotListenMissingPrice
	}

	hadRuleError := false
	for _, strategy := range strategies {
		if !strategy.Enable || strategy.Code == "" {
			continue
		}
		direction := ""
		switch strategy.Type {
		case "long":
			direction = "up"
		case "short":
			direction = "down"
		default:
			continue
		}
		program, err := expr.Compile(strategy.Code, expr.Env(env))
		if err != nil {
			logSpotListenProblem(coin, spotListenRuleError, strategy.Name, err)
			hadRuleError = true
			continue
		}
		result, err := expr.Run(program, env)
		if err != nil {
			logSpotListenProblem(coin, spotListenRuleError, strategy.Name, err)
			hadRuleError = true
			continue
		}
		matched, isBool := result.(bool)
		if !isBool {
			logSpotListenProblem(coin, spotListenRuleError, strategy.Name, nil)
			hadRuleError = true
			continue
		}
		if !matched {
			continue
		}
		if !saveSpotListenNotice(coin, direction) {
			return spotListenNoticeUpdateError
		}
		pusher.SetModuleName("coin_listen").SpotListenKlineCustom(notify.SpotListenParams{
			Title:  lang.Lang("spot.listen_custom_title"),
			Symbol: coin.Symbol, PositionSide: strategy.Type,
			NowPrice: price, StrategyName: strategy.Name, Remarks: strategy.Code,
		})
		logs.Info("spot listen notification dispatched: symbol=%s strategy=%q side=%s price=%g", coin.Symbol, strategy.Name, strategy.Type, price)
		// One notification per configured interval, even if multiple rules match.
		return spotListenNotified
	}
	if hadRuleError {
		return spotListenRuleError
	}
	return spotListenNoMatch
}
