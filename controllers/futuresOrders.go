package controllers

import (
	"strconv"
	"strings"

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
	page, _ := strconv.Atoi(ctrl.GetString("page", "1"))
	limit, _ := strconv.Atoi(ctrl.GetString("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 500 {
		limit = 500
	}
	conditions := []string{"t.account_id = 'main'"}
	args := make([]interface{}, 0)
	add := func(field, value string, allowed ...string) bool {
		if value == "" || value == "all" {
			return true
		}
		valid := len(allowed) == 0
		for _, a := range allowed {
			if value == a {
				valid = true
				break
			}
		}
		if !valid {
			return false
		}
		conditions = append(conditions, field+" = ?")
		args = append(args, value)
		return true
	}
	symbol := strings.TrimSpace(ctrl.GetString("symbol"))
	if symbol != "" {
		conditions = append(conditions, "t.symbol LIKE ?")
		args = append(args, "%"+symbol+"%")
	}
	if !add("t.positionSide", strings.ToUpper(strings.TrimSpace(ctrl.GetString("position_side"))), "LONG", "SHORT", "ALL") {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "invalid position_side"))
		return
	}
	if !add("t.side", strings.ToUpper(strings.TrimSpace(ctrl.GetString("type"))), "BUY", "SELL", "ALL") {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "invalid side"))
		return
	}
	if !add("t.status", strings.ToUpper(strings.TrimSpace(ctrl.GetString("status"))), "NEW", "PARTIALLY_FILLED", "FILLED", "CANCELED", "REJECTED", "EXPIRED", "ALL") {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "invalid status"))
		return
	}
	for _, v := range []struct{ field, param string }{{"t.updateTime >= ?", "start_time"}, {"t.updateTime <= ?", "end_time"}} {
		raw := strings.TrimSpace(ctrl.GetString(v.param))
		if raw == "" {
			continue
		}
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value < 0 {
			ctrl.Ctx.Resp(utils.ResJson(400, nil, "invalid "+v.param))
			return
		}
		conditions = append(conditions, v.field)
		args = append(args, value)
	}
	where := " WHERE " + strings.Join(conditions, " AND ")
	o := orm.NewOrm()
	var total int64
	if err := o.Raw("SELECT COUNT(*) FROM `futures_orders` t"+where, args...).QueryRow(&total); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	var orders []FuturesOrderTableList
	listArgs := append(append([]interface{}{}, args...), limit, (page-1)*limit)
	sql := "SELECT t.*, COALESCE(s.close,'') AS now_price FROM `futures_orders` t LEFT JOIN symbols s ON t.symbol=s.symbol" + where + " ORDER BY t.updateTime DESC LIMIT ? OFFSET ?"
	if _, err := o.Raw(sql, listArgs...).QueryRows(&orders); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": map[string]interface{}{"total": total, "list": orders}, "msg": "success"})
}

// func (ctrl *FuturesOrderController) Delete() {
// 	id := ctrl.Ctx.Input.Param(":id")
// 	o := orm.NewOrm()
// 	_, err := o.Raw("DELETE FROM order where id = ?", id).Exec()
// 	if err != nil {
// 		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
// 		return
// 	}
// 	ctrl.Ctx.Resp(utils.ResJson(200, nil))
// }

// func (ctrl *FuturesOrderController) DeleteAll() {

// 	o := orm.NewOrm()
// 	_, err := o.Raw("DELETE FROM order where 1=1").Exec()
// 	if err != nil {
// 		// 处理错误
// 		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
// 		return
// 	}
// 	ctrl.Ctx.Resp(utils.ResJson(200, nil))
// }
