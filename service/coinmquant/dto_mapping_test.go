package coinmquant

import (
	"reflect"
	"strings"
	"testing"

	"go_binance_futures/models"
)

// CM-3 rereview O2: these are NOT registered ORM models, but Beego's
// Raw.QueryRows needs column tags for the exact snake_case SQL projection.
func TestCM3RawSQLDTOColumnMapping(t *testing.T) {
	dtos := []any{
		models.CoinMManagedPosition{},
		models.CoinMManagedOrder{},
		models.CoinMTradeFill{},
		models.CoinMStrategyTemplate{},
		models.CoinMSymbolTradingConfig{},
	}
	for _, dto := range dtos {
		typ := reflect.TypeOf(dto)
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			column := strings.Split(field.Tag.Get("json"), ",")[0]
			expected := "column(" + column + ")"
			if column == "" || field.Tag.Get("orm") != expected {
				t.Errorf("%s.%s: orm column tag = %q; want %q",
					typ.Name(), field.Name, field.Tag.Get("orm"), expected)
			}
		}
	}
}
