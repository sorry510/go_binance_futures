package command

import (
	"strings"
	"testing"
)

func TestAccountIndexesFitMySQL56Utf8mb4KeyLimit(t *testing.T) {
	tests := []struct {
		cols  string
		mysql string
	}{
		{"account_id,symbol,position_side", "`account_id`(16),`symbol`(32),`position_side`(8)"},
		{"account_id,owner,status", "`account_id`(16),`owner`(32),`status`(32)"},
		{"account_id,order_id", "`account_id`(16),`order_id`(32)"},
		{"account_id,symbol,side", "`account_id`(16),`symbol`(32),`side`(8)"},
		{"account_id,side,symbol", "`account_id`(16),`side`(8),`symbol`(32)"},
	}
	for _, tc := range tests {
		t.Run(tc.cols, func(t *testing.T) {
			got, err := accountIndexColumns(tc.cols, true)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.mysql {
				t.Fatalf("mysql index %s, want %s", got, tc.mysql)
			}
			sqlite, err := accountIndexColumns(tc.cols, false)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(sqlite, "(32)") || strings.Contains(sqlite, "(16)") || strings.Contains(sqlite, "(8)") {
				t.Fatalf("SQLite must use full indexes: %s", sqlite)
			}
		})
	}
	if _, err := accountIndexColumns("account_id,bogus", true); err == nil {
		t.Fatal("unexpected unsafe index column allowed")
	}
}
