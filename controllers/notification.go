package controllers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"go_binance_futures/middlewares"
	"go_binance_futures/models"
	"go_binance_futures/utils"
	"go_binance_futures/webnotification"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/server/web"
	"github.com/gorilla/websocket"
)

type NotificationController struct {
	web.Controller
}

func (ctrl *NotificationController) Get() {
	page, _ := strconv.Atoi(ctrl.GetString("page", "1"))
	limit, _ := strconv.Atoi(ctrl.GetString("limit", "50"))
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	o := orm.NewOrm()
	query := o.QueryTable(new(models.Notification))
	if ctrl.GetString("unread_only") == "1" {
		query = query.Filter("is_read", 0)
	}
	total, err := query.Count()
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	var list []models.Notification
	if _, err := query.OrderBy("-create_time").Limit(limit, (page-1)*limit).All(&list); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	unreadCount, err := o.QueryTable(new(models.Notification)).Filter("is_read", 0).Count()
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]any{
		"code": 200,
		"data": map[string]any{
			"total":        total,
			"unread_count": unreadCount,
			"list":         list,
		},
		"msg": "success",
	})
}

func (ctrl *NotificationController) Read() {
	id, err := strconv.ParseInt(ctrl.Ctx.Input.Param(":id"), 10, 64)
	if err != nil || id <= 0 {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "invalid notification id"))
		return
	}
	notification := &models.Notification{ID: id}
	o := orm.NewOrm()
	if err := o.Read(notification); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(404, nil, "notification not found"))
		return
	}
	notification.IsRead = 1
	notification.ReadTime = time.Now().UnixMilli()
	if _, err := o.Update(notification, "IsRead", "ReadTime"); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(utils.ResJson(200, map[string]any{"notification": notification}))
}

func (ctrl *NotificationController) ReadAll() {
	now := time.Now().UnixMilli()
	_, err := orm.NewOrm().QueryTable(new(models.Notification)).Filter("is_read", 0).Update(orm.Params{
		"is_read":   1,
		"read_time": now,
	})
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(utils.ResJson(200, nil))
}

type NotificationWebSocketController struct {
	web.Controller
}

var notificationUpgrader = websocket.Upgrader{
	CheckOrigin: func(*http.Request) bool { return true },
}

func (ctrl *NotificationWebSocketController) Get() {
	authorization := strings.TrimSpace(ctrl.GetString("token"))
	if _, err := middlewares.ValidateAuthorization(authorization); err != nil {
		ctrl.Ctx.ResponseWriter.WriteHeader(http.StatusUnauthorized)
		return
	}
	connection, err := notificationUpgrader.Upgrade(ctrl.Ctx.ResponseWriter, ctrl.Ctx.Request, nil)
	if err != nil {
		return
	}
	webnotification.ServeConnection(connection)
}
