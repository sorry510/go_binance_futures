package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type portableStrategy struct {
	Name       string          `json:"name"`
	Technology json.RawMessage `json:"technology"`
	Strategy   []portableRule  `json:"strategy"`
}

type portableRule struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Code       string `json:"code"`
	FullScreen bool   `json:"fullScreen"`
	Enable     bool   `json:"enable"`
}

type replacementSpec struct {
	Type    string `json:"type"`
	Find    string `json:"find"`
	Replace string `json:"replace"`
}

type variantSpec struct {
	Name         string            `json:"name"`
	RuleSuffix   string            `json:"rule_suffix"`
	Replacements []replacementSpec `json:"replacements"`
}

func main() {
	if err := generateVariant(); err != nil {
		fmt.Fprintln(os.Stderr, "generate variant error:", err)
		os.Exit(1)
	}
}

func generateVariant() error {
	basePath := flag.String("base", "", "portable base strategy JSON")
	specPath := flag.String("spec", "", "variant replacement specification JSON")
	outputPath := flag.String("output", "", "output portable strategy JSON")
	flag.Parse()
	if *basePath == "" || *specPath == "" || *outputPath == "" {
		return errors.New("-base, -spec, and -output are required")
	}
	baseBytes, err := os.ReadFile(*basePath)
	if err != nil {
		return err
	}
	var base portableStrategy
	if err := json.Unmarshal(baseBytes, &base); err != nil {
		return err
	}
	specBytes, err := os.ReadFile(*specPath)
	if err != nil {
		return err
	}
	var spec variantSpec
	if err := json.Unmarshal(specBytes, &spec); err != nil {
		return err
	}
	if strings.TrimSpace(spec.Name) == "" || strings.TrimSpace(spec.RuleSuffix) == "" || len(spec.Replacements) == 0 {
		return errors.New("spec requires a name, rule suffix, and at least one replacement")
	}
	base.Name = spec.Name
	for i := range base.Strategy {
		base.Strategy[i].Name += spec.RuleSuffix
	}
	for _, replacement := range spec.Replacements {
		if replacement.Type == "" || replacement.Find == "" || replacement.Replace == "" {
			return errors.New("each replacement requires type, find, and replace")
		}
		matched := 0
		for i := range base.Strategy {
			if base.Strategy[i].Type != replacement.Type || !base.Strategy[i].Enable {
				continue
			}
			if strings.Count(base.Strategy[i].Code, replacement.Find) == 1 {
				base.Strategy[i].Code = strings.Replace(base.Strategy[i].Code, replacement.Find, replacement.Replace, 1)
				matched++
			}
		}
		if matched != 1 {
			return fmt.Errorf("replacement for %s matched %d rules, expected one", replacement.Type, matched)
		}
	}
	output, err := json.Marshal(base)
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
