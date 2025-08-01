package controllers

import (
	"strconv"

	"go_binance_futures/models"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/server/web"
)

type NotifyConfigController struct {
	web.Controller
}

func (ctrl *NotifyConfigController) Get() {
	
	o := orm.NewOrm()
	var notifyConfigs []models.NotifyConfig
	query := o.QueryTable("notify_config")
	_, err := query.OrderBy("ID").All(&notifyConfigs)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
	}
	
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": notifyConfigs,
		"msg": "success",
	})
}
	
func (ctrl *NotifyConfigController) Edit() {
	id := ctrl.Ctx.Input.Param(":id")
	notifyConfig := new(models.NotifyConfig)
	ctrl.BindJSON(&notifyConfig)
	intId, _ := strconv.ParseInt(id, 10, 64)
	notifyConfig.ID = intId


	o := orm.NewOrm()
	_, err := o.Update(notifyConfig) // _ 是受影响的条数
    if err != nil {
        // 处理错误
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "修改失败"))
		return
    }
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": notifyConfig,
		"msg": "success",
	})
}

func (ctrl *NotifyConfigController) Delete() {
	id := ctrl.Ctx.Input.Param(":id")
	notifyConfig := new(models.NotifyConfig)
	intId, _ := strconv.ParseInt(id, 10, 64)
	notifyConfig.ID = intId
	o := orm.NewOrm()

	_, err := o.Delete(notifyConfig)
    if err != nil {
        // 处理错误
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "删除错误"))
		return
    }
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"msg": "success",
	})
}

func (ctrl *NotifyConfigController) Post() {
	notifyConfig := new(models.NotifyConfig)
	ctrl.BindJSON(&notifyConfig)

	notifyConfig.Enable = 1 // 默认开启

	o := orm.NewOrm()
	id, err := o.Insert(notifyConfig)
	
    if err != nil {
        // 处理错误
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "新增失败"))
		return
    }
	notifyConfig.ID = id
	
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": notifyConfig,
		"msg": "success",
	})
}
