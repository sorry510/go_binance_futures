package line

// 技术指标
import (
	"fmt"
	"go_binance_futures/utils"
	"math"
)

// 简单移动平均（SMA）price数据从时间最近到最远 ma = (p1 + p2 + ... + pn) / n
func CalculateSimpleMovingAverage(prices []float64, period int) ([]float64, error) {
	if err := validatePeriod(len(prices), period); err != nil {
		return nil, err
	}

	prices = utils.ReverseArray(prices) // 时间由远到近

	sma := make([]float64, len(prices)-period+1)

	// 计算第一个 SMA 值
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += prices[i]
	}
	sma[0] = sum / float64(period)

	// 计算后续的 SMA 值
	for i := period; i < len(prices); i++ {
		sum += prices[i] - prices[i-period]
		sma[i-period+1] = sum / float64(period)
	}

	return utils.ReverseArray(sma), nil
}

// 指数移动平均（EMA）price数据从时间最近到最远 ema[t] = α * price[t] + (1 - α) * ema[t-1]; α = 2 / (n + 1)
func CalculateExponentialMovingAverage(price []float64, period int) ([]float64, error) {
	if err := validatePeriod(len(price), period); err != nil {
		return nil, err
	}

	price = utils.ReverseArray(price) // 时间由远到近

	alpha := 2.0 / (float64(period) + 1)
	ema := make([]float64, len(price)-period+1)
	ema[0] = calculateAverage(price[0:period])
	for i := period; i < len(price); i++ {
		ema[i-period+1] = alpha*price[i] + (1.0-alpha)*ema[i-period]
	}

	return utils.ReverseArray(ema), nil
}

// CalculateMACD returns DIF, DEA, and histogram values ordered from newest to oldest.
func CalculateMACD(prices []float64, fastPeriod, slowPeriod, signalPeriod int) (dif, dea, histogram []float64, err error) {
	if fastPeriod <= 0 || slowPeriod <= 0 || signalPeriod <= 0 {
		return nil, nil, nil, fmt.Errorf("MACD periods must be greater than zero")
	}
	if fastPeriod >= slowPeriod {
		return nil, nil, nil, fmt.Errorf("MACD fast period must be less than slow period")
	}
	if len(prices) < slowPeriod+signalPeriod-1 {
		return nil, nil, nil, fmt.Errorf("insufficient data for MACD periods %d/%d/%d: got %d values", fastPeriod, slowPeriod, signalPeriod, len(prices))
	}

	fastEMA, err := CalculateExponentialMovingAverage(prices, fastPeriod)
	if err != nil {
		return nil, nil, nil, err
	}
	slowEMA, err := CalculateExponentialMovingAverage(prices, slowPeriod)
	if err != nil {
		return nil, nil, nil, err
	}

	difValues := make([]float64, len(slowEMA))
	for i := range slowEMA {
		difValues[i] = fastEMA[i] - slowEMA[i]
	}
	dea, err = CalculateExponentialMovingAverage(difValues, signalPeriod)
	if err != nil {
		return nil, nil, nil, err
	}

	dif = difValues[:len(dea)]
	histogram = make([]float64, len(dea))
	for i := range dea {
		histogram[i] = dif[i] - dea[i]
	}
	return dif, dea, histogram, nil
}

// 布林带(boll) 中轨线（MB，通常为移动平均线）、上轨线（UP，通常为中轨线加上一定倍数的标准差）和下轨线（DN，通常为中轨线减去相同倍数的标准差）
// 默认 period = 21, stdDevMultiplier = 2, 返回的切片长度 len(clonePrices)-period+1
// bbw = (UP - DN) / MB * 100
func CalculateBollingerBands(clonePrices []float64, period int, stdDevMultiplier float64) (up, mb, dn []float64, err error) {
	if err := validatePeriod(len(clonePrices), period); err != nil {
		return nil, nil, nil, err
	}
	if stdDevMultiplier < 0 {
		return nil, nil, nil, fmt.Errorf("standard deviation multiplier must not be negative")
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
	if period <= 0 {
		return nil, fmt.Errorf("period must be greater than zero")
	}
	if len(prices) <= period {
		return nil, fmt.Errorf("insufficient data for period %d: got %d values", period, len(prices))
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

	// Initialize the first RSI at the candle that closes the seed window.
	rsiValues := make([]float64, len(prices)-period)
	rsiValues[0] = calculateRSIValue(avgGain, avgLoss)

	// Apply Wilder smoothing once for every change after the seed window.
	for i := period; i < len(gains); i++ {
		newGain := gains[i]
		newLoss := losses[i]

		avgGain = ((avgGain * float64(period-1)) + newGain) / float64(period)
		avgLoss = ((avgLoss * float64(period-1)) + newLoss) / float64(period)

		rsiValues[i-period+1] = calculateRSIValue(avgGain, avgLoss)
	}

	return utils.ReverseArray(rsiValues), nil
}

func calculateRSIValue(avgGain, avgLoss float64) float64 {
	if avgGain == 0 && avgLoss == 0 {
		return 50
	}
	if avgLoss == 0 {
		return 100
	}
	if avgGain == 0 {
		return 0
	}
	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

// CalculateROC returns percentage price changes ordered from newest to oldest.
func CalculateROC(close []float64, period int) ([]float64, error) {
	if len(close) == 0 {
		return nil, fmt.Errorf("close slice must not be empty")
	}
	if period <= 0 {
		return nil, fmt.Errorf("ROC period must be greater than zero")
	}
	if len(close) <= period {
		return nil, fmt.Errorf("insufficient data for ROC period %d: got %d values", period, len(close))
	}

	roc := make([]float64, len(close)-period)
	for i := range roc {
		previousClose := close[i+period]
		if previousClose == 0 {
			return nil, fmt.Errorf("ROC reference close must not be zero at index %d", i+period)
		}
		roc[i] = (close[i] - previousClose) / previousClose * 100
	}
	return roc, nil
}

// CalculateMFI returns money flow index values ordered from newest to oldest.
func CalculateMFI(high, low, close, amount []float64, period int) ([]float64, error) {
	if len(high) == 0 || len(high) != len(low) || len(high) != len(close) || len(high) != len(amount) {
		return nil, fmt.Errorf("high, low, close, and amount slices must be non-empty and have the same length")
	}
	if period <= 0 {
		return nil, fmt.Errorf("MFI period must be greater than zero")
	}
	if len(high) <= period {
		return nil, fmt.Errorf("insufficient data for MFI period %d: got %d values", period, len(high))
	}

	typicalPrice := make([]float64, len(high))
	for i := range high {
		if amount[i] < 0 {
			return nil, fmt.Errorf("MFI amount must not be negative")
		}
		typicalPrice[i] = (high[i] + low[i] + close[i]) / 3
	}

	positiveFlow := make([]float64, len(high)-1)
	negativeFlow := make([]float64, len(high)-1)
	for i := 0; i < len(positiveFlow); i++ {
		if typicalPrice[i] > typicalPrice[i+1] {
			positiveFlow[i] = amount[i]
		} else if typicalPrice[i] < typicalPrice[i+1] {
			negativeFlow[i] = amount[i]
		}
	}

	positiveSum := Sum(positiveFlow[:period])
	negativeSum := Sum(negativeFlow[:period])
	mfi := make([]float64, len(high)-period)
	mfi[0] = calculateMFIValue(positiveSum, negativeSum)
	for i := period; i < len(positiveFlow); i++ {
		positiveSum += positiveFlow[i] - positiveFlow[i-period]
		negativeSum += negativeFlow[i] - negativeFlow[i-period]
		mfi[i-period+1] = calculateMFIValue(positiveSum, negativeSum)
	}
	return mfi, nil
}

func calculateMFIValue(positiveFlow, negativeFlow float64) float64 {
	if positiveFlow == 0 && negativeFlow == 0 {
		return 50
	}
	if negativeFlow == 0 {
		return 100
	}
	if positiveFlow == 0 {
		return 0
	}
	moneyFlowRatio := positiveFlow / negativeFlow
	return 100 - (100 / (1 + moneyFlowRatio))
}

// CalculateOBV returns on-balance volume values ordered from newest to oldest.
func CalculateOBV(close, amount []float64) ([]float64, error) {
	if len(close) == 0 || len(close) != len(amount) {
		return nil, fmt.Errorf("close and amount slices must be non-empty and have the same length")
	}
	for i := range amount {
		if amount[i] < 0 {
			return nil, fmt.Errorf("OBV amount must not be negative at index %d", i)
		}
	}

	chronologicalClose := utils.ReverseArray(close)
	chronologicalAmount := utils.ReverseArray(amount)
	obv := make([]float64, len(close))
	for i := 1; i < len(obv); i++ {
		obv[i] = obv[i-1]
		if chronologicalClose[i] > chronologicalClose[i-1] {
			obv[i] += chronologicalAmount[i]
		} else if chronologicalClose[i] < chronologicalClose[i-1] {
			obv[i] -= chronologicalAmount[i]
		}
	}
	return utils.ReverseArray(obv), nil
}

// CalculateCCI returns commodity channel index values ordered from newest to oldest.
func CalculateCCI(high, low, close []float64, period int) ([]float64, error) {
	if len(high) == 0 || len(high) != len(low) || len(high) != len(close) {
		return nil, fmt.Errorf("high, low, and close slices must be non-empty and have the same length")
	}
	if period <= 0 {
		return nil, fmt.Errorf("CCI period must be greater than zero")
	}
	if len(high) < period {
		return nil, fmt.Errorf("insufficient data for CCI period %d: got %d values", period, len(high))
	}

	typicalPrice := make([]float64, len(high))
	for i := range high {
		if high[i] < low[i] {
			return nil, fmt.Errorf("CCI high must not be lower than low at index %d", i)
		}
		typicalPrice[i] = (high[i] + low[i] + close[i]) / 3
	}

	cci := make([]float64, len(high)-period+1)
	for start := 0; start <= len(high)-period; start++ {
		window := typicalPrice[start : start+period]
		average := calculateAverage(window)
		meanDeviation := 0.0
		for _, value := range window {
			meanDeviation += math.Abs(value - average)
		}
		meanDeviation /= float64(period)
		if meanDeviation == 0 {
			cci[start] = 0
			continue
		}
		cci[start] = (typicalPrice[start] - average) / (0.015 * meanDeviation)
	}
	return cci, nil
}

// Kdj returns K, D, and J values ordered from newest to oldest.
func Kdj(high, low, close []float64, period, kPeriod, dPeriod int) (kValues, dValues, jValues []float64, err error) {
	if len(high) == 0 || len(high) != len(low) || len(high) != len(close) {
		return nil, nil, nil, fmt.Errorf("high, low, and close slices must be non-empty and have the same length")
	}
	if period <= 0 || kPeriod <= 0 || dPeriod <= 0 {
		return nil, nil, nil, fmt.Errorf("KDJ periods must be greater than zero")
	}
	if len(high) < period {
		return nil, nil, nil, fmt.Errorf("insufficient data for KDJ period %d: got %d values", period, len(high))
	}

	high = utils.ReverseArray(high)
	low = utils.ReverseArray(low)
	close = utils.ReverseArray(close)
	resultLength := len(high) - period + 1
	kValues = make([]float64, resultLength)
	dValues = make([]float64, resultLength)
	jValues = make([]float64, resultLength)
	k, d := 50.0, 50.0
	for i := period - 1; i < len(high); i++ {
		highestHigh := high[i-period+1]
		lowestLow := low[i-period+1]
		for j := i - period + 2; j <= i; j++ {
			if high[j] > highestHigh {
				highestHigh = high[j]
			}
			if low[j] < lowestLow {
				lowestLow = low[j]
			}
		}
		rsv := 50.0
		if highestHigh != lowestLow {
			rsv = (close[i] - lowestLow) / (highestHigh - lowestLow) * 100
		}
		k = (float64(kPeriod-1)*k + rsv) / float64(kPeriod)
		d = (float64(dPeriod-1)*d + k) / float64(dPeriod)
		index := i - period + 1
		kValues[index] = k
		dValues[index] = d
		jValues[index] = 3*k - 2*d
	}
	return utils.ReverseArray(kValues), utils.ReverseArray(dValues), utils.ReverseArray(jValues), nil
}

// sum 计算浮点数切片的总和
func Sum(numbers []float64) float64 {
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
func KdjSimple(ma1 []float64, ma2 []float64, num int) bool {
	if ma1 == nil || ma2 == nil {
		return false
	}
	if len(ma1) < num || len(ma2) < num {
		return false
	}
	if ma1[0] < ma2[0] {
		// 最新数据的必须是短线在上
		return false
	}
	k := 0
	for i := 1; i < num; i++ {
		if ma1[i] < ma2[i] {
			// 发生过短线在下，说明产生过死叉
			k = i
			break
		}
	}
	// 之后数据不能再重新产生交叉
	for i := k; i < num; i++ {
		if ma1[i] > ma2[i] {
			return false
		}
	}
	return k > 0
}

// 计算真实范围 TR= max(High−Low,∣High−PreviousClose∣,∣Low−PreviousClose∣) 数据时间由新到旧
func calculateTrueRange(high, low, close []float64) ([]float64, error) {
	if len(high) == 0 {
		return nil, fmt.Errorf("price slices must not be empty")
	}
	if len(high) != len(low) || len(high) != len(close) {
		return nil, fmt.Errorf("high, low, and close slices must have the same length")
	}
	tr := make([]float64, len(high))

	for i := 0; i < len(high)-1; i++ {
		hl := high[i] - low[i]
		hpc := math.Abs(high[i] - close[i+1])
		lpc := math.Abs(low[i] - close[i+1])
		tr[i] = math.Max(hl, math.Max(hpc, lpc))
	}
	tr[len(high)-1] = high[len(high)-1] - low[len(high)-1]

	return tr, nil
}

// 平均真实波幅
func CalculateAtr(high, low, close []float64, period int) ([]float64, error) {
	tr, err := calculateTrueRange(high, low, close)
	if err != nil {
		return nil, err
	}
	return calculateWilderMovingAverage(tr, period)
}

// CalculateSupertrend returns the active trend line and direction ordered from newest to oldest.
func CalculateSupertrend(high, low, close []float64, period int, multiplier float64) (data, trend []float64, err error) {
	if len(high) == 0 || len(high) != len(low) || len(high) != len(close) {
		return nil, nil, fmt.Errorf("high, low, and close slices must be non-empty and have the same length")
	}
	if period <= 0 {
		return nil, nil, fmt.Errorf("Supertrend period must be greater than zero")
	}
	if multiplier <= 0 {
		return nil, nil, fmt.Errorf("Supertrend multiplier must be greater than zero")
	}
	for i := range high {
		if high[i] < low[i] {
			return nil, nil, fmt.Errorf("Supertrend high must not be lower than low at index %d", i)
		}
	}

	atrNewestFirst, err := CalculateAtr(high, low, close, period)
	if err != nil {
		return nil, nil, err
	}
	high = utils.ReverseArray(high)
	low = utils.ReverseArray(low)
	close = utils.ReverseArray(close)
	atr := utils.ReverseArray(atrNewestFirst)

	resultLength := len(atr)
	finalUpper := make([]float64, resultLength)
	finalLower := make([]float64, resultLength)
	data = make([]float64, resultLength)
	trend = make([]float64, resultLength)
	for resultIndex := 0; resultIndex < resultLength; resultIndex++ {
		priceIndex := period - 1 + resultIndex
		midpoint := (high[priceIndex] + low[priceIndex]) / 2
		basicUpper := midpoint + multiplier*atr[resultIndex]
		basicLower := midpoint - multiplier*atr[resultIndex]
		if resultIndex == 0 {
			finalUpper[resultIndex] = basicUpper
			finalLower[resultIndex] = basicLower
			if close[priceIndex] >= midpoint {
				trend[resultIndex] = 1
				data[resultIndex] = finalLower[resultIndex]
			} else {
				trend[resultIndex] = -1
				data[resultIndex] = finalUpper[resultIndex]
			}
			continue
		}

		previousIndex := resultIndex - 1
		previousClose := close[priceIndex-1]
		if basicUpper < finalUpper[previousIndex] || previousClose > finalUpper[previousIndex] {
			finalUpper[resultIndex] = basicUpper
		} else {
			finalUpper[resultIndex] = finalUpper[previousIndex]
		}
		if basicLower > finalLower[previousIndex] || previousClose < finalLower[previousIndex] {
			finalLower[resultIndex] = basicLower
		} else {
			finalLower[resultIndex] = finalLower[previousIndex]
		}

		if trend[previousIndex] < 0 {
			if close[priceIndex] > finalUpper[resultIndex] {
				trend[resultIndex] = 1
				data[resultIndex] = finalLower[resultIndex]
			} else {
				trend[resultIndex] = -1
				data[resultIndex] = finalUpper[resultIndex]
			}
		} else if close[priceIndex] < finalLower[resultIndex] {
			trend[resultIndex] = -1
			data[resultIndex] = finalUpper[resultIndex]
		} else {
			trend[resultIndex] = 1
			data[resultIndex] = finalLower[resultIndex]
		}
	}

	return utils.ReverseArray(data), utils.ReverseArray(trend), nil
}

// CalculateADX returns ADX, +DI, and -DI values ordered from newest to oldest.
func CalculateADX(high, low, close []float64, period int) (adx, plusDI, minusDI []float64, err error) {
	tr, err := calculateTrueRange(high, low, close)
	if err != nil {
		return nil, nil, nil, err
	}
	if period <= 0 {
		return nil, nil, nil, fmt.Errorf("ADX period must be greater than zero")
	}
	if len(high) < period*2 {
		return nil, nil, nil, fmt.Errorf("insufficient data for ADX period %d: got %d values", period, len(high))
	}

	directionalCount := len(high) - 1
	plusDM := make([]float64, directionalCount)
	minusDM := make([]float64, directionalCount)
	for i := 0; i < directionalCount; i++ {
		upMove := high[i] - high[i+1]
		downMove := low[i+1] - low[i]
		if upMove > downMove && upMove > 0 {
			plusDM[i] = upMove
		} else if downMove > upMove && downMove > 0 {
			minusDM[i] = downMove
		}
	}

	smoothedTR, err := calculateWilderMovingAverage(tr[:directionalCount], period)
	if err != nil {
		return nil, nil, nil, err
	}
	smoothedPlusDM, err := calculateWilderMovingAverage(plusDM, period)
	if err != nil {
		return nil, nil, nil, err
	}
	smoothedMinusDM, err := calculateWilderMovingAverage(minusDM, period)
	if err != nil {
		return nil, nil, nil, err
	}

	plusDIValues := make([]float64, len(smoothedTR))
	minusDIValues := make([]float64, len(smoothedTR))
	dx := make([]float64, len(smoothedTR))
	for i := range smoothedTR {
		if smoothedTR[i] == 0 {
			continue
		}
		plusDIValues[i] = 100 * smoothedPlusDM[i] / smoothedTR[i]
		minusDIValues[i] = 100 * smoothedMinusDM[i] / smoothedTR[i]
		directionalSum := plusDIValues[i] + minusDIValues[i]
		if directionalSum > 0 {
			dx[i] = 100 * math.Abs(plusDIValues[i]-minusDIValues[i]) / directionalSum
		}
	}

	adx, err = calculateWilderMovingAverage(dx, period)
	if err != nil {
		return nil, nil, nil, err
	}
	return adx, plusDIValues[:len(adx)], minusDIValues[:len(adx)], nil
}

// 肯纳特通道
func CalculateKeltnerChannels(high, low, close []float64, period int, multiplier float64) (upper, ma, lower []float64, err error) {
	if multiplier < 0 {
		return nil, nil, nil, fmt.Errorf("multiplier must not be negative")
	}
	ma, err = CalculateExponentialMovingAverage(close, period)
	if err != nil {
		return nil, nil, nil, err
	}
	atr, err := CalculateAtr(high, low, close, period)
	if err != nil {
		return nil, nil, nil, err
	}
	if len(ma) != len(atr) {
		return nil, nil, nil, fmt.Errorf("EMA and ATR lengths do not match")
	}

	upper = make([]float64, len(ma))
	lower = make([]float64, len(ma))

	for i := 0; i < len(ma); i++ {
		upper[i] = ma[i] + multiplier*atr[i]
		lower[i] = ma[i] - multiplier*atr[i]
	}

	return upper, ma, lower, nil
}

// CalculateDonchianChannels returns upper, middle, and lower bands ordered from newest to oldest.
func CalculateDonchianChannels(high, low []float64, period int) (upper, middle, lower []float64, err error) {
	if len(high) == 0 || len(high) != len(low) {
		return nil, nil, nil, fmt.Errorf("high and low slices must be non-empty and have the same length")
	}
	if err := validatePeriod(len(high), period); err != nil {
		return nil, nil, nil, err
	}
	for i := range high {
		if high[i] < low[i] {
			return nil, nil, nil, fmt.Errorf("Donchian high must not be lower than low at index %d", i)
		}
	}

	resultLength := len(high) - period + 1
	upper = make([]float64, resultLength)
	middle = make([]float64, resultLength)
	lower = make([]float64, resultLength)
	for start := 0; start < resultLength; start++ {
		upper[start] = high[start]
		lower[start] = low[start]
		for i := start + 1; i < start+period; i++ {
			if high[i] > upper[start] {
				upper[start] = high[i]
			}
			if low[i] < lower[start] {
				lower[start] = low[i]
			}
		}
		middle[start] = (upper[start] + lower[start]) / 2
	}
	return upper, middle, lower, nil
}

func calculateWilderMovingAverage(values []float64, period int) ([]float64, error) {
	if err := validatePeriod(len(values), period); err != nil {
		return nil, err
	}

	values = utils.ReverseArray(values)
	result := make([]float64, len(values)-period+1)
	result[0] = calculateAverage(values[:period])
	for i := period; i < len(values); i++ {
		result[i-period+1] = (result[i-period]*float64(period-1) + values[i]) / float64(period)
	}
	return utils.ReverseArray(result), nil
}

func validatePeriod(dataLength, period int) error {
	if period <= 0 {
		return fmt.Errorf("period must be greater than zero")
	}
	if dataLength < period {
		return fmt.Errorf("insufficient data for period %d: got %d values", period, dataLength)
	}
	return nil
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
	closesInsideFirstBody := second.Close < first.Open && second.Close > first.Close*0.5+first.Open*0.5

	return isFirstBullish && isSecondBearish && opensAboveFirstClose && closesInsideFirstBody
}
