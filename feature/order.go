package feature

import (
	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

// 检查开仓并且没有结束的订单，结束是指没有找到对应平仓的订单
func UpdateOrderStatus() {
	o := orm.NewOrm()
	var openOrders []models.Order
	sql := "SELECT * FROM `order` where side = 'open' and closedTime = 0 limit 200"
	_, err := o.Raw(sql).QueryRows(&openOrders)
	if err != nil {
		logs.Error("Error fetching orders:", err)
	}
	
	positions, _ := GetTransformPositions()
	
	for _, openOrder := range openOrders {
		var closeOrder models.Order
		o.QueryTable("order").
			Filter("Side", "close").
			Filter("Symbol", openOrder.Symbol).
			Filter("Amount", openOrder.Amount).
			Filter("PositionSide", openOrder.PositionSide).
			Filter("OrderId__gt", openOrder.OrderId). // 查询大于开仓订单的平仓订单
			OrderBy("Id").
			One(&closeOrder)
		if closeOrder.OrderId != 0 {
			// 找到对应的平仓订单，进行处理
			openOrder.Inexact_profit = closeOrder.Inexact_profit
			openOrder.ClosedTime = closeOrder.UpdateTime
			o.Update(&openOrder)
			logs.Info("Updated order status for open order, symbol:", openOrder.Symbol)
		} else {
			// 检查当前持仓情况，是否已经没有持仓，就删掉对应的开仓订单
			find := false
			for _, position := range positions {
				if position.Symbol == openOrder.Symbol {
					// 仍然有持仓，跳过(不检查 做多还是做空)
					find = true
					break
				}
			}
			if !find {
				// 没有持仓，删除对应的开仓订单
				o.Delete(&openOrder)
				logs.Info("Deleted open order with no corresponding position, symbol:", openOrder.Symbol)
			}
		}
	}
}