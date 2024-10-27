package technology

type IndicatorConfig struct {
	KlineInterval string  `json:"kline_interval"`
	Period        int     `json:"period"`
	Multiplier    float64 `json:"multiplier,omitempty"` // 可选字段
	StdDevMultiplier float64 `json:"std_dev_multiplier,omitempty"` // 可选字段
}

// MA（简单移动平均线）配置
type MAConfig struct {
	MA IndicatorConfig `json:"ma"`
}

// EMA（指数移动平均线）配置
type EMAConfig struct {
	EMA IndicatorConfig `json:"ema"`
}

// RSI（相对强弱指数）配置
type RSIConfig struct {
	RSI IndicatorConfig `json:"rsi"`
}

// KC（肯特纳通道）配置
type KCConfig struct {
	KC IndicatorConfig `json:"kc"`
}

// BOLL（布林带）配置
type BOLLConfig struct {
	BOLL IndicatorConfig `json:"boll"`
}

// 顶层技术指标配置结构
type TechnologyConfig struct {
	MA       []MAConfig     `json:"ma"`
	EMA      []EMAConfig    `json:"ema"`
	RSI      []RSIConfig    `json:"rsi"`
	KC       []KCConfig     `json:"kc"`
	BOLL     []BOLLConfig   `json:"boll"`
}