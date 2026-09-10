package controllers

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var deprecatedStrategyMarketGlobal = regexp.MustCompile(`(^|[^A-Za-z0-9_])(BasicTrend|BTCUSDT|ETHUSDT|SOLUSDT|BNBUSDT)([^A-Za-z0-9_]|$)`)

func TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals(t *testing.T) {
	files, err := filepath.Glob("../strategy_templates/*.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no static strategy templates found")
	}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := parseStrategyTemplateImport(data)
		if err != nil {
			t.Fatalf("%s is not a valid executable strategy template: %v", filepath.Base(file), err)
		}
		if deprecatedStrategyMarketGlobal.MatchString(parsed.Strategy) {
			t.Fatalf("%s still references a removed market env global", filepath.Base(file))
		}
		if strings.TrimSpace(parsed.Strategy) == "" {
			t.Fatalf("%s has empty strategy after cleanup", filepath.Base(file))
		}
	}
}
