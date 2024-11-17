package line

import (
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/feature/strategy"
	"go_binance_futures/utils"
	"math"
	"strconv"
)

// 策略逻辑: 看的是 3min k线，高频交易
// 做多逻辑
// 1. 最低点在最近3个line，最高点在8个line之前
// 2. 最低点到现在在涨, 最低点之前9个line里面至少7个是红线，最低点的line是红线，（下影线长度 / 实体长度）/ 0.66
// 做空相反逻辑
type TradeLine1 struct {
}

func (tradeLine1 TradeLine1) GetCanLongOrShort(openParams strategy.OpenParams) (openResult strategy.OpenResult) {
	symbols := openParams.Symbols
	openResult.CanLong = false
	openResult.CanShort = false
	
	kline_3m, err := binance.GetKlineData(symbols.Symbol, "3m", 50)
	if err != nil {
		return openResult
	}
	line3m_result := normalizationLineData(kline_3m)
	if checkLongLine3m(line3m_result) {
		openResult.CanLong = true
	}
	if checkShortLine3m(line3m_result) {
		openResult.CanShort = true
	}
	return openResult
}

// 达到止盈或止损后判断是否可以平仓
func (tradeLine1 TradeLine1) CanOrderComplete(closeParams strategy.CloseParams) (closeResult strategy.CloseResult) {
	symbols := closeParams.Symbols // 交易对
	position := closeParams.Position // 当前仓位
	closeResult.Complete = false
	
	lines, err := binance.GetKlineData(symbols.Symbol, "1m", 2) // 1min 线最近2条
	if err != nil {
		return closeResult
	}
	close0, _ := strconv.ParseFloat(lines[0].Close, 64)
	close1, _ := strconv.ParseFloat(lines[1].Close, 64)
	if position.PositionSide == "LONG" {
		closeResult.Complete = close0 < close1 // 价格在下跌中
	} else if position.PositionSide == "SHORT" {
		closeResult.Complete = close0 > close1 // 价格在上涨中
	} else {
		closeResult.Complete = true
	}
	return closeResult
}

func (tradeLine1 TradeLine1) AutoStopOrder(closeParams strategy.CloseParams) (closeResult strategy.CloseResult) {
	closeResult.Complete = false
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
	return closeResult
}

func checkLongLine3m(lineData *LineData) bool {
	maxIndex := lineData.MaxIndex
	minIndex := lineData.MinIndex
	line := lineData.Line
	if minIndex >= 1 && minIndex <= 3 && maxIndex >= 8 {
		// 最低点在3分前，最高点之前24分
		ma3List := utils.MaNList(GetClosePrices(line), 3, 20) // 3min kline 最近20条ma均线，时间从最新到最老
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
		ma3List := utils.MaNList(GetClosePrices(line), 3, 20) // 3min kline 最近20条ma均线，时间从最新到最老
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
