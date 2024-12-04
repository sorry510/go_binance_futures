package controllers

import (
	"strconv"

	"go_binance_futures/models"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/server/web"
)

type OrderController struct {
	web.Controller
}

type OrderTableList struct {
	models.Order
	NowPrice string `orm:"column(now_price)" json:"now_price"`
}

func (ctrl *OrderController) Get() {
	paramsSymbol := ctrl.GetString("symbol")
	paramPositionSide := ctrl.GetString("position_side") // LONG, SHORT, all
	paramsPage := ctrl.GetString("page", "1")
	paramsLimit := ctrl.GetString("limit", "20")
	paramsStartTime := ctrl.GetString("start_time")
	paramsEndTime := ctrl.GetString("end_time")
	paramsType := ctrl.GetString("type") // open, close, all
	
	page, _ := strconv.Atoi(paramsPage)
	limit, _ := strconv.Atoi(paramsLimit)
	offset := (page - 1) * limit
	
	o := orm.NewOrm()
	var orders []OrderTableList
	
	var total int64
	sql := "SELECT t.*, s.close as now_price FROM `order` t LEFT JOIN symbols s ON t.symbol = s.symbol where 1 = 1"
	countSql := "SELECT COUNT(*) FROM `order` t LEFT JOIN symbols s ON t.symbol = s.symbol where 1 = 1"
	
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
	if (paramsType == "open") {
		sql += ` and t.side = '` + paramsType + `'`
		countSql += ` and t.side = '` + paramsType + `'`
	} else if (paramsType == "close") {
		sql += ` and t.side = '` + paramsType + `'`
		countSql += ` and t.side = '` + paramsType + `'`
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
