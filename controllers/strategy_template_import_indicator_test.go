package controllers

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestStrategyTemplateImportRequiresEnabledKlineIndicators(t *testing.T) {
	tests := []struct {
		name       string
		technology map[string]interface{}
		wantError  string
	}{
		{"empty", map[string]interface{}{}, "至少需要一个已启用的技术指标"},
		{"disabled", map[string]interface{}{"ma": []interface{}{map[string]interface{}{"name": "ma_1h_2", "enable": false, "kline_interval": "1h", "period": 2}}}, "至少需要一个已启用的技术指标"},
		{"wrong interval", map[string]interface{}{"ma": []interface{}{map[string]interface{}{"name": "ma_4h_2", "enable": true, "kline_interval": "4h", "period": 2}}}, "引用 kline_1h"},
		{"correct interval", map[string]interface{}{"ma": []interface{}{map[string]interface{}{"name": "ma_1h_2", "enable": true, "kline_interval": "1h", "period": 2}}}, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			payload := map[string]interface{}{
				"name":       "kline coverage test",
				"technology": tc.technology,
				"strategy": []interface{}{map[string]interface{}{
					"name": "entry", "type": "long", "code": "kline_1h.Close[1] > 0",
					"enable": true, "fullScreen": false,
				}},
			}
			data, err := json.Marshal(payload)
			if err != nil {
				t.Fatal(err)
			}
			_, err = parseStrategyTemplateImport(data)
			if tc.wantError == "" {
				if err != nil {
					t.Fatalf("valid indicator rejected: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("error = %v; want %q", err, tc.wantError)
			}
		})
	}
}
