//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/technology"
	"go_binance_futures/types"
)

type artifact struct {
	Name       string                      `json:"name"`
	Technology technology.TechnologyConfig `json:"technology"`
	Strategy   technology.StrategyConfig   `json:"strategy"`
}

type decision struct {
	Side     string  `json:"side"`
	Scenario string  `json:"scenario"`
	ROI      float64 `json:"roi"`
	Expected bool    `json:"expected"`
	Actual   bool    `json:"actual"`
}

func must(ok bool, label string) {
	if !ok {
		panic(label)
	}
}

func read(path string) artifact {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var a artifact
	if err := json.Unmarshal(data, &a); err != nil {
		panic(err)
	}
	if err := line.ValidateTechnologyConfig(a.Technology); err != nil {
		panic(err)
	}
	return a
}

func fill(value float64) []float64 {
	values := make([]float64, 12)
	for i := range values {
		values[i] = value
	}
	return values
}

func environment(side string, leverage int64, atr float64) map[string]any {
	k := line.KLinePrice{High: fill(102), Low: fill(98), Close: fill(100), Open: fill(100), Amount: fill(1), Qps: fill(100), TakerBuyAmount: fill(1), TakerBuyRatio: fill(.5)}
	h := line.KLinePrice{Close: fill(100)}
	adx := line.ADXConfigData{ADX: fill(24), PlusDI: fill(20), MinusDI: fill(20)}
	adx.ADX[1] = 25
	mfi := line.ConfigData{Data: fill(50)}
	st := line.SupertrendConfigData{Trend: fill(1), Data: fill(90)}
	if side == "LONG" {
		k.Close[0], k.Close[1], k.Open[1], k.TakerBuyRatio[1] = 101, 101, 100.5, .55
		h.Close[7], adx.PlusDI[1], mfi.Data[1], mfi.Data[2] = 98, 30, 55, 54
	} else {
		k.Close[0], k.Close[1], k.Open[1], k.TakerBuyRatio[1] = 99, 99, 99.5, .45
		h.Close[1], h.Close[7], adx.MinusDI[1], mfi.Data[1], mfi.Data[2] = 99, 102, 30, 45, 46
		st.Trend, st.Data = fill(-1), fill(110)
	}
	k.Qps[1] = 110
	return map[string]any{
		"NowTime": int64(3600000), "ROI": float64(0), "kline_1h": k, "kline_4h": h,
		"atr_1h_14": line.ConfigData{Data: fill(atr)}, "adx_1h_14": adx, "mfi_1h_14": mfi,
		"donchian_1h_14":     line.ConfigData{High: fill(100), Low: fill(100), Mid: fill(100)},
		"supertrend_4h_10_3": st,
		"Position":           types.FuturesPositionCode{EntryPrice: 100, MarkPrice: 100, Leverage: leverage, Side: side},
	}
}

func scenario(side string, leverage int64, atr float64, mode string) map[string]any {
	e := environment(side, leverage, atr)
	k := e["kline_1h"].(line.KLinePrice)
	k.Close[0], k.Close[1], k.Close[2], k.High[1], k.Low[1] = 100, 100, 100, 100, 100
	k.Open[1] = 100
	adx := e["adx_1h_14"].(line.ADXConfigData)
	adx.PlusDI[1], adx.MinusDI[1] = 20, 20
	if mode == "pullback" {
		if side == "LONG" {
			k.Close[0] -= atr * .3
		} else {
			k.Close[0] += atr * .3
		}
	}
	if mode == "fading" {
		adx.ADX[1] = 23
	}
	if mode == "rejection" {
		if side == "LONG" {
			k.High[1] += atr * .3
		} else {
			k.Low[1] -= atr * .3
		}
	}
	if mode == "di_only" || mode == "di_structure" {
		if side == "LONG" {
			adx.MinusDI[1] = 30
		} else {
			adx.PlusDI[1] = 30
		}
	}
	if mode == "structure_only" || mode == "di_structure" {
		if side == "LONG" {
			k.Close[1] = 99
			k.Low[1] = 99
		} else {
			k.Close[1] = 101
			k.High[1] = 101
		}
		k.Close[0] = k.Close[1]
	}
	if mode == "htf_flip" {
		st := e["supertrend_4h_10_3"].(line.SupertrendConfigData)
		h := e["kline_4h"].(line.KLinePrice)
		if side == "LONG" {
			st.Trend[1], h.Close[1], st.Data[1] = -1, 99, 100
		} else {
			st.Trend[1], h.Close[1], st.Data[1] = 1, 101, 100
		}
		e["supertrend_4h_10_3"], e["kline_4h"] = st, h
	}
	e["kline_1h"], e["adx_1h_14"] = k, adx
	return e
}

func run(program *vm.Program, e map[string]any) bool {
	output, err := expr.Run(program, e)
	if err != nil {
		panic(err)
	}
	value, ok := output.(bool)
	must(ok, "non-boolean expression result")
	return value
}

func main() {
	must(len(os.Args) == 3, "usage: verify.go PARENT CANDIDATE")
	parent, candidate := read(os.Args[1]), read(os.Args[2])
	must(parent.Name != candidate.Name, "candidate must have a new name")
	must(reflect.DeepEqual(parent.Technology, candidate.Technology), "technology changed")
	must(len(parent.Strategy) == 4 && len(candidate.Strategy) == 4, "expected four rules")
	checks, signalChecks, dominanceChecks, newEarlier, leverageEightChecks := 0, 0, 0, 0, 0
	matrix := []decision{}
	for i, rule := range candidate.Strategy {
		old := parent.Strategy[i]
		must(rule.Enable && old.Enable && old.Type == rule.Type, "rule type/order/enable changed")
		side := "LONG"
		if rule.Type == "short" || rule.Type == "close_short" {
			side = "SHORT"
		}
		sample := environment(side, 4, 1)
		program, err := expr.Compile(rule.Code, expr.Env(sample), expr.AsBool())
		if err != nil {
			panic(err)
		}
		baseline, err := expr.Compile(old.Code, expr.Env(sample), expr.AsBool())
		if err != nil {
			panic(err)
		}
		if rule.Type == "long" || rule.Type == "short" {
			must(reflect.DeepEqual(old, rule), "opening rule changed")
			must(run(program, sample), "baseline valid entry rejected")
			checks++
			for n := 0; n < 2000; n++ {
				e := environment(side, 4, .25+float64(n%35)/10)
				e["NowTime"] = int64(3600000 + n%180*1000)
				k := e["kline_1h"].(line.KLinePrice)
				k.Qps[1], k.TakerBuyRatio[1] = float64(n%350), float64(n%100)/100
				e["kline_1h"] = k
				must(run(program, e) == run(baseline, e), "entry decisions differ")
				signalChecks++
			}
			continue
		}
		for _, test := range []struct {
			mode string
			roi  float64
			want bool
		}{
			{"neutral", 8, false}, {"neutral", -6, false}, {"neutral", -8.99, false},
			{"neutral", -9, true}, {"pullback", 8, true}, {"pullback", 7.5, false},
			{"fading", 8, false}, {"fading", 10, true}, {"rejection", 10, true},
			{"di_only", -6, false}, {"structure_only", -6, false}, {"di_structure", -6, true},
			{"di_structure", -5.5, false}, {"di_structure", 8, true}, {"htf_flip", -6, true},
			{"htf_flip", 8, true}, {"htf_flip", 0, false}, {"neutral", 25, false},
		} {
			e := scenario(side, 4, 1, test.mode)
			e["ROI"] = test.roi
			actual := run(program, e)
			must(actual == test.want, fmt.Sprintf("%s %s ROI=%g", side, test.mode, test.roi))
			checks++
			matrix = append(matrix, decision{side, test.mode, test.roi, test.want, actual})
		}
		for _, leverage := range []int64{1, 3, 4, 6, 8} {
			for _, atr := range []float64{.25, .5, 1, 1.5, 2.5, 3.5} {
				for _, mode := range []string{"neutral", "pullback", "fading", "rejection", "di_only", "structure_only", "di_structure", "htf_flip"} {
					e := scenario(side, leverage, atr, mode)
					for step := -40; step <= 50; step++ {
						roi := float64(step) / 2
						if roi > -6 && roi < 8 {
							continue
						}
						position := e["Position"].(types.FuturesPositionCode)
						denominator := float64(leverage)*100 - roi
						if side == "SHORT" {
							denominator = float64(leverage)*100 + roi
						}
						must(denominator > 0, "invalid mark denominator")
						position.MarkPrice = position.EntryPrice * float64(leverage) * 100 / denominator
						position.Amount = 1
						if side == "SHORT" {
							position.Amount = -1
						}
						position.UnrealizedProfit = (position.MarkPrice - position.EntryPrice) * position.Amount
						e["Position"], e["ROI"] = position, roi
						a, b := run(baseline, e), run(program, e)
						must(!a || b, "candidate suppresses an eligible parent exit at leverage <=8")
						if leverage == 8 {
							must(a == b, "8x behavior changed under the same outer gates")
							leverageEightChecks++
						}
						if b && !a {
							newEarlier++
						}
						dominanceChecks++
					}
				}
			}
		}
	}
	must(newEarlier > 0, "candidate has no observable exit change")
	output := map[string]any{
		"status": "pass", "environment": "real Expr engine and production line/types Go structs; no database or market calls",
		"branch_assertions": checks, "entry_equivalence_cases": signalChecks,
		"eligible_exit_superset_cases_leverage_1_3_4_6_8": dominanceChecks,
		"eligible_exit_equivalence_cases_leverage_8":      leverageEightChecks,
		"candidate_additional_exits_in_synthetic_cases":   newEarlier, "close_decision_matrix": matrix,
		"limits": []string{"synthetic checks do not establish returns", "unchanged entry signals and earlier exits do not prove actual trade-count non-regression", "economic and frequency comparison remains pending"},
	}
	if err := json.NewEncoder(os.Stdout).Encode(output); err != nil {
		panic(err)
	}
}
