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

func (ctrl *TestStrategyResultController) Get() {
	paramsSymbol := ctrl.GetString("symbol")
	paramsPage := ctrl.GetString("page", "1")
	paramsLimit := ctrl.GetString("limit", "20")
	paramsStartTime := ctrl.GetString("start_time")
	paramsEndTime := ctrl.GetString("end_time")
	paramsType := ctrl.GetString("type") // open, close, all
	
	page, _ := strconv.Atoi(paramsPage)
	limit, _ := strconv.Atoi(paramsLimit)
	offset := (page - 1) * limit
	
	o := orm.NewOrm()
	var testResults []models.TestStrategyResults
	
	var query = o.QueryTable("test_strategy_results").
		OrderBy("-CreateTime")
		
	if (paramsSymbol != "") {
		query = query.Filter("Symbol__icontains", paramsSymbol)
	}
	if (paramsStartTime != "") {
		query = query.Filter("CreateTime__gte", paramsStartTime)
	}
	if (paramsEndTime != "") {
		query = query.Filter("CreateTime__lte", paramsEndTime)
	}
	if (paramsType == "open") {
		query = query.Filter("close_price", "0")
	} else if (paramsType == "close") {
		query = query.Exclude("close_price", "0")
	}
	total, err1 := query.Count()
	if err1 != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err1.Error()))
	}
	
	queryPagination := query.Limit(limit, offset)
	
	_, err2 := queryPagination.All(&testResults)
	if err2 != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err2.Error()))
	}

	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": map[string]interface{} {
			"total": total,
			"list": testResults,
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
