package strategy

import (
	"container/list"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

// Expr programs are immutable. Only the compilation is shared; each expr.Run
// creates its own VM, so evaluating concurrent symbols never shares state.
// The key includes the *actual* environment types, not just the rule/code:
// Futures, Spot and COIN-M have different Position/Positions layouts.
const maxCompiledRules = 256

type compiledRuleKey struct {
	code           string
	environment    string
	asBool         bool
	allowUndefined bool
}

type compiledRuleEntry struct {
	key     compiledRuleKey
	program *vm.Program
}

type compiledRuleCache struct {
	mu      sync.Mutex
	entries map[compiledRuleKey]*list.Element
	recent  list.List
}

var sharedCompiledRules = compiledRuleCache{
	entries: make(map[compiledRuleKey]*list.Element),
}

// CompileRule caches successful Expr compilation using exact source text,
// environment variable names/types, and compiler options. No cached result
// ever contains prices, indicator values, positions or exchange account data.
// A bounded LRU avoids unbounded memory use during strategy research.
func CompileRule(code string, environment map[string]interface{}, asBool, allowUndefined bool) (*vm.Program, error) {
	if strings.TrimSpace(code) == "" {
		return nil, fmt.Errorf("empty strategy expression")
	}
	key := compiledRuleKey{
		code:           code,
		environment:    ruleEnvironmentSignature(environment),
		asBool:         asBool,
		allowUndefined: allowUndefined,
	}
	sharedCompiledRules.mu.Lock()
	defer sharedCompiledRules.mu.Unlock()
	if entry := sharedCompiledRules.entries[key]; entry != nil {
		sharedCompiledRules.recent.MoveToFront(entry)
		return entry.Value.(compiledRuleEntry).program, nil
	}

	options := []expr.Option{expr.Env(environment)}
	if asBool {
		options = append(options, expr.AsBool())
	}
	if allowUndefined {
		options = append(options, expr.AllowUndefinedVariables())
	}
	program, err := expr.Compile(code, options...)
	if err != nil {
		// Invalid expressions are not cached: corrected code must compile.
		return nil, err
	}
	entry := sharedCompiledRules.recent.PushFront(compiledRuleEntry{key: key, program: program})
	sharedCompiledRules.entries[key] = entry
	if sharedCompiledRules.recent.Len() > maxCompiledRules {
		oldest := sharedCompiledRules.recent.Back()
		delete(sharedCompiledRules.entries, oldest.Value.(compiledRuleEntry).key)
		sharedCompiledRules.recent.Remove(oldest)
	}
	return program, nil
}

func ruleEnvironmentSignature(environment map[string]interface{}) string {
	names := make([]string, 0, len(environment))
	for name := range environment {
		names = append(names, name)
	}
	sort.Strings(names)
	var out strings.Builder
	for _, name := range names {
		out.WriteString(fmt.Sprintf("%d:", len(name)))
		out.WriteString(name)
		typ := reflect.TypeOf(environment[name])
		// An untyped nil may be compiled differently from a typed nil.
		signature := "<nil>"
		if typ != nil {
			signature = typ.String()
			if typ.Name() != "" {
				signature += "@" + typ.PkgPath()
			}
		}
		out.WriteString(fmt.Sprintf("%d:", len(signature)))
		out.WriteString(signature)
		out.WriteByte(';')
	}
	return out.String()
}
