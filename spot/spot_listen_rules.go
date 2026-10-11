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
func spotKlineCustomListen(coin models.ListenSymbols) {
	if !spotListenNoticeDue(coin) || coin.Strategy == "" {
		return
	}
	var strategies technology.StrategyConfig
	if err := json.Unmarshal([]byte(coin.Strategy), &strategies); err != nil {
		logs.Error("invalid spot listen strategy:", coin.Symbol, err)
		return
	}

	env := line.InitSpotParseEnv(coin.Symbol, coin.Technology)
	price, ok := env["NowPrice"].(float64)
	if !ok || price <= 0 {
		logs.Error("spot listen has no valid spot ticker:", coin.Symbol)
		return
	}
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
			logs.Error("compile spot listen strategy:", coin.Symbol, err)
			continue
		}
		result, err := expr.Run(program, env)
		if err != nil {
			logs.Error("run spot listen strategy:", coin.Symbol, err)
			continue
		}
		matched, isBool := result.(bool)
		if !isBool || !matched {
			continue
		}
		if !saveSpotListenNotice(coin, direction) {
			return
		}
		pusher.SetModuleName("coin_listen").SpotListenKlineCustom(notify.SpotListenParams{
			Title:  lang.Lang("spot.listen_custom_title"),
			Symbol: coin.Symbol, PositionSide: strategy.Type,
			NowPrice: price, StrategyName: strategy.Name, Remarks: strategy.Code,
		})
		// One notification per configured interval, even if multiple rules match.
		return
	}
}
