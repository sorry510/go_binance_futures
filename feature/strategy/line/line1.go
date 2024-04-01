package line

import (
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/utils"
	"math"
	"strconv"

	"github.com/adshao/go-binance/v2/futures"
)

// var hold_max_time, _ = config.String("coin::hold_max_time")
// var hold_max_time_float64, _ = strconv.ParseFloat(hold_max_time, 64)
// var profit, _ = config.String("trade::profit")
// var profit_float64, _ = strconv.ParseFloat(profit, 64)

// var loss, _ = config.String("trade::loss")
// var loss_float64, _ = strconv.ParseFloat(loss, 64)
type Line struct {
	Position string
	High float64
	Low float64
	Close float64
	Open float64
	TradeNum int64
}
type LineData struct {
	MaxIndex int
	MinIndex int
	Line []*Line
}

type TradeLine1 struct {
}

func (tradeLine1 TradeLine1) GetCanLongOrShort(symbol string) (canLong bool, canShort bool) {
	kline_3m, err := binance.GetKlineData(symbol, "3m", 50)
	if err != nil {
		return false, false
	}
	line3m_result := normalizationLineData(kline_3m)
	if checkLongLine3m(line3m_result) {
		return true, false
	}
	if checkShortLine3m(line3m_result) {
		return false, true
	}
	return false, false
}

// 达到止盈或止损后判断是否可以平仓
func (tradeLine1 TradeLine1) CanOrderComplete(symbol string, positionSide string) (complete bool) {
	lines, err := binance.GetKlineData(symbol, "1m", 2) // 1min 线最近2条
	if err != nil {
		return true
	}
	close0, _ := strconv.ParseFloat(lines[0].Close, 64)
	close1, _ := strconv.ParseFloat(lines[1].Close, 64)
	if positionSide == "LONG" {
		return close0 < close1 // 价格在下跌中
	} else if positionSide == "SHORT" {
		return close0 > close1 // 价格在上涨中
	} else {
		return false
	}
}

func (tradeLine1 TradeLine1) AutoStopOrder(position *futures.PositionRisk, nowProfit float64) (stop bool) {
	// symbol := position.Symbol
	// side := position.PositionSide
	// // if hold_max_time_float64 > 0 {
	// // 	// 最大持仓时间(api的bug，没有UpdateTime字段)
	// // 	// if position.UpdateTime+hold_max_time_float64*60*60*1000 < time.Now().UnixNano()/1e6 {
	// // 	// 	return true
	// // 	// }
	// // }
	// if side == "LONG" {
	// 	if (nowProfit < profit_float64 * 0.6 && nowProfit > 0) || nowProfit < -profit_float64 {
	// 		return true
	// 	}
	// }
	return false
}

// 归一化处理k线数据
func normalizationLineData(data []*futures.Kline) (*LineData) {
	maxIndex := 0
	maxPrice := 0.0
	minIndex := 0
	minPrice := 0.0
	line := make([]*Line, len(data))
	for key, item := range data {
		open, _ := strconv.ParseFloat(item.Open, 64)
		high, _ := strconv.ParseFloat(item.High, 64)
		low, _ := strconv.ParseFloat(item.Low, 64)
		close, _ := strconv.ParseFloat(item.Close, 64)
		if key == 0 {
			maxPrice = high
			minPrice = close
		} else {
			if (high > maxPrice) {
				maxPrice = high
				maxIndex = key
			}
			if (low < minPrice) {
				minPrice = low
				minIndex = key
			}
		}
		position := "LONG"
		if close < open {
			position = "SHORT"
		}
		line[key] = &Line{
			Position: position,
			High: high,
			Low: low,
			Close: close,
			Open: open,
			TradeNum: item.TradeNum,
		}
	}
	return &LineData{
		MaxIndex: maxIndex,
		MinIndex: minIndex,
		Line: line,
	}
}

func checkLongLine3m(lineData *LineData) bool {
	maxIndex := lineData.MaxIndex
	minIndex := lineData.MinIndex
	line := lineData.Line
	if minIndex >= 1 && minIndex <= 3 && maxIndex >= 8 {
		// 最低点在3分前，最高点之前24分
		ma3List := utils.MaNList(getClosePrices(line), 3, 20) // 3min kline 最近20条ma均线，时间从最新到最老
		linePoint := line[minIndex] // 最低的那个line
		underLength := math.Abs(linePoint.Close - linePoint.Low) // 下影线长度
		entityLength := math.Abs(linePoint.Open - linePoint.Close) // 实体长度
		if utils.IsDesc(ma3List[0:minIndex]) && // 最低点到现在在涨
			getRightLine(line[minIndex:minIndex+9], "SHORT") && // 最低点到最低点+9个line里面至少7个是红线
			linePoint.Position == "SHORT" && // 最低点的line是红线
			(underLength / entityLength) > 0.66 { // 下影线长度  实体长度
				return true
		}
	}
	return false
}

func checkShortLine3m(lineData *LineData) bool {
	maxIndex := lineData.MaxIndex
	minIndex := lineData.MinIndex
	line := lineData.Line
	if maxIndex >= 1 && maxIndex <= 3 && minIndex >= 8 {
		// 最高点在3分前，最低点之前30分
		ma3List := utils.MaNList(getClosePrices(line), 3, 20) // 3min kline 最近20条ma均线，时间从最新到最老
		linePoint := line[maxIndex] // 最高的那个line
		upperLength := math.Abs(linePoint.High - linePoint.Close) // 上影线长度
		entityLength := math.Abs(linePoint.Open - linePoint.Close) // 实体长度
		if utils.IsAsc(ma3List[0:maxIndex]) &&
			getRightLine(line[maxIndex:maxIndex+9], "LONG") && // 最低点到最低点+9个line里面至少7个是绿线
			linePoint.Position == "LONG" && // 最低点的line是红线
			(upperLength / entityLength) > 0.66 { // 上影线长度 > 实体长度
				return true
		}
	}
	return false
}

func getClosePrices(data []*Line) ([]float64) {
	clonePrices := make([]float64, len(data))
	for key, line := range data {
		clonePrices[key] = line.Close
	}
	return clonePrices
}

// 得到适配的line
func getRightLine(data []*Line, position string) bool {
	positionCount := 0
	for _, line := range data {
		if line.Position == position {
			positionCount++
		}
	}
	return len(data) - positionCount <= 2
}