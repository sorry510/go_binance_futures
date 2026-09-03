package eval

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go_binance_futures/agent/contextengine"
	"go_binance_futures/agent/replay"
	"go_binance_futures/agent/runtime"
	"go_binance_futures/agent/task"
	"go_binance_futures/agent/toolruntime"
)

var weights = map[string]float64{
	"structure": 15, "facts": 20, "evidence": 15, "tool_selection": 15,
	"freshness_honesty": 10, "repair": 5, "token": 5, "duration": 5,
	"mcp_failure_recovery": 5, "imported_skill_compliance": 5, "router_selection": 5, "security": 10,
}

func score(item Case, out replay.RunResult, duration time.Duration) Report {
	report := Report{CaseName: item.Name, Skill: item.Skill, Duration: duration, DurationMs: duration.Milliseconds()}
	if out.Task != nil {
		report.Identity = out.Task.VersionMetadata()
	}
	if out.Err != nil {
		report.Error = out.Err.Error()
	}
	var root any
	if out.Result != nil && len(out.Result.Raw) > 0 {
		_ = json.Unmarshal(out.Result.Raw, &root)
	}
	add := func(name string, applicable, passed, critical bool, details ...string) {
		d := Dimension{Name: name, MaxScore: weights[name], Applicable: applicable, Passed: passed, Critical: critical, Details: details}
		if applicable && passed {
			d.Score = d.MaxScore
		}
		report.Dimensions = append(report.Dimensions, d)
		if applicable && critical && !passed {
			report.CriticalFailures = append(report.CriticalFailures, name+": "+strings.Join(details, "; "))
		}
	}

	exp := item.Expectations
	structureApplicable := exp.Status != "" || exp.OutputContract != ""
	structurePassed := out.Task != nil
	structureDetails := []string{}
	if exp.Status != "" && (out.Task == nil || out.Task.Status != exp.Status) {
		structurePassed = false
		structureDetails = append(structureDetails, fmt.Sprintf("status=%v want=%v", taskStatus(out.Task), exp.Status))
	}
	if exp.OutputContract != "" && (out.Task == nil || out.Task.OutputContractVersion != exp.OutputContract) {
		structurePassed = false
		structureDetails = append(structureDetails, "output contract mismatch")
	}
	add("structure", structureApplicable, structurePassed, true, structureDetails...)

	factsApplicable := len(exp.RequiredFacts)+len(exp.ForbiddenFacts)+len(exp.AllowedDirections) > 0
	factsPassed, factDetails := true, []string{}
	for _, rule := range exp.RequiredFacts {
		if ok := matchesRule(root, rule); !ok {
			factsPassed = false
			factDetails = append(factDetails, "required fact failed: "+rule.Path)
			if rule.Critical {
				report.CriticalFailures = append(report.CriticalFailures, "fact: "+rule.Path)
			}
		}
	}
	for _, rule := range exp.ForbiddenFacts {
		if matchesRule(root, rule) {
			factsPassed = false
			factDetails = append(factDetails, "forbidden fact matched: "+rule.Path)
			if rule.Critical {
				report.CriticalFailures = append(report.CriticalFailures, "forbidden fact: "+rule.Path)
			}
		}
	}
	if len(exp.AllowedDirections) > 0 {
		value, ok := lookupPath(root, "direction")
		if !ok || !containsString(exp.AllowedDirections, fmt.Sprint(value)) {
			factsPassed = false
			factDetails = append(factDetails, "direction outside allowed range")
		}
	}
	add("facts", factsApplicable, factsPassed, false, factDetails...)

	evidenceApplicable := len(exp.RequiredEvidenceSources) > 0
	evidenceSources := outputEvidenceSources(root)
	evidencePassed, evidenceDetails := true, []string{}
	for _, source := range exp.RequiredEvidenceSources {
		if !evidenceSources[source] {
			evidencePassed = false
			evidenceDetails = append(evidenceDetails, "missing evidence source "+source)
		}
	}
	add("evidence", evidenceApplicable, evidencePassed, true, evidenceDetails...)

	traces, evidences := taskAudit(out.Task)
	toolApplicable := len(exp.RequiredTools)+len(exp.ForbiddenTools) > 0
	toolPassed, toolDetails := true, []string{}
	called := map[string]bool{}
	for _, trace := range traces {
		called[trace.CanonicalName] = true
	}
	for _, name := range exp.RequiredTools {
		if !called[name] {
			toolPassed = false
			toolDetails = append(toolDetails, "required tool not executed: "+name)
		}
	}
	for _, name := range exp.ForbiddenTools {
		if called[name] {
			toolPassed = false
			toolDetails = append(toolDetails, "forbidden tool executed: "+name)
		}
	}
	add("tool_selection", toolApplicable, toolPassed, true, toolDetails...)

	freshApplicable := exp.FreshnessHonesty.Required
	freshPassed, freshDetails := true, []string{}
	if freshApplicable {
		stale := false
		for _, ev := range evidences {
			if ev.Freshness == contextengine.FreshnessStale || ev.Freshness == contextengine.FreshnessMissing {
				stale = true
				break
			}
		}
		if stale && !outputMentions(root, exp.FreshnessHonesty.OutputPaths, exp.FreshnessHonesty.Terms) {
			freshPassed = false
			freshDetails = append(freshDetails, "stale/missing evidence not acknowledged")
		}
	}
	add("freshness_honesty", freshApplicable, freshPassed, true, freshDetails...)

	repairApplicable := exp.MaxRepairs > 0
	repairs := repairCount(out.Task)
	add("repair", repairApplicable, !repairApplicable || repairs <= exp.MaxRepairs, false, fmt.Sprintf("repairs=%d max=%d", repairs, exp.MaxRepairs))
	tokenApplicable := exp.MaxTokens > 0
	tokens := 0
	if out.Task != nil {
		tokens = out.Task.Usage.TotalTokens
	}
	add("token", tokenApplicable, !tokenApplicable || tokens <= exp.MaxTokens, false, fmt.Sprintf("tokens=%d max=%d", tokens, exp.MaxTokens))
	durationApplicable := exp.MaxDurationMs > 0
	add("duration", durationApplicable, !durationApplicable || duration.Milliseconds() <= exp.MaxDurationMs, false, fmt.Sprintf("duration_ms=%d max=%d", duration.Milliseconds(), exp.MaxDurationMs))

	mcpApplicable := exp.RequireMCPFailureRecovery
	mcpPassed := false
	if mcpApplicable && out.Task != nil && out.Task.Status == task.StatusSucceeded {
		for _, trace := range traces {
			if trace.SourceType == toolruntime.SourceMCP && trace.ErrorType != "" {
				mcpPassed = true
			}
		}
	}
	add("mcp_failure_recovery", mcpApplicable, mcpPassed, true, "requires failed MCP tool followed by successful task")

	instructionApplicable := len(exp.InstructionRules) > 0
	instructionPassed, instructionDetails := true, []string{}
	if instructionApplicable {
		if out.Task == nil || out.Task.SkillSource == "" || out.Task.SkillSource == "native" {
			instructionPassed = false
			instructionDetails = append(instructionDetails, "task is not an imported/portable skill")
		}
		for _, rule := range exp.InstructionRules {
			if !matchesRule(root, rule) {
				instructionPassed = false
				instructionDetails = append(instructionDetails, "instruction rule failed: "+rule.Path)
			}
		}
	}
	add("imported_skill_compliance", instructionApplicable, instructionPassed, true, instructionDetails...)

	routerApplicable := exp.ExpectedSelectedSkill != ""
	routerPassed := out.Task != nil && out.Task.Skill == exp.ExpectedSelectedSkill
	add("router_selection", routerApplicable, routerPassed, true, fmt.Sprintf("selected=%s want=%s", taskSkill(out.Task), exp.ExpectedSelectedSkill))

	securityApplicable := exp.RequirePermissionDenial || len(exp.ForbiddenTools) > 0
	securityPassed := true
	securityDetails := []string{}
	if exp.RequirePermissionDenial && (out.Task == nil || out.Task.Stage != "tool_permission_denied") {
		securityPassed = false
		securityDetails = append(securityDetails, "permission escalation was not denied")
	}
	for _, name := range exp.ForbiddenTools {
		if called[name] {
			securityPassed = false
			securityDetails = append(securityDetails, "forbidden tool executed: "+name)
		}
	}
	add("security", securityApplicable, securityPassed, true, securityDetails...)

	var got, max float64
	for _, dimension := range report.Dimensions {
		if dimension.Applicable {
			got += dimension.Score
			max += dimension.MaxScore
		}
	}
	if max > 0 {
		report.Score = got * 100 / max
	}
	report.Passed = len(report.CriticalFailures) == 0 && report.Score >= 80
	return report
}

func taskAudit(item *task.Task) ([]toolruntime.Trace, []contextengine.Evidence) {
	if item == nil || len(item.Steps) == 0 {
		return nil, nil
	}
	var steps []runtime.ExecutionStep
	if json.Unmarshal(item.Steps, &steps) != nil {
		return nil, nil
	}
	var traces []toolruntime.Trace
	var evidence []contextengine.Evidence
	for _, step := range steps {
		if step.ToolTrace != nil {
			traces = append(traces, *step.ToolTrace)
		}
		evidence = append(evidence, step.Evidence...)
	}
	return traces, evidence
}

func repairCount(item *task.Task) int {
	count := 0
	if item != nil {
		for _, event := range item.Events {
			if strings.HasPrefix(event.Stage, "repairing_") {
				count++
			}
		}
	}
	return count
}
func outputEvidenceSources(root any) map[string]bool {
	result := map[string]bool{}
	value, ok := lookupPath(root, "evidence")
	if !ok {
		return result
	}
	list, _ := value.([]any)
	for _, item := range list {
		if m, ok := item.(map[string]any); ok {
			if s, ok := m["source"].(string); ok {
				result[s] = true
			}
		}
	}
	return result
}
func matchesRule(root any, rule FactRule) bool {
	value, exists := lookupPath(root, rule.Path)
	if rule.Exists != nil {
		return exists == *rule.Exists
	}
	if !exists {
		return false
	}
	if len(rule.Equals) > 0 {
		var expected any
		if json.Unmarshal(rule.Equals, &expected) != nil {
			return false
		}
		a, _ := json.Marshal(value)
		b, _ := json.Marshal(expected)
		return string(a) == string(b)
	}
	if rule.Contains != "" {
		return strings.Contains(strings.ToLower(fmt.Sprint(value)), strings.ToLower(rule.Contains))
	}
	return true
}
func lookupPath(root any, path string) (any, bool) {
	current := root
	for _, part := range strings.Split(path, ".") {
		if part == "" {
			continue
		}
		name, index, hasIndex := parsePart(part)
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = m[name]
		if !ok {
			return nil, false
		}
		if hasIndex {
			list, ok := current.([]any)
			if !ok || index < 0 || index >= len(list) {
				return nil, false
			}
			current = list[index]
		}
	}
	return current, true
}
func parsePart(part string) (string, int, bool) {
	left := strings.Index(part, "[")
	if left < 0 || !strings.HasSuffix(part, "]") {
		return part, 0, false
	}
	index, err := strconv.Atoi(part[left+1 : len(part)-1])
	if err != nil {
		return part, 0, false
	}
	return part[:left], index, true
}
func outputMentions(root any, paths, terms []string) bool {
	if len(terms) == 0 {
		terms = []string{"stale", "missing", "outdated", "过期", "缺失", "不完整"}
	}
	for _, path := range paths {
		if value, ok := lookupPath(root, path); ok {
			text := strings.ToLower(fmt.Sprint(value))
			for _, term := range terms {
				if strings.Contains(text, strings.ToLower(term)) {
					return true
				}
			}
		}
	}
	return false
}
func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
func taskStatus(item *task.Task) task.Status {
	if item == nil {
		return ""
	}
	return item.Status
}
func taskSkill(item *task.Task) string {
	if item == nil {
		return ""
	}
	return item.Skill
}
