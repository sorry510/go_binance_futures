package controllers

import (
	"fmt"
	"strconv"
	"strings"

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

type orderSearchParams struct {
	Symbol       string
	PositionSide string
	StartTime    string
	EndTime      string
	Type         string
}

func (ctrl *OrderController) getSearchParams() orderSearchParams {
	return orderSearchParams{
		Symbol:       strings.TrimSpace(ctrl.GetString("symbol")),
		PositionSide: strings.ToUpper(strings.TrimSpace(ctrl.GetString("position_side"))),
		StartTime:    strings.TrimSpace(ctrl.GetString("start_time")),
		EndTime:      strings.TrimSpace(ctrl.GetString("end_time")),
		Type:         strings.ToLower(strings.TrimSpace(ctrl.GetString("type"))),
	}
}

func orderOpenClause(alias string) string {
	prefix := ""
	if alias != "" {
		prefix = alias + "."
	}
	return prefix + "side = 'open'"
}

func (params orderSearchParams) whereClause(alias string) (string, []interface{}, bool, error) {
	prefix := ""
	if alias != "" {
		prefix = alias + "."
	}
	conditions := make([]string, 0, 5)
	args := make([]interface{}, 0, 5)
	if params.Symbol != "" {
		if strings.ContainsAny(params.Symbol, "%_") {
			return "", nil, false, fmt.Errorf("invalid symbol")
		}
		conditions = append(conditions, prefix+"symbol LIKE ?")
		args = append(args, "%"+params.Symbol+"%")
	}
	if params.PositionSide != "" && params.PositionSide != "ALL" {
		if params.PositionSide != "LONG" && params.PositionSide != "SHORT" {
			return "", nil, false, fmt.Errorf("invalid position_side")
		}
		conditions = append(conditions, prefix+"positionSide = ?")
		args = append(args, params.PositionSide)
	}
	var startTime int64
	var endTime int64
	if params.StartTime != "" {
		value, err := strconv.ParseInt(params.StartTime, 10, 64)
		if err != nil || value <= 0 {
			return "", nil, false, fmt.Errorf("invalid start_time")
		}
		startTime = value
		conditions = append(conditions, prefix+"updateTime >= ?")
		args = append(args, value)
	}
	if params.EndTime != "" {
		value, err := strconv.ParseInt(params.EndTime, 10, 64)
		if err != nil || value <= 0 {
			return "", nil, false, fmt.Errorf("invalid end_time")
		}
		endTime = value
		conditions = append(conditions, prefix+"updateTime <= ?")
		args = append(args, value)
	}
	if startTime > 0 && endTime > 0 && startTime > endTime {
		return "", nil, false, fmt.Errorf("start_time must not exceed end_time")
	}
	switch params.Type {
	case "", "all":
	case "open":
		conditions = append(conditions, prefix+"closeOrderId = 0")
	case "close":
		conditions = append(conditions, prefix+"closeOrderId > 0")
	default:
		return "", nil, false, fmt.Errorf("invalid type")
	}
	if len(conditions) == 0 {
		return "", nil, false, nil
	}
	return " AND " + strings.Join(conditions, " AND "), args, true, nil
}

func (ctrl *OrderController) Get() {
	page, _ := strconv.Atoi(ctrl.GetString("page", "1"))
	limit, _ := strconv.Atoi(ctrl.GetString("limit", "100"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 100
	}
	if limit > 10000 {
		limit = 10000
	}
	where, args, _, err := ctrl.getSearchParams().whereClause("t")
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	offset := (page - 1) * limit
	o := orm.NewOrm()
	var orders []OrderTableList
	selectSQL := "SELECT t.*, COALESCE(s.close, '') AS now_price FROM `order` t LEFT JOIN symbols s ON t.symbol = s.symbol WHERE " + orderOpenClause("t") + where + " ORDER BY t.updateTime DESC LIMIT ? OFFSET ?"
	selectArgs := append(append([]interface{}{}, args...), limit, offset)
	if _, err := o.Raw(selectSQL, selectArgs...).QueryRows(&orders); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	var total int64
	countSQL := "SELECT COUNT(*) FROM `order` t WHERE " + orderOpenClause("t") + where
	if err := o.Raw(countSQL, args...).QueryRow(&total); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{"total": total, "list": orders},
		"msg":  "success",
	})
}

func (ctrl *OrderController) Delete() {
	id, err := strconv.ParseInt(strings.TrimSpace(ctrl.Ctx.Input.Param(":id")), 10, 64)
	if err != nil || id <= 0 {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "invalid order id"))
		return
	}
	o := orm.NewOrm()
	var openOrder models.Order
	if err := o.QueryTable("order").Filter("Id", id).Filter("Side", "open").One(&openOrder); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	tx, err := o.Begin()
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	if openOrder.CloseOrderId > 0 {
		if _, err := tx.Raw("DELETE FROM `order` WHERE id = ? AND side = 'close'", openOrder.CloseOrderId).Exec(); err != nil {
			_ = tx.Rollback()
			ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
			return
		}
	}
	if _, err := tx.Raw("DELETE FROM `order` WHERE id = ? AND side = 'open'", openOrder.ID).Exec(); err != nil {
		_ = tx.Rollback()
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	if err := tx.Commit(); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(utils.ResJson(200, map[string]interface{}{"deleted": 1}))
}

func (ctrl *OrderController) DeleteAll() {
	params := ctrl.getSearchParams()
	where, args, hasSearchCondition, err := params.whereClause("t")
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	if !hasSearchCondition {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "at least one search condition is required"))
		return
	}
	o := orm.NewOrm()
	tx, err := o.Begin()
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	var rows []struct {
		ID           int64 `orm:"column(id)"`
		CloseOrderId int64 `orm:"column(closeOrderId)"`
	}
	if _, err := tx.Raw("SELECT t.id, t.closeOrderId FROM `order` t WHERE "+orderOpenClause("t")+where, args...).QueryRows(&rows); err != nil {
		_ = tx.Rollback()
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	closeIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		if row.CloseOrderId > 0 {
			closeIDs = append(closeIDs, row.CloseOrderId)
		}
	}
	if err := deleteOrderIDs(tx, closeIDs); err != nil {
		_ = tx.Rollback()
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	deleteWhere, deleteArgs, _, err := params.whereClause("")
	if err != nil {
		_ = tx.Rollback()
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	result, err := tx.Raw("DELETE FROM `order` WHERE "+orderOpenClause("")+deleteWhere, deleteArgs...).Exec()
	if err != nil {
		_ = tx.Rollback()
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	if err := tx.Commit(); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	deleted, _ := result.RowsAffected()
	ctrl.Ctx.Resp(utils.ResJson(200, map[string]interface{}{"deleted": deleted, "deleted_close_orders": len(closeIDs)}))
}

func deleteOrderIDs(tx orm.TxOrmer, ids []int64) error {
	const chunkSize = 500
	for start := 0; start < len(ids); start += chunkSize {
		end := start + chunkSize
		if end > len(ids) {
			end = len(ids)
		}
		chunk := ids[start:end]
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(chunk)), ",")
		args := make([]interface{}, len(chunk))
		for i, id := range chunk {
			args[i] = id
		}
		if _, err := tx.Raw("DELETE FROM `order` WHERE side = 'close' AND id IN ("+placeholders+")", args...).Exec(); err != nil {
			return err
		}
	}
	return nil
}
