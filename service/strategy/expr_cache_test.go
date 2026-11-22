package strategy

import (
	"fmt"
	"sync"
	"testing"

	"github.com/expr-lang/expr"
)

type cacheFuturesPosition struct{ EntryPrice float64 }
type cacheSpotPosition struct{ EntryPrice float64 }

func TestCompileRuleCacheReuseAndIsolation(t *testing.T) {
	futures := map[string]interface{}{"NowPrice": 120.0, "Position": cacheFuturesPosition{EntryPrice: 100}}
	program, err := CompileRule("NowPrice > Position.EntryPrice", futures, true, false)
	if err != nil {
		t.Fatal(err)
	}
	again, err := CompileRule("NowPrice > Position.EntryPrice",
		map[string]interface{}{"Position": cacheFuturesPosition{EntryPrice: 115}, "NowPrice": 123.0}, true, false)
	if err != nil || program != again {
		t.Fatalf("same names/types must reuse compiled program, got %p %p err=%v", program, again, err)
	}
	out, err := expr.Run(again, map[string]interface{}{
		"NowPrice": 80.0, "Position": cacheFuturesPosition{EntryPrice: 100},
	})
	if err != nil || out != false {
		t.Fatalf("cache captured an earlier price or position: %v %v", out, err)
	}
	spot, err := CompileRule("NowPrice > Position.EntryPrice",
		map[string]interface{}{"NowPrice": 120.0, "Position": cacheSpotPosition{EntryPrice: 100}}, true, false)
	if err != nil || spot == program {
		t.Fatal("Spot Position type must have a distinct program", err)
	}
	changed, err := CompileRule("NowPrice >= Position.EntryPrice", futures, true, false)
	if err != nil || changed == program {
		t.Fatal("a code edit must not reuse stale expression", err)
	}
	notTyped, err := CompileRule("NowPrice > Position.EntryPrice", futures, false, false)
	if err != nil || notTyped == program {
		t.Fatal("AsBool compiler option must be part of cache identity", err)
	}
}

func TestCompileRuleCacheOptionAndErrorIsolation(t *testing.T) {
	env := map[string]interface{}{"NowPrice": 123.0}
	if _, err := CompileRule("UndeclaredValue > 0", env, true, false); err == nil {
		t.Fatal("strict compilation accepted an unknown identifier")
	}
	weak, err := CompileRule("UndeclaredValue > 0", env, true, true)
	if err != nil {
		t.Fatal(err)
	}
	weakAgain, err := CompileRule("UndeclaredValue > 0", env, true, true)
	if err != nil || weak != weakAgain {
		t.Fatal("AllowUndefinedVariables mode did not reuse correctly", err)
	}
	if _, err := CompileRule("NowPrice + 1", env, true, false); err == nil {
		t.Fatal("AsBool must reject a numeric final result")
	}
	if _, err := CompileRule("NowPrice + 1", env, false, false); err != nil {
		t.Fatal("untyped legacy path should permit numeric result", err)
	}
	if _, err := CompileRule("NowPrice >", env, false, false); err == nil {
		t.Fatal("syntax error must fail")
	}
	if _, err := CompileRule("NowPrice > 0", env, true, false); err != nil {
		t.Fatal("fixed expression should compile", err)
	}
}

func TestCompileRuleCacheBoundedSize(t *testing.T) {
	env := map[string]interface{}{"NowPrice": 100.0}
	for i := 0; i < maxCompiledRules+40; i++ {
		if _, err := CompileRule(fmt.Sprintf("NowPrice > %d", i), env, true, false); err != nil {
			t.Fatal(err)
		}
	}
	sharedCompiledRules.mu.Lock()
	size := len(sharedCompiledRules.entries)
	recent := sharedCompiledRules.recent.Len()
	sharedCompiledRules.mu.Unlock()
	if size != maxCompiledRules || recent != maxCompiledRules {
		t.Fatalf("cache leaked programs: map=%d LRU=%d limit=%d", size, recent, maxCompiledRules)
	}
}

func TestCompileRuleConcurrentUse(t *testing.T) {
	const workers = 12
	var wg sync.WaitGroup
	failures := make(chan error, workers)
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < 90; i++ {
				env := map[string]interface{}{"NowPrice": float64(worker + i)}
				code := fmt.Sprintf("NowPrice > %d", i%8)
				program, err := CompileRule(code, env, true, false)
				if err != nil {
					failures <- err
					return
				}
				value, err := expr.Run(program, env)
				if err != nil {
					failures <- err
					return
				}
				want := float64(worker+i) > float64(i%8)
				if value != want {
					failures <- fmt.Errorf("incorrect concurrent result: got %v want %v", value, want)
					return
				}
			}
		}(worker)
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
}

func BenchmarkCompileRuleHotCache(b *testing.B) {
	env := map[string]interface{}{"NowPrice": 101.0, "ROI": 3.0}
	code := "NowPrice > 100 && ROI > 2"
	if _, err := CompileRule(code, env, true, false); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := CompileRule(code, env, true, false); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompileRuleUncached(b *testing.B) {
	env := map[string]interface{}{"NowPrice": 101.0, "ROI": 3.0}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := expr.Compile("NowPrice > 100 && ROI > 2", expr.Env(env), expr.AsBool()); err != nil {
			b.Fatal(err)
		}
	}
}
