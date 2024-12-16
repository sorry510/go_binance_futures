package controllers

import (
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	"go_binance_futures/utils"
	"math"
	"strconv"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/server/web"
)

type AccountController struct {
	web.Controller
}

func (ctrl *AccountController) GetBinanceFuturesAccount() {
	account, err := binance.GetFuturesAccount()
	
	if err != nil {
		ctrl.Ctx.Resp(map[string]interface{} {
			"code": 400,
			"msg": err.Error(),
		})
	}
	
	var assets = make([]*futures.AccountAsset, len(account.Assets))
	for _, asset := range account.Assets {
		walletBalance, _ := strconv.ParseFloat(asset.WalletBalance, 64)
		if walletBalance < 0.0000001 {
			continue
		}
		assets = append(assets, asset)
	}
	
	var positions = make([]*futures.AccountPosition, len(account.Positions))
	for _, position := range account.Positions {
		positionAmt, _ := strconv.ParseFloat(position.PositionAmt, 64)
		positionAmtFloatAbs := math.Abs(positionAmt) // 空单为负数,纠正为绝对值
		if positionAmtFloatAbs < 0.0000001 {
			continue
		}
		positions = append(positions, position)
	}
	
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": map[string]interface{} {
			"assets": assets,
			"positions": positions,
		},
		"msg": "success",
	})
}

func (ctrl *AccountController) GetBinanceFuturesPositions() {
	allPositions, err := binance.GetPosition(binance.PositionParams{})
	if err != nil {
		ctrl.Ctx.Resp(map[string]interface{} {
			"code": 400,
			"msg": err.Error(),
		})
	}
	
	var positions []*futures.PositionRisk
	for _, position := range allPositions {
		positionAmt, _ := strconv.ParseFloat(position.PositionAmt, 64)
		positionAmtFloatAbs := math.Abs(positionAmt) // 空单为负数,纠正为绝对值
		if positionAmtFloatAbs < 0.0000001 {
			continue
		}
		positions = append(positions, position)
	}

	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": map[string]interface{} {
			"positions": positions,
		},
		"msg": "success",
	})
}

func (ctrl *AccountController) GetBinanceFuturesOpenOrders() {
	allOpenOrders, err := binance.GetOpenOrder()
	
	if err != nil {
		ctrl.Ctx.Resp(map[string]interface{} {
			"code": 400,
			"msg": err.Error(),
		})
	}

	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": map[string]interface{} {
			"openOrders": allOpenOrders,
		},
		"msg": "success",
	})
}

func (ctrl *AccountController) GetLocalFuturesPositions() {
	var positions []models.FuturesPosition
	o := orm.NewOrm()
	sql := "SELECT f.id, f.symbol, f.side, f.amount, f.leverage, f.margin_type, f.isolated_wallet, f.entry_price, s.close as mark_price FROM `futures_positions` f LEFT JOIN symbols s ON f.symbol = s.symbol where 1 = 1"
	sql += ` and f.amount <> '0'`
	_, err := o.Raw(sql).QueryRows(&positions)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
	}
	
	var usePositions []models.FuturesPosition
	for _, position := range positions {
		positionAmt, _ := strconv.ParseFloat(position.Amount, 64)
		positionAmtFloatAbs := math.Abs(positionAmt) // 空单为负数,纠正为绝对值
		if positionAmtFloatAbs < 0.0000001 {
			continue
		}
		enterPrice_float64, _ := strconv.ParseFloat(position.EntryPrice, 64)
		markPrice_float64, _ := strconv.ParseFloat(position.MarkPrice, 64)
		unRealizedProfit := (markPrice_float64 - enterPrice_float64) * positionAmt // 未实现盈亏
		position.UnrealizedProfit = strconv.FormatFloat(unRealizedProfit, 'f', -1, 64)
		
		usePositions = append(usePositions, position)
	}
	
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": map[string]interface{} {
			"positions": usePositions,
		},
		"msg": "success",
	})
}

func (ctrl *AccountController) GetLocalFuturesOpenOrders() {
	var orders []models.FuturesOrder
	o := orm.NewOrm()
	sql := "SELECT * FROM `futures_orders` as f where 1 = 1"
	sql += ` and (f.status = 'NEW' or f.status = 'PARTIALLY_FILLED')` // 下单类型
	_, err := o.Raw(sql).QueryRows(&orders)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
	}
	
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": map[string]interface{} {
			"openOrders": orders,
		},
		"msg": "success",
	})
}

func (ctrl *AccountController) EditLocalFuturesPositions() {
	id := ctrl.Ctx.Input.Param(":id")
	var position models.FuturesPosition
	o := orm.NewOrm()
	o.QueryTable(position.TableName()).Filter("Id", id).One(&position)
	
	ctrl.BindJSON(&position)
	
	_, err := o.Update(&position) // _ 是受影响的条数
    if err != nil {
        // 处理错误
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
    }
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": position,
		"msg": "success",
	})
}

func (ctrl *AccountController) DelLocalFuturesPositions() {
	id := ctrl.Ctx.Input.Param(":id")
	var position models.FuturesPosition
	o := orm.NewOrm()
	o.QueryTable(position.TableName()).Filter("Id", id).One(&position)
	
	_, err := o.Delete(&position)
	if err != nil {
		// 处理错误
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": nil,
		"msg": "success",
	})
}