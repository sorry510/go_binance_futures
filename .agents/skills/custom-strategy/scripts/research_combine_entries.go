package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
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
	name := flag.String("name", "", "combined strategy name")
	outputPath := flag.String("output", "", "output portable strategy JSON")
	flag.Parse()
	if *basePath == "" || *supplementPath == "" || *name == "" || *outputPath == "" {
		return errors.New("-base, -supplement, -name, and -output are required")
	}
	if *exitSource != "base" && *exitSource != "supplement" {
		return errors.New("-exit-source must be base or supplement")
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
