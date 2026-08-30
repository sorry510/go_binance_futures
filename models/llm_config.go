package models

type LLMConfig struct {
	ID             int64   `orm:"column(id);auto" json:"id"`
	Name           string  `orm:"column(name);size(128);unique" json:"name"`
	Provider       string  `orm:"column(provider);size(32)" json:"provider"`
	APIURL         string  `orm:"column(api_url);type(text)" json:"api_url"`
	APIKey         string  `orm:"column(api_key);type(text)" json:"-"`
	Model          string  `orm:"column(model);size(256)" json:"model"`
	APIVersion     string  `orm:"column(api_version);size(64)" json:"api_version,omitempty"`
	TimeoutSeconds int     `orm:"column(timeout_seconds);default(60)" json:"timeout_seconds"`
	Temperature    float64 `orm:"column(temperature);digits(6);decimals(3);default(0.2)" json:"temperature"`
	Enabled        int     `orm:"column(enabled);default(0)" json:"enabled"`
	CreatedAt      int64   `orm:"column(created_at)" json:"created_at"`
	UpdatedAt      int64   `orm:"column(updated_at)" json:"updated_at"`
	Deleted        int     `orm:"column(deleted);default(0)" json:"-"`
}

func (*LLMConfig) TableName() string {
	return "llm_configs"
}
