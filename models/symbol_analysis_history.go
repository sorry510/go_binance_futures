package models

type SymbolAnalysisHistory struct {
	ID              int64   `orm:"column(id)" json:"id"`
	TaskID          string  `orm:"column(task_id);size(64);unique" json:"task_id"`
	Symbol          string  `orm:"column(symbol);size(32);index" json:"symbol"`
	Prompt          string  `orm:"column(prompt);type(text)" json:"prompt"`
	Status          string  `orm:"column(status);size(32);index" json:"status"`
	Direction       string  `orm:"column(direction);size(16);index" json:"direction"`
	Confidence      float64 `orm:"column(confidence);digits(10);decimals(6)" json:"confidence"`
	MarketCondition int     `orm:"column(market_condition)" json:"market_condition"`
	AnalysisPrice   float64 `orm:"column(analysis_price);digits(30);decimals(12)" json:"analysis_price"`
	Summary         string  `orm:"column(summary);type(text)" json:"summary"`
	ResultJSON      string  `orm:"column(result_json);type(text)" json:"-"`
	Error           string  `orm:"column(error);type(text)" json:"error"`
	Provider        string  `orm:"column(provider);size(64)" json:"provider"`
	Model           string  `orm:"column(model);size(128)" json:"model"`
	CreatedAt       int64   `orm:"column(created_at);index" json:"created_at"`
	CompletedAt     int64   `orm:"column(completed_at);index" json:"completed_at"`
}

func (u *SymbolAnalysisHistory) TableName() string {
	return "symbol_analysis_history"
}
