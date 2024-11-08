package utils

import (
	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
)

func GetSystemConfig() (systemConfig models.Config, err error) {
	o := orm.NewOrm()
	err = o.QueryTable("config").Filter("Id", 1).One(&systemConfig)
	if err != nil {
		return systemConfig, err
	}
	return systemConfig, nil
}

