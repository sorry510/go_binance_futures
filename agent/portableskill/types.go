package portableskill

import "go_binance_futures/models"

const (
	SkillTypeNative   = "native"
	SkillTypePortable = "portable"
	ValidationValid   = "valid"
)

type Frontmatter struct {
	Name          string            `yaml:"name" json:"name"`
	Description   string            `yaml:"description" json:"description"`
	License       string            `yaml:"license,omitempty" json:"license,omitempty"`
	Compatibility string            `yaml:"compatibility,omitempty" json:"compatibility,omitempty"`
	Metadata      map[string]string `yaml:"metadata,omitempty" json:"metadata,omitempty"`
	AllowedTools  string            `yaml:"allowed-tools,omitempty" json:"allowed-tools,omitempty"`
}

type Diagnostic struct {
	Level   string `json:"level"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
}

type Package struct {
	Root           string
	Frontmatter    Frontmatter
	Body           string
	PackageHash    string
	Files          []string
	FileCount      int
	TotalBytes     int64
	RequestedTools []string
	Diagnostics    []Diagnostic
}

type ImportResult struct {
	Skill     models.AgentSkill        `json:"skill"`
	Version   models.AgentSkillVersion `json:"version"`
	Duplicate bool                     `json:"duplicate"`
}

type VersionDetail struct {
	Version     models.AgentSkillVersion      `json:"version"`
	Permissions []models.AgentSkillPermission `json:"permissions"`
	Files       []string                      `json:"files"`
}
