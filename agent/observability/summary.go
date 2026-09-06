package observability

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	"go_binance_futures/models"
)

type TaskAggregate struct {
	Tasks             int64   `json:"tasks"`
	Succeeded         int64   `json:"succeeded"`
	Failed            int64   `json:"failed"`
	Cancelled         int64   `json:"cancelled"`
	SuccessRate       float64 `json:"success_rate"`
	TotalTokens       int64   `json:"total_tokens"`
	AverageTokens     float64 `json:"average_tokens"`
	AverageDurationMs float64 `json:"average_duration_ms"`
	P95DurationMs     int64   `json:"p95_duration_ms"`
	AverageRounds     float64 `json:"average_rounds"`
}

type DimensionAggregate struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	TaskAggregate
}

type ContextAggregate struct {
	Builds           int64   `json:"builds"`
	AverageTokens    float64 `json:"average_tokens"`
	TrimRate         float64 `json:"trim_rate"`
	MemoryHitRate    float64 `json:"memory_hit_rate"`
	SelectedMemories int64   `json:"selected_memories"`
	TrimmedMemories  int64   `json:"trimmed_memories"`
}

type ToolAggregate struct {
	Tool             string  `json:"tool"`
	Source           string  `json:"source"`
	Calls            int64   `json:"calls"`
	Errors           int64   `json:"errors"`
	ErrorRate        float64 `json:"error_rate"`
	CacheHitRate     float64 `json:"cache_hit_rate"`
	PartialRate      float64 `json:"partial_rate"`
	Timeouts         int64   `json:"timeouts"`
	AverageLatencyMs float64 `json:"average_latency_ms"`
	P95LatencyMs     int64   `json:"p95_latency_ms"`
}

type MCPServerAggregate struct {
	ID              int64   `json:"id"`
	Name            string  `json:"name"`
	Status          string  `json:"status"`
	ProtocolVersion string  `json:"protocol_version,omitempty"`
	CatalogHash     string  `json:"catalog_hash,omitempty"`
	LastSuccessAt   int64   `json:"last_success_at,omitempty"`
	LastErrorAt     int64   `json:"last_error_at,omitempty"`
	Calls           int64   `json:"calls"`
	Errors          int64   `json:"errors"`
	Availability    float64 `json:"availability"`
	P95LatencyMs    int64   `json:"p95_latency_ms"`
}

type toolAccumulator struct {
	calls, errors, cache, partial, timeouts int64
	latency                                 int64
	durations                               []int64
	tool, source                            string
}

type NamedCount struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type EvalAggregate struct {
	Runs         int64   `json:"runs"`
	Passed       int64   `json:"passed"`
	PassRate     float64 `json:"pass_rate"`
	AverageScore float64 `json:"average_score"`
}

type EvidenceAggregate struct {
	Validations     int64   `json:"validations"`
	WithEvidence    int64   `json:"with_evidence"`
	CoverageRate    float64 `json:"coverage_rate"`
	AverageEvidence float64 `json:"average_evidence"`
}

type Summary struct {
	StartTime       int64                `json:"start_time"`
	EndTime         int64                `json:"end_time"`
	Global          TaskAggregate        `json:"global"`
	BySkill         []DimensionAggregate `json:"by_skill"`
	ByModel         []DimensionAggregate `json:"by_model"`
	ByPrompt        []DimensionAggregate `json:"by_prompt"`
	BySkillRevision []DimensionAggregate `json:"by_skill_revision"`
	Context         ContextAggregate     `json:"context"`
	Tools           []ToolAggregate      `json:"tools"`
	MCPServers      []MCPServerAggregate `json:"mcp_servers"`
	Repairs         []NamedCount         `json:"repairs"`
	Errors          []NamedCount         `json:"errors"`
	Evidence        EvidenceAggregate    `json:"evidence"`
	Eval            EvalAggregate        `json:"eval"`
	ChangeEvents    int64                `json:"change_events"`
}

type taskAccumulator struct {
	TaskAggregate
	durations []int64
	rounds    int64
}

func (a *taskAccumulator) add(row models.AgentTask) {
	a.Tasks++
	switch row.Status {
	case "succeeded":
		a.Succeeded++
	case "cancelled":
		a.Cancelled++
	default:
		a.Failed++
	}
	a.TotalTokens += int64(row.TotalTokens)
	a.rounds += int64(row.Round)
	if row.StartedAt > 0 && row.CompletedAt >= row.StartedAt {
		a.durations = append(a.durations, row.CompletedAt-row.StartedAt)
	}
}

func (a *taskAccumulator) finish() TaskAggregate {
	r := a.TaskAggregate
	r.SuccessRate = safeRatio(r.Succeeded, r.Tasks)
	if r.Tasks > 0 {
		r.AverageTokens = float64(r.TotalTokens) / float64(r.Tasks)
		r.AverageRounds = float64(a.rounds) / float64(r.Tasks)
	}
	if len(a.durations) > 0 {
		var sum int64
		for _, v := range a.durations {
			sum += v
		}
		r.AverageDurationMs = float64(sum) / float64(len(a.durations))
		r.P95DurationMs = percentile64(a.durations, .95)
	}
	return r
}

func normalizeWindow(start, end int64) (int64, int64) {
	now := time.Now().UTC().UnixMilli()
	if end <= 0 {
		end = now
	}
	if start <= 0 {
		start = end - int64(24*time.Hour/time.Millisecond)
	}
	return start, end
}

func (s Store) Summary(ctx context.Context, start, end int64) (Summary, error) {
	start, end = normalizeWindow(start, end)
	if err := validateWindow(start, end); err != nil {
		return Summary{}, err
	}
	o := s.orm()
	var tasks []models.AgentTask
	if _, err := o.QueryTable(new(models.AgentTask)).Filter("created_at__gte", start).Filter("created_at__lte", end).All(&tasks); err != nil {
		return Summary{}, err
	}
	var observations []models.AgentObservation
	if _, err := o.QueryTable(new(models.AgentObservation)).Filter("created_at__gte", start).Filter("created_at__lte", end).All(&observations); err != nil {
		return Summary{}, err
	}
	result := Summary{StartTime: start, EndTime: end}
	global := &taskAccumulator{}
	skillMap, modelMap, promptMap, revisionMap := map[string]*taskAccumulator{}, map[string]*taskAccumulator{}, map[string]*taskAccumulator{}, map[string]*taskAccumulator{}
	for _, row := range tasks {
		global.add(row)
		addDimension(skillMap, row.Skill, row)
		modelKey := strings.TrimSpace(row.Provider) + "/" + strings.TrimSpace(row.Model) + "#" + strconv.FormatInt(max64(row.FinalModelConfigID, row.ModelConfigID), 10)
		addDimension(modelMap, modelKey, row)
		promptKey := strings.TrimSpace(row.PromptVersion)
		if promptKey == "" {
			promptKey = shortKey(row.PromptHash)
		}
		addDimension(promptMap, promptKey, row)
		rev := strings.TrimSpace(row.SkillVersion)
		if row.SkillSourceVersion != "" {
			rev = row.SkillSource + ":" + row.SkillSourceVersion
		}
		addDimension(revisionMap, rev, row)
	}
	result.Global = global.finish()
	result.BySkill = finishDimensions(skillMap)
	result.ByModel = finishDimensions(modelMap)
	result.ByPrompt = finishDimensions(promptMap)
	result.BySkillRevision = finishDimensions(revisionMap)
	builds, contextTokens, trimmedBuilds, memoryHitBuilds := int64(0), int64(0), int64(0), int64(0)
	selectedMemory, trimmedMemory := int64(0), int64(0)
	tools := map[string]*toolAccumulator{}
	repairs := map[string]int64{}
	errorsByType := map[string]int64{}
	mcpCalls := map[int64]*toolAccumulator{}
	validations, withEvidence, evidenceTotal := int64(0), int64(0), int64(0)
	evalRuns, evalPassed := int64(0), int64(0)
	evalScoreTotal := float64(0)
	for _, obs := range observations {
		if obs.ErrorType != "" {
			errorsByType[obs.ErrorType]++
		}
		switch obs.Type {
		case "context_build":
			builds++
			contextTokens += int64(obs.ContextTokens)
			if obs.TrimmedBlocks > 0 {
				trimmedBuilds++
			}
			if obs.MemorySelected > 0 {
				memoryHitBuilds++
			}
			selectedMemory += int64(obs.MemorySelected)
			trimmedMemory += int64(obs.MemoryTrimmed)
		case "repair":
			repairs[obs.Status]++
		case "eval":
			evalRuns++
			evalScoreTotal += obs.EvalScore
			if obs.Status == "passed" {
				evalPassed++
			}
		case "validation":
			validations++
			evidenceTotal += int64(obs.EvidenceCount)
			if obs.EvidenceCount > 0 {
				withEvidence++
			}
		case "tool_call":
			key := obs.ToolSource + "|" + obs.Tool
			a := tools[key]
			if a == nil {
				a = &toolAccumulator{tool: obs.Tool, source: obs.ToolSource}
				tools[key] = a
			}
			accumulateTool(a, obs)
			if obs.ToolSource == "mcp" {
				if id := providerRefID(obs.ProviderRef); id > 0 {
					a := mcpCalls[id]
					if a == nil {
						a = &toolAccumulator{}
						mcpCalls[id] = a
					}
					accumulateTool(a, obs)
				}
			}
		}
	}
	result.Context = ContextAggregate{Builds: builds, TrimRate: safeRatio(trimmedBuilds, builds), MemoryHitRate: safeRatio(memoryHitBuilds, builds), SelectedMemories: selectedMemory, TrimmedMemories: trimmedMemory}
	if builds > 0 {
		result.Context.AverageTokens = float64(contextTokens) / float64(builds)
	}
	for _, a := range tools {
		result.Tools = append(result.Tools, finishTool(a))
	}
	sort.Slice(result.Tools, func(i, j int) bool { return result.Tools[i].Calls > result.Tools[j].Calls })
	result.Repairs = finishCounts(repairs)
	result.Errors = finishCounts(errorsByType)
	result.Evidence = EvidenceAggregate{Validations: validations, WithEvidence: withEvidence, CoverageRate: safeRatio(withEvidence, validations)}
	if validations > 0 {
		result.Evidence.AverageEvidence = float64(evidenceTotal) / float64(validations)
	}
	result.Eval = EvalAggregate{Runs: evalRuns, Passed: evalPassed, PassRate: safeRatio(evalPassed, evalRuns)}
	if evalRuns > 0 {
		result.Eval.AverageScore = evalScoreTotal / float64(evalRuns)
	}
	var servers []models.AgentMCPServer
	_, _ = o.QueryTable(new(models.AgentMCPServer)).All(&servers)
	for _, server := range servers {
		a := mcpCalls[server.ID]
		m := MCPServerAggregate{ID: server.ID, Name: server.Name, Status: server.Status, ProtocolVersion: server.ProtocolVersion, CatalogHash: server.CatalogHash, LastSuccessAt: server.LastSuccessAt, LastErrorAt: server.LastErrorAt}
		if a != nil {
			m.Calls = a.calls
			m.Errors = a.errors
			m.Availability = 1 - safeRatio(a.errors, a.calls)
			m.P95LatencyMs = percentile64(a.durations, .95)
		}
		result.MCPServers = append(result.MCPServers, m)
	}
	changes, err := o.QueryTable(new(models.AgentChangeEvent)).Filter("created_at__gte", start).Filter("created_at__lte", end).Count()
	if err == nil {
		result.ChangeEvents = changes
	}
	return result, nil
}

func addDimension(m map[string]*taskAccumulator, key string, row models.AgentTask) {
	key = strings.TrimSpace(key)
	if key == "" {
		key = "unknown"
	}
	a := m[key]
	if a == nil {
		a = &taskAccumulator{}
		m[key] = a
	}
	a.add(row)
}
func finishDimensions(m map[string]*taskAccumulator) []DimensionAggregate {
	r := make([]DimensionAggregate, 0, len(m))
	for k, a := range m {
		r = append(r, DimensionAggregate{Key: k, Label: k, TaskAggregate: a.finish()})
	}
	sort.Slice(r, func(i, j int) bool { return r[i].Tasks > r[j].Tasks })
	return r
}
func shortKey(v string) string {
	v = strings.TrimSpace(v)
	if len(v) > 12 {
		return v[:12]
	}
	if v == "" {
		return "unknown"
	}
	return v
}
func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
func providerRefID(v string) int64 {
	const p = "mcp-server:"
	if !strings.HasPrefix(v, p) {
		return 0
	}
	n, _ := strconv.ParseInt(strings.TrimPrefix(v, p), 10, 64)
	return n
}
func accumulateTool(a *toolAccumulator, obs models.AgentObservation) {
	a.calls++
	if obs.Status == "error" {
		a.errors++
	}
	if obs.CacheHit {
		a.cache++
	}
	if obs.Partial {
		a.partial++
	}
	if obs.ErrorType == "timeout" {
		a.timeouts++
	}
	a.latency += obs.DurationMs
	a.durations = append(a.durations, obs.DurationMs)
}

func finishTool(a *toolAccumulator) ToolAggregate {
	r := ToolAggregate{Tool: a.tool, Source: a.source, Calls: a.calls, Errors: a.errors, ErrorRate: safeRatio(a.errors, a.calls), CacheHitRate: safeRatio(a.cache, a.calls), PartialRate: safeRatio(a.partial, a.calls), Timeouts: a.timeouts, P95LatencyMs: percentile64(a.durations, .95)}
	if a.calls > 0 {
		r.AverageLatencyMs = float64(a.latency) / float64(a.calls)
	}
	return r
}

func finishCounts(m map[string]int64) []NamedCount {
	r := make([]NamedCount, 0, len(m))
	for k, v := range m {
		if strings.TrimSpace(k) != "" {
			r = append(r, NamedCount{Name: k, Count: v})
		}
	}
	sort.Slice(r, func(i, j int) bool { return r[i].Count > r[j].Count })
	return r
}
