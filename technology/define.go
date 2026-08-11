package technology

type IndicatorConfig struct {
	Name             string  `json:"name"`                         // 指标名称
	Enable           bool    `json:"enable"`                       // 是否启用
	KlineInterval    string  `json:"kline_interval"`               // K线周期
	Period           int     `json:"period"`                       // 周期
	FastPeriod       int     `json:"fast_period,omitempty"`        // MACD fast period
	SlowPeriod       int     `json:"slow_period,omitempty"`        // MACD slow period
	SignalPeriod     int     `json:"signal_period,omitempty"`      // MACD signal period
	KPeriod          int     `json:"k_period,omitempty"`           // KDJ K smoothing period
	DPeriod          int     `json:"d_period,omitempty"`           // KDJ D smoothing period
	Multiplier       float64 `json:"multiplier,omitempty"`         // 可选字段
	StdDevMultiplier float64 `json:"std_dev_multiplier,omitempty"` // 可选字段
}

// 顶层技术指标配置结构
type TechnologyConfig struct {
	MA         []IndicatorConfig `json:"ma"`         // 简单移动平均线
	EMA        []IndicatorConfig `json:"ema"`        // 指数移动平均线
	MACD       []IndicatorConfig `json:"macd"`       // Moving average convergence divergence
	RSI        []IndicatorConfig `json:"rsi"`        // 相对强弱指数
	KC         []IndicatorConfig `json:"kc"`         // 肯特纳通道
	BOLL       []IndicatorConfig `json:"boll"`       // 布林带
	ATR        []IndicatorConfig `json:"atr"`        // 平均真实波幅
	ADX        []IndicatorConfig `json:"adx"`        // Average directional index
	MFI        []IndicatorConfig `json:"mfi"`        // Money flow index
	OBV        []IndicatorConfig `json:"obv"`        // On-balance volume
	CCI        []IndicatorConfig `json:"cci"`        // Commodity channel index
	ROC        []IndicatorConfig `json:"roc"`        // Rate of change
	KDJ        []IndicatorConfig `json:"kdj"`        // Stochastic KDJ oscillator
	Supertrend []IndicatorConfig `json:"supertrend"` // ATR-based trend indicator
	Donchian   []IndicatorConfig `json:"donchian"`   // Donchian channel
}

type StrategyConfig []struct {
	Name   string `json:"name"`   // 策略名称
	Enable bool   `json:"enable"` // 是否启用
	Code   string `json:"code"`   // 自定义规则的表达式
	Type   string `json:"type"`   // long,short,close_long,close_short
}
