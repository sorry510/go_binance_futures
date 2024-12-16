package controllers

import (
	"strconv"

	"go_binance_futures/models"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/server/web"
)

type FuturesOrderController struct {
	web.Controller
}

type FuturesOrderTableList struct {
	models.FuturesOrder
	NowPrice string `orm:"column(now_price)" json:"now_price"`
}

func (ctrl *FuturesOrderController) Get() {
	paramsSymbol := ctrl.GetString("symbol")
	paramPositionSide := ctrl.GetString("position_side") // LONG, SHORT, all
	paramsPage := ctrl.GetString("page", "1")
	paramsLimit := ctrl.GetString("limit", "20")
	paramsStartTime := ctrl.GetString("start_time")
	paramsEndTime := ctrl.GetString("end_time")
	paramsSide := ctrl.GetString("type") // BUY, SELL, all
	paramsStatus := ctrl.GetString("status") // NEW, PARTIALLY_FILLED, FILLED, CANCELED, REJECTED, EXPIRED, all
	
	page, _ := strconv.Atoi(paramsPage)
	limit, _ := strconv.Atoi(paramsLimit)
	offset := (page - 1) * limit
	
	o := orm.NewOrm()
	var orders []OrderTableList
	
	var total int64
	sql := "SELECT t.*, s.close as now_price FROM `futures_orders` t LEFT JOIN symbols s ON t.symbol = s.symbol where 1 = 1"
	countSql := "SELECT COUNT(*) FROM `futures_orders` t LEFT JOIN symbols s ON t.symbol = s.symbol where 1 = 1"
	
	if (paramsSymbol != "") {
		sql += ` and t.symbol like '%` + paramsSymbol + `%'`
		countSql += ` and t.symbol like '%` + paramsSymbol + `%'`
	}
	if (paramPositionSide != "all" && paramPositionSide != "") {
		sql += ` and t.positionSide = '` + paramPositionSide + `'`
		countSql += ` and t.positionSide = '` + paramPositionSide + `'`
	}
	if (paramsStartTime != "") {
		sql += ` and t.updateTime >= '` + paramsStartTime + `'`
		countSql += ` and t.updateTime >= '` + paramsStartTime + `'`
	}
	if (paramsEndTime != "") {
		sql += ` and t.updateTime <= '` + paramsEndTime + `'`
		countSql += ` and t.updateTime <= '` + paramsEndTime + `'`
	}
	if (paramsSide != "all" && paramsSide != "") {
		sql += ` and t.side = '` + paramsSide + `'`
		countSql += ` and t.side = '` + paramsSide + `'`
	}
	if (paramsStatus != "all" && paramsStatus != "") {
		sql += ` and t.status = '` + paramsStatus + `'`
		countSql += ` and t.status = '` + paramsStatus + `'`
	}
	
	sql = sql + " ORDER BY t.updateTime DESC LIMIT " + strconv.Itoa(limit) + " OFFSET " + strconv.Itoa(offset)
    _, err := o.Raw(sql).QueryRows(&orders)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
	}
	err = o.Raw(countSql).QueryRow(&total)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
	}
	
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": map[string]interface{} {
			"total": total,
			"list": orders,
		},
		"msg": "success",
	})
}

// func (ctrl *FuturesOrderController) Delete() {
// 	id := ctrl.Ctx.Input.Param(":id")
// 	o := orm.NewOrm()
// 	_, err := o.Raw("DELETE FROM \"order\" where id = ?", id).Exec()
// 	if err != nil {
// 		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
// 		return
// 	}
// 	ctrl.Ctx.Resp(utils.ResJson(200, nil))
// }

// func (ctrl *FuturesOrderController) DeleteAll() {
	
// 	o := orm.NewOrm()
// 	_, err := o.Raw("DELETE FROM \"order\" where 1=1").Exec()
// 	if err != nil {
// 		// 处理错误
// 		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
// 		return
// 	}
// 	ctrl.Ctx.Resp(utils.ResJson(200, nil))
// }
