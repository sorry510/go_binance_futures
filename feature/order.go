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
	sql := "SELECT * FROM `order` where side = 'open' and closedTime = 0 limit 1000"
	_, err := o.Raw(sql).QueryRows(&openOrders)
	if err != nil {
		logs.Error("Error fetching orders:", err)
	}
	
	for _, openOrder := range openOrders {
		var closeOrder models.Order
		err := o.QueryTable("order").
			Filter("Side", "close").
			Filter("Symbol", openOrder.Symbol).
			Filter("Amount", openOrder.Amount).
			One(&closeOrder)
		if err != nil {
			continue
		}
		if closeOrder.OrderId != 0 {
			// 找到对应的平仓订单，进行处理
			openOrder.Inexact_profit = closeOrder.Inexact_profit
			openOrder.ClosedTime = closeOrder.UpdateTime
			o.Update(&openOrder)
			logs.Info("Updated order status for open order, symbol:", openOrder.Symbol)
		}
	}
}