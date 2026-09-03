package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func Load(path string) (Case, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Case{}, err
	}
	var value Case
	if err := json.Unmarshal(raw, &value); err != nil {
		return Case{}, fmt.Errorf("decode eval case: %w", err)
	}
	if value.Version != CaseVersion {
		return Case{}, fmt.Errorf("unsupported eval case version %q", value.Version)
	}
	if strings.TrimSpace(value.Name) == "" || strings.TrimSpace(value.Skill) == "" {
		return Case{}, fmt.Errorf("eval case requires name and skill")
	}
	value.SourcePath = path
	return value, nil
}

func LoadDir(path string) ([]Case, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	result := make([]Case, 0, len(names))
	for _, name := range names {
		item, err := Load(filepath.Join(path, name))
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (item Case) FixturePath() string {
	if filepath.IsAbs(item.Fixture) {
		return item.Fixture
	}
	if item.SourcePath == "" {
		return item.Fixture
	}
	return filepath.Clean(filepath.Join(filepath.Dir(item.SourcePath), item.Fixture))
}
