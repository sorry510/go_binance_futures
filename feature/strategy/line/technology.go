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