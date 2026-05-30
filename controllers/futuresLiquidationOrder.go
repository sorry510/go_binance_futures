package controllers

import (
	"strconv"
	"strings"

	"go_binance_futures/models"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/server/web"
)

type FuturesLiquidationOrderController struct {
	web.Controller
}

func (ctrl *FuturesLiquidationOrderController) Get() {
	paramsSymbol := strings.TrimSpace(ctrl.GetString("symbol"))
	paramsSide := strings.TrimSpace(ctrl.GetString("side"))
	paramsStartTime := strings.TrimSpace(ctrl.GetString("start_time"))
	paramsEndTime := strings.TrimSpace(ctrl.GetString("end_time"))
	paramsMinNotional := strings.TrimSpace(ctrl.GetString("min_notional"))
	paramsPage := ctrl.GetString("page", "1")
	paramsLimit := ctrl.GetString("limit", "20")

	page, _ := strconv.Atoi(paramsPage)
	limit, _ := strconv.Atoi(paramsLimit)
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 500 {
		limit = 500
	}
	offset := (page - 1) * limit

	o := orm.NewOrm()
	query := o.QueryTable(new(models.FuturesLiquidationOrder))

	if paramsSymbol != "" {
		query = query.Filter("symbol__icontains", strings.ToUpper(paramsSymbol))
	}
	if paramsSide != "" && paramsSide != "all" {
		query = query.Filter("side", strings.ToUpper(paramsSide))
	}
	if startTime := parseTimestamp(paramsStartTime); startTime > 0 {
		query = query.Filter("event_time__gte", startTime)
	}
	if endTime := parseTimestamp(paramsEndTime); endTime > 0 {
		query = query.Filter("event_time__lte", endTime)
	}
	if minNotional, _ := strconv.ParseFloat(paramsMinNotional, 64); minNotional > 0 {
		query = query.Filter("notional__gte", minNotional)
	}

	total, err := query.Count()
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}

	var list []models.FuturesLiquidationOrder
	_, err = query.OrderBy("-event_time").Limit(limit, offset).All(&list)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}

	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"total": total,
			"list":  list,
		},
		"msg": "success",
	})
}

func parseTimestamp(value string) int64 {
	timestamp, _ := strconv.ParseInt(value, 10, 64)
	if timestamp > 0 && timestamp < 1000000000000 {
		return timestamp * 1000
	}
	return timestamp
}
