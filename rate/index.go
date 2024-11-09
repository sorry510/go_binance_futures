package rate

import (
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	"strconv"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

// 监听套利情况
func ListenRateEat() {
	o := orm.NewOrm()
	var symbolList []models.EatRateSymbols
	query := o.QueryTable("eat_rate_symbols").Filter("enable", 1)
	_, err := query.OrderBy("ID").All(&symbolList)
	if err != nil {
		logs.Error("listRateEat error:", err.Error())
		return
	}
	
	for _, symbols := range symbolList {
		futuresSymbol := symbols.FuturesSymbol
		oldProfit := symbols.Profit
		lastProfitTime := symbols.LastProfitTime
		incomes, err := binance.GetIncome(binance.IncomeParams{
			Symbol: futuresSymbol,
			IncomeType: "FUNDING_FEE", // 资金费率
			StartTime: symbols.LastProfitTime,
		})
		
		if err != nil {
			logs.Error("GetIncome error, symbol:", futuresSymbol)
			logs.Error("err:", err.Error())
			continue
		}
		for _, income := range incomes {
			feeFloat, _ := strconv.ParseFloat(income.Income, 64)
			oldProfit += feeFloat
			lastProfitTime = income.Time
		}
		symbols.Profit = oldProfit
		symbols.LastProfitTime = lastProfitTime + 1000 * 60 * 60 // 往后加1小时
		o.Update(&symbols) // _ 是受影响的条数
	}
}