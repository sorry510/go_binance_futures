package technology

type IndicatorConfig struct {
	Name          string  `json:"name"` // 指标名称
	Enable		  bool    `json:"enable"` // 是否启用
	KlineInterval string  `json:"kline_interval"` // K线周期
	Period        int     `json:"period"` // 周期
	Multiplier    float64 `json:"multiplier,omitempty"` // 可选字段
	StdDevMultiplier float64 `json:"std_dev_multiplier,omitempty"` // 可选字段
}

// 顶层技术指标配置结构
type TechnologyConfig struct {
	MA       []IndicatorConfig     `json:"ma"`   // 简单移动平均线
	EMA      []IndicatorConfig     `json:"ema"`  // 指数移动平均线
	RSI      []IndicatorConfig     `json:"rsi"`  // 相对强弱指数
	KC       []IndicatorConfig     `json:"kc"`   // 肯特纳通道
	BOLL     []IndicatorConfig     `json:"boll"` // 布林带
}

type StrategyConfig [] struct {
	Name   string `json:"name"`   // 策略名称
	Enable bool   `json:"enable"` // 是否启用 
	Code   string `json:"code"`   // 自定义规则的表达式
	Type   string `json:"type"`   // long or short
}