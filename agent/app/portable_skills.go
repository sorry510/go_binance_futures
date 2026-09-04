package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"unicode"

	"go_binance_futures/agent/permission"
	"go_binance_futures/agent/portableskill"
	"go_binance_futures/agent/skill"
	agenttools "go_binance_futures/agent/tools"
	"go_binance_futures/models"
)

var portableMu sync.Mutex
var portableSkills *skill.Registry
var portableTools *agenttools.Registry
var portableRegisteredSkills = map[string]bool{}
var portableRegisteredTools = map[string]bool{}

func initializeDefaultPortableSkills(skills *skill.Registry, tools *agenttools.Registry) error {
	portableMu.Lock()
	portableSkills, portableTools = skills, tools
	portableMu.Unlock()
	return SyncDefaultPortableSkills(context.Background())
}

func SyncDefaultPortableSkills(ctx context.Context) error {
	portableMu.Lock()
	skills, tools := portableSkills, portableTools
	portableMu.Unlock()
	if skills == nil || tools == nil {
		return fmt.Errorf("portable skill runtime is not initialized")
	}
	versions, err := (portableskill.Store{}).ActiveVersions(ctx)
	if err != nil {
		return err
	}
	activeSkills := map[string]bool{}
	activeTools := map[string]bool{}
	for _, version := range versions {
		adapter, err := portableskill.LoadAdapter(version)
		if err != nil {
			return err
		}
		if existing, ok := skills.Get(adapter.Name()); ok && !portableRegisteredSkills[adapter.Name()] && existing.Name() == adapter.Name() {
			return fmt.Errorf("portable skill %q conflicts with registered native skill", adapter.Name())
		}
		if err := skills.Upsert(adapter); err != nil {
			return err
		}
		activeSkills[adapter.Name()] = true
		resource := adapter.ResourceTool()
		if resource != nil {
			if err := tools.Upsert(resource); err != nil {
				return err
			}
			activeTools[resource.Name()] = true
		}
	}
	portableMu.Lock()
	for name := range portableRegisteredSkills {
		if !activeSkills[name] {
			skills.Unregister(name)
		}
	}
	for name := range portableRegisteredTools {
		if !activeTools[name] {
			tools.Unregister(name)
		}
	}
	portableRegisteredSkills, portableRegisteredTools = activeSkills, activeTools
	portableMu.Unlock()
	return nil
}

func PortableToolAllowlist(ctx context.Context, skillName string) ([]string, error) {
	return (portableskill.Store{}).GrantedToolNames(ctx, skillName)
}
func EffectiveToolAllowlist(ctx context.Context, skillName string) ([]string, error) {
	mcp, err := MCPToolAllowlist(ctx, skillName)
	if err != nil {
		return nil, err
	}
	portable, err := PortableToolAllowlist(ctx, skillName)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	out := []string{}
	for _, group := range [][]string{mcp, portable} {
		for _, name := range group {
			name = strings.TrimSpace(name)
			if name != "" && !seen[name] {
				seen[name] = true
				out = append(out, name)
			}
		}
	}
	sort.Strings(out)
	return out, nil
}

func ReviewPortableSkillPermissions(ctx context.Context, versionID int64) error {
	if _, err := DefaultManager(); err != nil {
		return err
	}
	portableMu.Lock()
	registry := portableTools
	portableMu.Unlock()
	if registry == nil {
		return fmt.Errorf("tool registry is not initialized")
	}
	store := portableskill.Store{}
	rows, err := store.Permissions(ctx, versionID)
	if err != nil {
		return err
	}
	all := registry.List()
	for idx := range rows {
		row := rows[idx]
		candidates := resolveRequestedTool(row.RequestedName, all)
		previous := row.ResolvedName
		switch len(candidates) {
		case 0:
			row.ResolvedName = ""
			row.Risk = ""
			row.Status = "unresolved"
			row.Granted = 0
			row.Reason = "no registered Tool matches the requested allowed-tools entry"
		case 1:
			selected := candidates[0]
			row.ResolvedName = selected.Name()
			row.Risk = string(selected.Risk())
			if previous != row.ResolvedName {
				row.Granted = 0
			}
			if row.Granted == 1 {
				row.Status = "granted"
				row.Reason = ""
			} else if selected.Risk() == permission.RiskRead {
				row.Status = "review_required"
				row.Reason = "author requested this Tool; administrator approval is required"
			} else {
				row.Status = "high_risk"
				row.Reason = "write/trade Tool requires explicit administrator approval and runtime policy"
			}
		default:
			row.ResolvedName = ""
			row.Risk = ""
			row.Status = "ambiguous"
			row.Granted = 0
			names := []string{}
			for _, candidate := range candidates {
				names = append(names, candidate.Name())
			}
			row.Reason = "ambiguous Tool request: " + strings.Join(names, ", ")
		}
		if err := store.SavePermissionReview(ctx, &row); err != nil {
			return err
		}
	}
	return nil
}
func resolveRequestedTool(requested string, all []agenttools.Tool) []agenttools.Tool {
	requested = strings.TrimSpace(requested)
	for _, tool := range all {
		if strings.TrimSpace(tool.Metadata().SourceType) == "portable_skill" {
			continue
		}
		if tool.Name() == requested {
			return []agenttools.Tool{tool}
		}
	}
	key := portableAliasKey(requested)
	if key == "" {
		return nil
	}
	out := []agenttools.Tool{}
	for _, tool := range all {
		if strings.TrimSpace(tool.Metadata().SourceType) == "portable_skill" {
			continue
		}
		name := tool.Name()
		if index := strings.LastIndex(name, "."); index >= 0 {
			name = name[index+1:]
		}
		if portableAliasKey(name) == key {
			out = append(out, tool)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}
func portableAliasKey(value string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}
func SetPortableSkillPermission(ctx context.Context, id int64, granted int) (*models.AgentSkillPermission, error) {
	row, err := (portableskill.Store{}).SetPermissionGrant(ctx, id, granted)
	if err != nil {
		return nil, err
	}
	return row, nil
}
