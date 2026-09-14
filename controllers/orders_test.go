package controllers

import (
	"reflect"
	"strings"
	"testing"
)

func TestOrderSearchParamsWhereClauseUsesCloseOrderLink(t *testing.T) {
	tests := []struct {
		name     string
		params   orderSearchParams
		wantPart string
		wantArgs []interface{}
		wantHas  bool
		wantErr  bool
	}{
		{name: "no filters", params: orderSearchParams{}, wantHas: false},
		{name: "open", params: orderSearchParams{Type: "open"}, wantPart: "t.closeOrderId = 0", wantHas: true},
		{name: "closed", params: orderSearchParams{Type: "close"}, wantPart: "t.closeOrderId > 0", wantHas: true},
		{name: "symbol and side", params: orderSearchParams{Symbol: "BTC", PositionSide: "LONG"}, wantPart: "t.symbol LIKE ? AND t.positionSide = ?", wantArgs: []interface{}{"%BTC%", "LONG"}, wantHas: true},
		{name: "invalid type", params: orderSearchParams{Type: "bad"}, wantErr: true},
		{name: "invalid side", params: orderSearchParams{PositionSide: "BOTH"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			where, args, has, err := tt.params.whereClause("t")
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if has != tt.wantHas {
				t.Fatalf("has=%v want=%v", has, tt.wantHas)
			}
			if tt.wantPart != "" && !strings.Contains(where, tt.wantPart) {
				t.Fatalf("where=%q missing %q", where, tt.wantPart)
			}
			if tt.wantArgs != nil && !reflect.DeepEqual(args, tt.wantArgs) {
				t.Fatalf("args=%#v want=%#v", args, tt.wantArgs)
			}
		})
	}
}

func TestOrderOpenClauseOnlyFiltersOpenRows(t *testing.T) {
	got := orderOpenClause("t")
	want := "t.side = 'open'"
	if got != want {
		t.Fatalf("clause=%q want=%q", got, want)
	}
}
