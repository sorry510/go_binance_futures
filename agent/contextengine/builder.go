package contextengine

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"go_binance_futures/llm"
)

type Engine struct {
	Estimator TokenEstimator
	Freshness FreshnessPolicy
}

func New() *Engine {
	return &Engine{Estimator: HeuristicEstimator{}, Freshness: DefaultFreshnessPolicy()}
}

func (engine *Engine) Build(options BuildOptions) ([]llm.Message, BuildTrace, error) {
	if engine == nil {
		engine = New()
	}
	estimator := engine.Estimator
	if estimator == nil {
		estimator = HeuristicEstimator{}
	}
	now := options.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	trace := BuildTrace{
		BudgetTokens: options.MaxTokens, BudgetBytes: options.MaxBytes,
		SystemTokens: estimator.Estimate(options.SystemPrompt), InputBlocks: len(options.Blocks), BuiltAt: now.Format(time.RFC3339),
		Trimmed: []TrimRecord{}, SelectedBlockIDs: []string{}, StaleEvidenceIDs: []string{},
	}
	if options.MaxTokens > 0 && trace.SystemTokens > options.MaxTokens {
		return nil, trace, fmt.Errorf("required system context exceeds token budget: %d > %d", trace.SystemTokens, options.MaxTokens)
	}
	if options.MaxBytes > 0 && len(options.SystemPrompt) > options.MaxBytes {
		return nil, trace, fmt.Errorf("required system context exceeds byte budget: %d > %d", len(options.SystemPrompt), options.MaxBytes)
	}

	blocks := normalizeBlocks(options.Blocks, estimator)
	blocks, duplicateTrim := deduplicateBlocks(blocks)
	trace.Trimmed = append(trace.Trimmed, duplicateTrim...)

	required := make([]ContextBlock, 0, len(blocks))
	optional := make([]ContextBlock, 0, len(blocks))
	for _, block := range blocks {
		if block.Required {
			required = append(required, block)
		} else {
			optional = append(optional, block)
		}
	}
	selected := make([]ContextBlock, 0, len(blocks))
	usedTokens := trace.SystemTokens
	usedBytes := len(options.SystemPrompt)
	for _, block := range required {
		if !fitsBudget(usedTokens+block.EstimatedTokens, usedBytes+len(block.Content), options.MaxTokens, options.MaxBytes) {
			return nil, trace, fmt.Errorf("required context block %q exceeds context budget", block.ID)
		}
		selected = append(selected, block)
		usedTokens += block.EstimatedTokens
		usedBytes += len(block.Content)
	}

	sort.SliceStable(optional, func(i, j int) bool {
		left, right := effectivePriority(optional[i]), effectivePriority(optional[j])
		if left == right {
			return optional[i].Order > optional[j].Order
		}
		return left > right
	})
	for _, block := range optional {
		if fitsBudget(usedTokens+block.EstimatedTokens, usedBytes+len(block.Content), options.MaxTokens, options.MaxBytes) {
			selected = append(selected, block)
			usedTokens += block.EstimatedTokens
			usedBytes += len(block.Content)
			continue
		}
		reason := "token_budget"
		if options.MaxBytes > 0 && usedBytes+len(block.Content) > options.MaxBytes {
			reason = "byte_budget"
		}
		trace.Trimmed = append(trace.Trimmed, TrimRecord{BlockID: block.ID, Type: block.Type, Source: block.Source, EstimatedTokens: block.EstimatedTokens, Reason: reason})
	}

	sort.SliceStable(selected, func(i, j int) bool { return selected[i].Order < selected[j].Order })
	messages := make([]llm.Message, 0, len(selected))
	for _, block := range selected {
		role := strings.TrimSpace(block.Role)
		if role == "" {
			role = llm.RoleUser
		}
		content := block.Content
		if block.Freshness == FreshnessStale || block.Freshness == FreshnessMissing {
			content = contextFreshnessHeader(block) + "\n" + content
		}
		messages = append(messages, llm.Message{Role: role, Content: content})
		trace.SelectedBlockIDs = append(trace.SelectedBlockIDs, block.ID)
		if block.Freshness == FreshnessStale || block.Freshness == FreshnessMissing {
			trace.StaleEvidenceIDs = append(trace.StaleEvidenceIDs, block.EvidenceIDs...)
		}
	}
	trace.SelectedTokens = usedTokens
	trace.SelectedBytes = usedBytes
	trace.SelectedBlocks = len(selected)
	trace.TrimmedBlocks = len(trace.Trimmed)
	return messages, trace, nil
}

func normalizeBlocks(blocks []ContextBlock, estimator TokenEstimator) []ContextBlock {
	result := make([]ContextBlock, 0, len(blocks))
	for index, original := range blocks {
		block := original
		block.Content = strings.TrimSpace(block.Content)
		if block.Content == "" {
			continue
		}
		if block.ID == "" {
			block.ID = fmt.Sprintf("ctx-%03d", index+1)
		}
		if block.Source == "" {
			block.Source = string(block.Type)
		}
		if block.Priority == 0 {
			block.Priority = DefaultPriority(block.Type)
		}
		if block.Freshness == "" {
			block.Freshness = FreshnessUnknown
		}
		if block.ContentHash == "" {
			block.ContentHash = ContentHash(block.Content)
		}
		if block.EstimatedTokens <= 0 {
			block.EstimatedTokens = estimator.Estimate(block.Content)
		}
		block.Order = index
		result = append(result, block)
	}
	return result
}

func deduplicateBlocks(blocks []ContextBlock) ([]ContextBlock, []TrimRecord) {
	best := map[string]int{}
	keep := make([]bool, len(blocks))
	for index, block := range blocks {
		key := block.ContentHash
		if previous, exists := best[key]; exists {
			if effectivePriority(block) > effectivePriority(blocks[previous]) || (effectivePriority(block) == effectivePriority(blocks[previous]) && block.Order > blocks[previous].Order) {
				keep[previous] = false
				best[key] = index
				keep[index] = true
			}
			continue
		}
		best[key] = index
		keep[index] = true
	}
	result := make([]ContextBlock, 0, len(blocks))
	trimmed := []TrimRecord{}
	for index, block := range blocks {
		if keep[index] {
			result = append(result, block)
		} else {
			trimmed = append(trimmed, TrimRecord{BlockID: block.ID, Type: block.Type, Source: block.Source, EstimatedTokens: block.EstimatedTokens, Reason: "duplicate"})
		}
	}
	return result, trimmed
}

func DefaultPriority(blockType BlockType) int {
	switch blockType {
	case BlockSystem:
		return 1000
	case BlockTask:
		return 900
	case BlockMarket:
		return 800
	case BlockSkillInstruction:
		return 700
	case BlockTool:
		return 600
	case BlockMCPResource:
		return 500
	case BlockMemory:
		return 300
	case BlockHistory:
		return 200
	default:
		return 100
	}
}

func effectivePriority(block ContextBlock) int {
	priority := block.Priority
	if priority == 0 {
		priority = DefaultPriority(block.Type)
	}
	if block.Freshness == FreshnessStale {
		priority -= 100
	} else if block.Freshness == FreshnessMissing {
		priority -= 200
	}
	return priority
}

func fitsBudget(tokens, bytes, maxTokens, maxBytes int) bool {
	return (maxTokens <= 0 || tokens <= maxTokens) && (maxBytes <= 0 || bytes <= maxBytes)
}

func contextFreshnessHeader(block ContextBlock) string {
	return fmt.Sprintf("CONTEXT_FRESHNESS source=%s status=%s as_of=%s", block.Source, block.Freshness, block.AsOf)
}
