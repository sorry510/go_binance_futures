package controllers

import (
	"strconv"
	"time"

	"go_binance_futures/models"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/server/web"
)

type OrderController struct {
	web.Controller
}

func (ctrl *OrderController) Get() {
	paramsSymbol := ctrl.GetString("symbol")
	paramsPage := ctrl.GetString("page", "1")
	paramsLimit := ctrl.GetString("limit", "20")
	paramsStartTime := ctrl.GetString("start_time")
	paramsEndTime := ctrl.GetString("end_time")
	
	page, _ := strconv.Atoi(paramsPage)
	limit, _ := strconv.Atoi(paramsLimit)
	offset := (page - 1) * limit
	
	o := orm.NewOrm()
	var orders []models.Order
	
	var query = o.QueryTable("order").
		OrderBy("-UpdateTime")
		
	if (paramsSymbol != "") {
		query = query.Filter("Symbol__icontains", paramsSymbol)
	}
	if (paramsStartTime != "") {
		query = query.Filter("UpdateTime__gte", paramsStartTime)
	}
	if (paramsEndTime != "") {
		query = query.Filter("UpdateTime__lte", paramsEndTime)
	}
	total, err1 := query.Count()
	if err1 != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "error_count"))
	}
	
	queryPagination := query.Limit(limit, offset)
	
	_, err2 := queryPagination.All(&orders)
	if err2 != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "error_data"))
	}
	
	var orderTables []models.OrderTable
	for _, order := range orders {
		orderTable := models.OrderTable {
			Order: order,
			UpdateDate: time.Unix(order.UpdateTime / 1000, 0).Format("2006-01-02 15:04:05"),
		}
		orderTable.SideText = "平仓"
		if (order.Side == "open") {
			orderTable.SideText = "开仓"
		}
		orderTable.PositionText = "做空"
		if (order.PositionSide == "LONG") {
			orderTable.PositionText = "做多"
		}
		orderTables = append(orderTables, orderTable)
	}

	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": map[string]interface{} {
			"total": total,
			"list": orderTables,
		},
		"msg": "success",
	})
}

func (ctrl *OrderController) DeleteAll() {
	
	o := orm.NewOrm()
	_, err := o.Raw("DELETE FROM \"order\" where 1=1").Exec()
	if err != nil {
		// 处理错误
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "删除错误"))
		return
	}
	ctrl.Ctx.Resp(utils.ResJson(200, nil))
}
