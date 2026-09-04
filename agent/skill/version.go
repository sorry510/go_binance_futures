package skill

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
)

const DefaultSource = "native"

type VersionInfo struct {
	SkillVersion          string `json:"skill_version"`
	PromptVersion         string `json:"prompt_version"`
	PromptHash            string `json:"prompt_hash"`
	InputContractVersion  string `json:"input_contract_version"`
	OutputContractVersion string `json:"output_contract_version"`
	Source                string `json:"source"`
	SourceVersion         string `json:"source_version,omitempty"`
}

type VersionProvider interface {
	VersionInfo() VersionInfo
}

type PackageHashProvider interface {
	PackageHash() string
}

func ResolveVersionInfo(value Skill, prompt string) VersionInfo {
	info := VersionInfo{
		SkillVersion: "v1", PromptVersion: "v1",
		InputContractVersion: "v1", OutputContractVersion: "v1", Source: DefaultSource,
	}
	if provider, ok := value.(VersionProvider); ok {
		mergeVersionInfo(&info, provider.VersionInfo())
	}
	info.PromptHash = HashPrompt(prompt)
	return info
}

func PackageHash(value Skill, prompt string) string {
	if value == nil {
		return ""
	}
	if provider, ok := value.(PackageHashProvider); ok {
		if hash := strings.TrimSpace(provider.PackageHash()); len(hash) == 64 {
			return hash
		}
	}
	info := ResolveVersionInfo(value, prompt)
	tools := append([]string(nil), value.Tools()...)
	for index := range tools {
		tools[index] = strings.TrimSpace(tools[index])
	}
	sort.Strings(tools)
	identity := struct {
		SkillName             string   `json:"skill_name"`
		SkillVersion          string   `json:"skill_version"`
		PromptVersion         string   `json:"prompt_version"`
		PromptHash            string   `json:"prompt_hash"`
		InputContractVersion  string   `json:"input_contract_version"`
		OutputContractVersion string   `json:"output_contract_version"`
		Source                string   `json:"source"`
		SourceVersion         string   `json:"source_version"`
		Tools                 []string `json:"tools"`
	}{
		SkillName: value.Name(), SkillVersion: info.SkillVersion, PromptVersion: info.PromptVersion, PromptHash: info.PromptHash,
		InputContractVersion: info.InputContractVersion, OutputContractVersion: info.OutputContractVersion,
		Source: info.Source, SourceVersion: info.SourceVersion, Tools: tools,
	}
	raw, _ := json.Marshal(identity)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func HashPrompt(prompt string) string {
	sum := sha256.Sum256([]byte(prompt))
	return hex.EncodeToString(sum[:])
}

func mergeVersionInfo(dst *VersionInfo, src VersionInfo) {
	if strings.TrimSpace(src.SkillVersion) != "" {
		dst.SkillVersion = strings.TrimSpace(src.SkillVersion)
	}
	if strings.TrimSpace(src.PromptVersion) != "" {
		dst.PromptVersion = strings.TrimSpace(src.PromptVersion)
	}
	if strings.TrimSpace(src.InputContractVersion) != "" {
		dst.InputContractVersion = strings.TrimSpace(src.InputContractVersion)
	}
	if strings.TrimSpace(src.OutputContractVersion) != "" {
		dst.OutputContractVersion = strings.TrimSpace(src.OutputContractVersion)
	}
	if strings.TrimSpace(src.Source) != "" {
		dst.Source = strings.TrimSpace(src.Source)
	}
	if strings.TrimSpace(src.SourceVersion) != "" {
		dst.SourceVersion = strings.TrimSpace(src.SourceVersion)
	}
}
