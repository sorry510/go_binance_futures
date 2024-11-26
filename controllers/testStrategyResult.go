package controllers

import (
	"strconv"

	"go_binance_futures/models"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/server/web"
)

type TestStrategyResultController struct {
	web.Controller
}

type TestStrategyResultsTableList struct {
	models.TestStrategyResults
	NowPrice string `orm:"column(now_price)" json:"now_price"`
}

func (ctrl *TestStrategyResultController) Get() {
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
	
	var results []TestStrategyResultsTableList
	var total int64
	sql := `SELECT t.*, s.close as now_price FROM test_strategy_results t LEFT JOIN symbols s ON t.symbol = s.symbol where 1 = 1`
	countSql := `SELECT COUNT(*) FROM test_strategy_results t LEFT JOIN symbols s ON t.symbol = s.symbol where 1 = 1`
	
	if (paramsSymbol != "") {
		sql += ` and t.symbol like '%` + paramsSymbol + `%'`
		countSql += ` and t.symbol like '%` + paramsSymbol + `%'`
	}
	if (paramPositionSide != "all" && paramPositionSide != "") {
		sql += ` and t.position_side = '` + paramPositionSide + `'`
		countSql += ` and t.position_side = '` + paramPositionSide + `'`
	}
	if (paramsStartTime != "") {
		sql += ` and t.createTime >= '` + paramsStartTime + `'`
		countSql += ` and t.createTime >= '` + paramsStartTime + `'`
	}
	if (paramsEndTime != "") {
		sql += ` and t.createTime <= '` + paramsEndTime + `'`
		countSql += ` and t.createTime <= '` + paramsEndTime + `'`
	}
	if (paramsType == "open") {
		sql += ` and t.close_price = '0'`
		countSql += ` and t.close_price = '0'`
	} else if (paramsType == "close") {
		sql += ` and t.close_price != '0'`
		countSql += ` and t.close_price != '0'`
	}
	
	sql = sql + " ORDER BY t.createTime DESC LIMIT " + strconv.Itoa(limit) + " OFFSET " + strconv.Itoa(offset)
    _, err := o.Raw(sql).QueryRows(&results)
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
			"list": results,
		},
		"msg": "success",
	})
}

func (ctrl *TestStrategyResultController) Delete() {
	id := ctrl.Ctx.Input.Param(":id")
	o := orm.NewOrm()
	_, err := o.Raw("DELETE FROM \"test_strategy_results\" where id = ?", id).Exec()
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(utils.ResJson(200, nil))
}

func (ctrl *TestStrategyResultController) DeleteAll() {
	o := orm.NewOrm()
	_, err := o.Raw("DELETE FROM \"test_strategy_results\" where 1=1").Exec()
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(utils.ResJson(200, nil))
}
