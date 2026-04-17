package controllers

import (
	"strconv"
	"strings"

	"go_binance_futures/models"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/server/web"
)

type FuturesMarketNoticeLogController struct {
	web.Controller
}

func (ctrl *FuturesMarketNoticeLogController) Get() {
	paramsSymbol := strings.TrimSpace(ctrl.GetString("symbol"))
	paramsWindow := strings.TrimSpace(ctrl.GetString("window"))
	paramsDirection := strings.TrimSpace(ctrl.GetString("direction"))
	paramsStartTime := strings.TrimSpace(ctrl.GetString("start_time"))
	paramsEndTime := strings.TrimSpace(ctrl.GetString("end_time"))
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
	offset := (page - 1) * limit

	o := orm.NewOrm()
	query := o.QueryTable(new(models.FuturesMarketNoticeLog))

	if paramsSymbol != "" {
		query = query.Filter("symbol__icontains", strings.ToUpper(paramsSymbol))
	}
	if paramsWindow != "" && paramsWindow != "all" {
		query = query.Filter("window_size", paramsWindow)
	}
	if paramsDirection != "" && paramsDirection != "all" {
		query = query.Filter("direction", paramsDirection)
	}

	if paramsStartTime != "" {
		startTime, _ := strconv.ParseInt(paramsStartTime, 10, 64)
		if startTime > 0 {
			if startTime < 1000000000000 {
				startTime = startTime * 1000
			}
			query = query.Filter("createTime__gte", startTime)
		}
	}
	if paramsEndTime != "" {
		endTime, _ := strconv.ParseInt(paramsEndTime, 10, 64)
		if endTime > 0 {
			if endTime < 1000000000000 {
				endTime = endTime * 1000
			}
			query = query.Filter("createTime__lte", endTime)
		}
	}

	var total int64
	var list []models.FuturesMarketNoticeLog
	total, err := query.Count()
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}

	_, err = query.OrderBy("-createTime").Limit(limit, offset).All(&list)
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
