package line

// 技术指标
import (
	"fmt"
	"go_binance_futures/utils"
	"math"
)

// 简单移动平均（SMA）price数据从时间最近到最远 ma = (p1 + p2 + ... + pn) / n
func CalculateSimpleMovingAverage(price []float64, period int) ([]float64, error) {
	if len(price) < period * 2 {
		return nil, fmt.Errorf("insufficient data for period %d", period)
	}
	
	sma := make([]float64, period)
	
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += price[i]
	}
	sma[0] = sum / float64(period)
	
	for i := 1; i < period; i++ {
		sum += price[period + i - 1] - price[i - 1] // 删掉前一个添加后一个
		sma[i] = sum / float64(period)
	}
	
	return sma, nil
}

// 指数移动平均（EMA）price数据从时间最近到最远 ema[t] = α * price[t] + (1 - α) * ema[t-1]; α = 2 / (n + 1)
func CalculateExponentialMovingAverage(price []float64, period int) ([]float64, error) {
	if len(price) < period {
		return nil, fmt.Errorf("insufficient data for period %d", period)
	}
	
	price = utils.ReverseArray(price) // 时间由远到近

	alpha := 2.0 / (float64(period) + 1)
	ema := make([]float64, len(price))
	ema[0] = calculateAverage(price[0:period])
	for i := 1; i < len(price); i++ {
		ema[i] = alpha * price[i] + (1.0 - alpha) * ema[i-1]
	}

	return utils.ReverseArray(ema), nil
}

// 布林带(boll) 中轨线（MB，通常为移动平均线）、上轨线（UP，通常为中轨线加上一定倍数的标准差）和下轨线（DN，通常为中轨线减去相同倍数的标准差）
// 默认 period = 21, stdDevMultiplier = 2, 返回的切片长度 len(clonePrices)-period+1
func CalculateBollingerBands(clonePrices []float64, period int, stdDevMultiplier float64) (up, mb, dn []float64, err error) {
	if len(clonePrices) < period {
		return nil, nil, nil, fmt.Errorf("insufficient data for period %d", period)
	}
	
	clonePrices = utils.ReverseArray(clonePrices) // 时间由远到近

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

	return utils.ReverseArray(up), utils.ReverseArray(mb), utils.ReverseArray(dn), nil
}

// 平均数
func calculateAverage(values []float64) float64 {
	var sum float64
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}

// 所选周期 = 14
// 平均收益 = 前 14 天所有正向差额之和 / 14
// 平均损失 = 前 14 天所有负向差额之和 / 14
// RS = 平均收益 / 平均损失
// RSI = 100 - (100 / (1 + RS))
// 对于后续 RSI 周期，可以使用平滑移动平均法来更新平均收益和平均损失。
// 新平均收益 = [(前一个周期的平均收益 × (周期 - 1)) + 当前周期的收益] / 周期
// 新平均损失 = [(前一个周期的平均损失 × (周期 - 1)) + 当前周期的损失] / 周期
func CalculateRSI(prices []float64, period int) ([]float64, error) {
	if len(prices) < period {
		return nil, fmt.Errorf("price slice too short for period %d", period)
	}

	prices = utils.ReverseArray(prices) // 时间由远到近
	
	// 初始化收益和损失切片
	gains := make([]float64, len(prices)-1)
	losses := make([]float64, len(prices)-1)

	// 计算每日的差额
	for i := 1; i < len(prices); i++ {
		diff := prices[i] - prices[i-1]
		if diff > 0 {
			gains[i-1] = diff
		} else {
			losses[i-1] = math.Abs(diff)
		}
	}

	// 计算前 period 天的平均收益和平均损失
	var sumGains, sumLosses float64
	for i := 0; i < period; i++ {
		sumGains += gains[i]
		sumLosses += losses[i]
	}

	avgGain := sumGains / float64(period)
	avgLoss := sumLosses / float64(period)

	// 初始化 RSI 切片
	rsiValues := make([]float64, len(prices)-period+1)

	// 计算第一个 RSI 值
	if avgLoss == 0 {
		rsiValues[0] = 100.0
	} else {
		rs := avgGain / avgLoss
		rsiValues[0] = 100 - (100 / (1 + rs))
	}

	// 计算后续的 RSI 值
	for i := period; i < len(prices); i++ {
		newGain := gains[i-1]
		newLoss := losses[i-1]

		avgGain = ((avgGain * float64(period - 1)) + newGain) / float64(period)
		avgLoss = ((avgLoss * float64(period - 1)) + newLoss) / float64(period)

		if avgLoss == 0 {
			rsiValues[i-period+1] = 100.0
		} else {
			rs := avgGain / avgLoss
			rsiValues[i-period+1] = 100 - (100 / (1 + rs))
		}
	}

	return utils.ReverseArray(rsiValues), nil
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
 * 是否只产生过一次金叉(短线穿越长线一次，没有反复穿越)
 * @param ma1 短线
 * @param ma2 长线
 * @param num 数据数
 * @returns Boolean
 */
func Kdj(ma1 []float64, ma2[]float64, num int) bool {
	if (ma1 == nil || ma2 == nil) {
		return false
	}
	if (len(ma1) < num || len(ma2) < num) {
		return false
	}
	if (ma1[0] < ma2[0]) {
		// 最新数据的必须是短线在上
		return false
	}
	k := 0
	for i := 1; i < num; i++ {
		if (ma1[i] < ma2[i]) {
			// 发生过短线在下，说明产生过死叉
			k = i
			break
		}
	}
	// 之后数据不能再重新产生交叉
	for i := k; i < num; i++ {
		if (ma1[i] > ma2[i]) {
			return false
		}
	}
	return k > 0
}

// 计算真实范围 TR= max(High−Low,∣High−PreviousClose∣,∣Low−PreviousClose∣) 数据时间由近到远
func calculateTrueRange(high, low, close []float64) []float64 {
	tr := make([]float64, len(high))
	if len(high) == 0 {
		return tr
	}

	for i := 0; i < len(high) - 1; i++ {
		hl := high[i] - low[i]
		hpc := math.Abs(high[i] - close[i+1])
		lpc := math.Abs(low[i] - close[i+1])
		tr[i] = math.Max(hl, math.Max(hpc, lpc))
	}
	tr[len(high)-1] = high[len(high)-1] - low[len(high)-1]

	return tr
}

// 肯纳特通道
func CalculateKeltnerChannels(high, low, close []float64, period int, multiplier float64) ([]float64, []float64, []float64) {
	ma, _ := CalculateExponentialMovingAverage(close, period)
	tr := calculateTrueRange(high, low, close)
	atr, _ := CalculateExponentialMovingAverage(tr, period)

	upper := make([]float64, len(ma))
	lower := make([]float64, len(ma))

	for i := 0; i < len(ma); i++ {
		upper[i] = ma[i] + multiplier * atr[i]
		lower[i] = ma[i] - multiplier * atr[i]
	}

	return upper, ma, lower
}

type Candle struct {
    Open  float64
    Close float64
    High  float64
    Low   float64
}

// 根据日本蜡烛图帮我写一个黑云压顶的函数
func IsDarkCloudCover(first, second Candle) bool {
    // First candle is bullish
    isFirstBullish := first.Close > first.Open
    // Second candle is bearish
    isSecondBearish := second.Close < second.Open
    // Second candle opens above the first candle's close
    opensAboveFirstClose := second.Open > first.Close
    // Second candle closes inside the body of the first candle
    closesInsideFirstBody := second.Close < first.Open && second.Close > first.Close * 0.5 + first.Open * 0.5

    return isFirstBullish && isSecondBearish && opensAboveFirstClose && closesInsideFirstBody
}