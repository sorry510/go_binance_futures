package coin

import (
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	"math/rand"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// 最近5min交易过的币种订单
func getLimitMinOrder(minute int64) (symbols map[string]bool) {
	nowTime := time.Now().Unix() * 1000 // 毫秒时间戳
	orders, _ := binance.GetOrders(binance.ListOrderParams{
		StartTime: nowTime - minute * 60 * 1000,
	})
	symbols = make(map[string]bool)
	for _, order := range orders {
		symbols[order.Symbol] = true
	}
	return symbols
}

// 最近min交易过的币种local订单
func getLimitMinLocalOrder(minute int64) (symbols map[string]bool) {
	startTime := time.Now().Unix() * 1000 - minute * 60 * 1000 // 毫秒时间戳
	o := orm.NewOrm()
	var orders []models.Order
	_, _ = o.QueryTable("order").
		Filter("UpdateTime__gte", startTime).
		Filter("Side", "close").
		All(&orders)
	symbols = make(map[string]bool)
	for _, order := range orders {
		symbols[order.Symbol] = true
	}
	return symbols
}


func GetRandArr(arr []*models.Symbols, num int) (result []*models.Symbols) {
	if num >= len(arr) {
        return arr
    }
	
	indices := make(map[int]bool, len(arr))
    for len(indices) < num {
        index := rand.Intn(len(arr))
        indices[index] = true
    }

	result = make([]*models.Symbols, num)
    i := 0
    for k := range indices {
        result[i] = arr[k]
        i++
    }

    return result
}