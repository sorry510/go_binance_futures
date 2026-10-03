package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	strategyservice "go_binance_futures/service/strategy"
)

type combinedPortableStrategy struct {
	Name       string                       `json:"name"`
	Technology map[string][]json.RawMessage `json:"technology"`
	Strategy   []combinedRule               `json:"strategy"`
}

type combinedRule struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Code       string `json:"code"`
	FullScreen bool   `json:"fullScreen"`
	Enable     bool   `json:"enable"`
}

type combinedExitGuards struct {
	LongBase  string `json:"long_base"`
	ShortBase string `json:"short_base"`
}

func main() {
	if err := combineEntries(); err != nil {
		fmt.Fprintln(os.Stderr, "combine entries error:", err)
		os.Exit(1)
	}
}

func combineEntries() error {
	basePath := flag.String("base", "", "first portable strategy JSON")
	supplementPath := flag.String("supplement", "", "second portable strategy JSON")
	exitSource := flag.String("exit-source", "base", "exit rule source: base or supplement")
	guardSpecPath := flag.String("conditional-exit-guards", "", "optional JSON with exclusive long_base and short_base guards")
	entryBoundExits := flag.Bool("entry-bound-exits", false, "route exits by the supplement's exact opening rule hash, not current market regime")
	name := flag.String("name", "", "combined strategy name")
	outputPath := flag.String("output", "", "output portable strategy JSON")
	flag.Parse()
	if *basePath == "" || *supplementPath == "" || *name == "" || *outputPath == "" {
		return errors.New("-base, -supplement, -name, and -output are required")
	}
	if *exitSource != "base" && *exitSource != "supplement" {
		return errors.New("-exit-source must be base or supplement")
	}
	if *entryBoundExits && *guardSpecPath != "" {
		return errors.New("-entry-bound-exits and -conditional-exit-guards are mutually exclusive")
	}
	read := func(path string) (combinedPortableStrategy, error) {
		content, err := os.ReadFile(path)
		if err != nil {
			return combinedPortableStrategy{}, err
		}
		var item combinedPortableStrategy
		if err := json.Unmarshal(content, &item); err != nil {
			return combinedPortableStrategy{}, err
		}
		if item.Name == "" || item.Technology == nil || len(item.Strategy) == 0 {
			return combinedPortableStrategy{}, fmt.Errorf("incomplete strategy %s", path)
		}
		return item, nil
	}
	base, err := read(*basePath)
	if err != nil {
		return err
	}
	supplement, err := read(*supplementPath)
	if err != nil {
		return err
	}
	combined := combinedPortableStrategy{Name: *name, Technology: make(map[string][]json.RawMessage)}
	indicatorNames := make(map[string]string)
	for _, candidate := range []combinedPortableStrategy{base, supplement} {
		for family, items := range candidate.Technology {
			if _, ok := combined.Technology[family]; !ok {
				combined.Technology[family] = []json.RawMessage{}
			}
			for _, raw := range items {
				var descriptor struct {
					Name string `json:"name"`
				}
				if err := json.Unmarshal(raw, &descriptor); err != nil {
					return err
				}
				if descriptor.Name == "" {
					return fmt.Errorf("indicator in %s lacks name", family)
				}
				key := family + ":" + descriptor.Name
				var normalized any
				if err := json.Unmarshal(raw, &normalized); err != nil {
					return err
				}
				canonical, err := json.Marshal(normalized)
				if err != nil {
					return err
				}
				if existing, ok := indicatorNames[key]; ok {
					if existing != string(canonical) {
						return fmt.Errorf("indicator %s has conflicting definitions", key)
					}
					continue
				}
				indicatorNames[key] = string(canonical)
				combined.Technology[family] = append(combined.Technology[family], raw)
			}
		}
	}
	for _, kind := range []string{"long", "short"} {
		for _, candidate := range []combinedPortableStrategy{base, supplement} {
			for _, rule := range candidate.Strategy {
				if rule.Type == kind && rule.Enable {
					combined.Strategy = append(combined.Strategy, rule)
				}
			}
		}
	}
	if *guardSpecPath != "" || *entryBoundExits {
		var guards combinedExitGuards
		if *entryBoundExits {
			for _, item := range []struct {
				kind string
				dst  *string
			}{{"long", &guards.LongBase}, {"short", &guards.ShortBase}} {
				var supplementHash string
				found := 0
				for _, rule := range supplement.Strategy {
					if rule.Enable && rule.Type == item.kind {
						supplementHash = strategyservice.RuleHash(rule.Code)
						found++
					}
				}
				if found != 1 || supplementHash == "" {
					return fmt.Errorf("entry-bound supplement needs exactly one %s rule, found %d", item.kind, found)
				}
				for _, rule := range base.Strategy {
					if rule.Enable && rule.Type == item.kind && strategyservice.RuleHash(rule.Code) == supplementHash {
						return fmt.Errorf("entry-bound %s rule collides with base rule", item.kind)
					}
				}
				*item.dst = "OpenStrategyHash != " + strconv.Quote(supplementHash)
			}
		} else {
			guardsContent, err := os.ReadFile(*guardSpecPath)
			if err != nil {
				return err
			}
			if err := json.Unmarshal(guardsContent, &guards); err != nil {
				return err
			}
		}
		if strings.TrimSpace(guards.LongBase) == "" || strings.TrimSpace(guards.ShortBase) == "" {
			return errors.New("conditional exit guards require long_base and short_base")
		}
		for _, item := range []struct {
			kind  string
			guard string
		}{{"close_long", guards.LongBase}, {"close_short", guards.ShortBase}} {
			for index, candidate := range []combinedPortableStrategy{base, supplement} {
				guard := item.guard
				if index == 1 {
					guard = "!(" + guard + ")"
				}
				found := 0
				for _, rule := range candidate.Strategy {
					if rule.Type != item.kind || !rule.Enable {
						continue
					}
					rule.Code, err = guardCombinedExit(rule.Code, guard)
					if err != nil {
						return fmt.Errorf("guard %s: %w", rule.Name, err)
					}
					combined.Strategy = append(combined.Strategy, rule)
					found++
				}
				if found != 1 {
					return fmt.Errorf("%s source %d has %d enabled rules, expected one", item.kind, index, found)
				}
			}
		}
	} else {
		exits := base
		if *exitSource == "supplement" {
			exits = supplement
		}
		for _, kind := range []string{"close_long", "close_short"} {
			found := 0
			for _, rule := range exits.Strategy {
				if rule.Type == kind && rule.Enable {
					combined.Strategy = append(combined.Strategy, rule)
					found++
				}
			}
			if found != 1 {
				return fmt.Errorf("%s requires exactly one enabled exit rule, found %d", kind, found)
			}
		}
	}
	output, err := json.Marshal(combined)
	if err != nil {
		return err
	}
	content := append(output, '\n')
	if prior, err := os.ReadFile(*outputPath); err == nil {
		if !bytes.Equal(prior, content) {
			return fmt.Errorf("output %s already exists with different content", *outputPath)
		}
		fmt.Println(*outputPath, "already matches")
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(*outputPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(*outputPath, content, 0o644); err != nil {
		return err
	}
	fmt.Println(*outputPath, "created")
	return nil
}

func guardCombinedExit(code, guard string) (string, error) {
	trimmed := strings.TrimSpace(code)
	lastNewline := strings.LastIndex(trimmed, "\n")
	if lastNewline < 0 {
		return "", errors.New("exit rule has no final expression line")
	}
	final := strings.TrimSpace(trimmed[lastNewline+1:])
	if final == "" || strings.HasSuffix(final, ";") {
		return "", errors.New("exit rule final expression is empty or statement-like")
	}
	return trimmed[:lastNewline+1] + "(" + guard + ") && (" + final + ")", nil
}
