package line

import (
	"fmt"
	"go_binance_futures/utils"
	"math"
	"strconv"

	"github.com/adshao/go-binance/v2/futures"
)

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

// 获取收盘价列表
func GetClosePrices(data []*Line) ([]float64) {
	clonePrices := make([]float64, len(data))
	for key, line := range data {
		clonePrices[key] = line.Close
	}
	return clonePrices
}

// 从k线获取收盘价列表
func GetLineClosePrices(data []*futures.Kline) ([]float64) {
	clonePrices := make([]float64, len(data))
	for key, item := range data {
		close, _ := strconv.ParseFloat(item.Close, 64)
		clonePrices[key] = close
	}
	return clonePrices
}

// 获取某中类型的line的数量是否超过阈值
func getRightLine(data []*Line, position string) bool {
	positionCount := 0
	for _, line := range data {
		if line.Position == position {
			positionCount++
		}
	}
	return len(data) - positionCount <= 2
}

// 简单移动平均（MA）返回一个数组从时间最近到最远
func CalculateSimpleMovingAverage(clonePrices []float64, period int) ([]float64, error) {
	if len(clonePrices) < period {
		return nil, fmt.Errorf("insufficient data for period %d", period)
	}

	maValues := make([]float64, len(clonePrices)-period+1)
	for i := period - 1; i < len(clonePrices); i++ {
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += clonePrices[j]
		}
		maValues[i-period+1] = sum / float64(period)
	}
	return maValues, nil
}

// 指数移动平均（EMA）
func CalculateExponentialMovingAverage(clonePrices []float64, period int) ([]float64, error) {
	if len(clonePrices) < period {
		return nil, fmt.Errorf("insufficient data for period %d", period)
	}

	alpha := 2.0 / (float64(period) + 1)
	emaValues := make([]float64, len(clonePrices))
	emaValues[0] = calculateAverage(clonePrices[0:period])
	for i := 1; i < len(clonePrices); i++ {
		emaValues[i] = alpha * clonePrices[i] + (1.0-alpha)*emaValues[i-1]
	}

	return emaValues, nil // Return EMA values after the initial period is "warmed up"
}

// 布林带(boll) 中轨线（MB，通常为移动平均线）、上轨线（UP，通常为中轨线加上一定倍数的标准差）和下轨线（DN，通常为中轨线减去相同倍数的标准差）
// 默认 period = 21, stdDevMultiplier = 2, 返回的切片长度 len(clonePrices)-period+1
func CalculateBollingerBands(clonePrices []float64, period int, stdDevMultiplier float64) (up, mb, dn []float64, err error) {
	if len(clonePrices) < period {
		return nil, nil, nil, fmt.Errorf("insufficient data for period %d", period)
	}

	// Calculate the simple moving average (SMA) as the middle band (MB)
	mb = make([]float64, len(clonePrices)-period+1)
	for i := period - 1; i < len(clonePrices); i++ {
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += clonePrices[j]
		}
		mb[i-period+1] = sum / float64(period)
	}

	// Calculate the standard deviation (SD) over the same period
	sd := make([]float64, len(mb))
	for i := 0; i < len(mb); i++ { // Fix: Use correct index range for sd calculation
		sumOfSquares := 0.0
		for j := i; j < i+period; j++ { // Fix: Use correct index range for sumOfSquares calculation
			deviation := clonePrices[j] - mb[i]
			sumOfSquares += deviation * deviation
		}
		sd[i] = math.Sqrt(sumOfSquares / float64(period))
	}

	// Compute the upper and lower bands (UP and DN) as MB ± stdDevMultiplier * SD
	up = make([]float64, len(mb))
	dn = make([]float64, len(mb))
	for i := range mb {
		up[i] = mb[i] + stdDevMultiplier*sd[i]
		dn[i] = mb[i] - stdDevMultiplier*sd[i]
	}

	return up, mb, dn, nil
}

// 平均数
func calculateAverage(values []float64) float64 {
	var sum float64
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}

// CalculateRSI 计算给定价格序列的 RSI 值
func CalculateRSI(prices []float64, period int) ([]float64, error) {
	if len(prices) < period+1 {
		return nil, fmt.Errorf("insufficient data for RSI calculation (need at least %d periods, got %d)", period, len(prices))
	}

	rsiValues := make([]float64, len(prices)-period+1)

	for i := period; i <= len(prices)-1; i++ {
		// 计算前 period 个周期的平均收益和平均损失
		gains := make([]float64, period)
		losses := make([]float64, period)

		for j := i - period + 1; j <= i; j++ {
			change := prices[j] - prices[j-1]
			if change > 0 {
				gains[j-(i-period+1)] = change
			} else {
				losses[j-(i-period+1)] = math.Abs(change)
			}
		}

		averageGain := sum(gains) / float64(period)
		averageLoss := sum(losses) / float64(period)

		// 防止除以零，当所有收益或损失为零时，使用 100 或 0 来计算 RSI
		if averageLoss == 0 {
			rsiValues[i-period+1] = 100
		} else if averageGain == 0 {
			rsiValues[i-period+1] = 0
		} else {
			rsi := 100 * (averageGain / (averageGain + averageLoss))
			rsiValues[i-period+1] = rsi
		}
	}

	return rsiValues, nil
}

// sum 计算浮点数切片的总和
func sum(numbers []float64) float64 {
	sum := 0.0
	for _, num := range numbers {
		sum += num
	}
	return sum
}


/**
 * 是否产生过金叉
 * @param ma1 短线
 * @param ma2 长线
 * @param num 数据数
 * @param string side 方向 
 * @returns Boolean
 */
func Kdj(ma1 []float64, ma2[]float64, num int, side string) bool {
	if (side == "LONG") {
		if (ma1[0] < ma2[0]) {
			// 最新数据的必须是短线在上
			return false
		}
		for i := 1; i < num; i++ {
			if (ma1[i] < ma2[i]) {
				// 发生过短线在下，说明产生过金叉
				return true
			}
		}
		return false
	} else if (side == "SHORT") {
	  	if ma1[0] > ma2[0] {
			// 最后的必须是短线在上
			return false
		}
		for i := 1; i < num; i++ {
			if (ma1[i] > ma2[i]) {
				// 发生过短线在上，说明产生过死叉
				return true
			}
		}
	  	return false
	}
	return false
}