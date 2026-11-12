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
	// The committed CI fixture is always validated. Private strategy
	// templates (gitignored) are checked additionally when present, never
	// required to exist in a clean checkout or CI runner.
	roots := []string{filepath.Join("testdata", "strategy_templates")}
	if info, err := os.Stat("../strategy_templates"); err == nil && info.IsDir() {
		roots = append(roots, "../strategy_templates")
	}
	var files []string
	for _, root := range roots {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			// Research archives hold experimental metadata and can be huge.
			if info.IsDir() && filepath.Base(path) == "research" && filepath.Dir(path) == filepath.Clean(root) {
				return filepath.SkipDir
			}
			if !info.IsDir() && strings.EqualFold(filepath.Ext(path), ".json") {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(files) == 0 {
		t.Fatal("no strategy templates found (private templates or tracked testdata fixture)")
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
