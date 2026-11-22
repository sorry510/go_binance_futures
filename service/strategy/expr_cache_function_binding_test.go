package strategy

import (
	"github.com/expr-lang/expr"
	"testing"
)

func TestCompileRuleUsesCurrentEnvironmentFunctions(t *testing.T) {
	source := "Gate(NowPrice)"
	first := map[string]interface{}{"Gate": func(v float64) bool { return v > 100 }, "NowPrice": 120.0}
	second := map[string]interface{}{"Gate": func(v float64) bool { return v < 100 }, "NowPrice": 120.0}
	compiled, err := CompileRule(source, first, true, false)
	if err != nil {
		t.Fatal(err)
	}
	other, err := CompileRule(source, second, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if compiled != other {
		t.Fatal("same type cannot reuse?")
	}
	a, err := expr.Run(compiled, first)
	if err != nil {
		t.Fatal(err)
	}
	b, err := expr.Run(other, second)
	if err != nil {
		t.Fatal(err)
	}
	if a != true || b != false {
		t.Fatalf("cached expr captured an old function: first=%v second=%v", a, b)
	}
}
