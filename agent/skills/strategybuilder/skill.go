package strategybuilder

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go_binance_futures/agent/skill"
	"go_binance_futures/agent/validator"
	"go_binance_futures/llm"
)

const (
	Name               = "strategy_builder"
	HistoryMetadataKey = "strategy_builder_history"
	maxRounds          = 10
)

type Input struct {
	Prompt          string `json:"prompt"`
	PreviousJSON    string `json:"previous_json,omitempty"`
	ValidationError string `json:"validation_error,omitempty"`
}

type Options struct {
	Validate               func([]byte) error
	RepairGuidance         func(errorMessage, candidateJSON string) string
	RequireMarketCondition bool
}

type Builder struct{ options Options }

func New(options Options) *Builder    { return &Builder{options: options} }
func (*Builder) Name() string         { return Name }
func (*Builder) SystemPrompt() string { return systemPrompt }
func (*Builder) Tools() []string {
	return []string{"get_features", "get_test_strategy_results", "get_market_condition"}
}
func (*Builder) MaxRounds() int { return maxRounds }

func (builder *Builder) BuildInput(_ context.Context, req skill.Request) ([]llm.Message, error) {
	var input Input
	if err := json.Unmarshal([]byte(req.Input), &input); err != nil {
		return nil, fmt.Errorf("decode strategy builder input: %w", err)
	}
	input.Prompt = strings.TrimSpace(input.Prompt)
	if input.Prompt == "" {
		return nil, fmt.Errorf("strategy builder prompt is required")
	}
	messages := historyFromMetadata(req.Metadata)
	messages = compactRepairHistory(messages)
	messages = append(messages, llm.Message{Role: llm.RoleUser, Content: BuildUserPrompt(input)})
	return messages, nil
}

func (builder *Builder) RequiredTools(req skill.Request) []string {
	var input Input
	if json.Unmarshal([]byte(req.Input), &input) != nil {
		return nil
	}
	required := requiredTools(input.Prompt)
	if RequiresMarketConditionForConversation(input.Prompt, historyFromMetadata(req.Metadata)) {
		required = appendUniqueTool(required, "get_market_condition")
	}
	return required
}

func (builder *Builder) Validator() validator.FinalValidator {
	return validator.Func(func(_ context.Context, raw json.RawMessage) (any, error) {
		if builder.options.Validate == nil {
			return nil, fmt.Errorf("strategy builder validator is unavailable")
		}
		if err := builder.options.Validate(raw); err != nil {
			return nil, builder.validationError(err, raw)
		}
		if builder.options.RequireMarketCondition {
			if err := validateMarketConditionCoverage(raw); err != nil {
				return nil, builder.validationError(err, raw)
			}
		}
		if err := validateCloseStrategyQuality(raw); err != nil {
			return nil, builder.validationError(err, raw)
		}
		if err := validateStrategyCodeReadability(raw); err != nil {
			return nil, builder.validationError(err, raw)
		}
		var value any
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, err
		}
		return value, nil
	})
}

func BuildUserPrompt(input Input) string {
	var builder strings.Builder
	builder.WriteString("User requirements:\n")
	builder.WriteString(input.Prompt)
	if strings.TrimSpace(input.PreviousJSON) != "" || strings.TrimSpace(input.ValidationError) != "" {
		builder.WriteString("\n\nRepair context from the previous attempt. Generate a complete replacement JSON, not a patch.\n")
		if strings.TrimSpace(input.ValidationError) != "" {
			builder.WriteString("Validation/import error:\n")
			builder.WriteString(input.ValidationError)
			builder.WriteString("\n")
		}
		if strings.TrimSpace(input.PreviousJSON) != "" {
			builder.WriteString("Previous JSON:\n")
			builder.WriteString(input.PreviousJSON)
		}
	}
	return builder.String()
}

func requiredTools(prompt string) []string {
	normalized := strings.Join(strings.Fields(strings.ToLower(prompt)), "")
	required := make([]string, 0, 2)
	if strings.Contains(normalized, "get_test_strategy_results") {
		required = append(required, "get_test_strategy_results")
	}
	markers := []string{"调用测试结果", "查询测试结果", "获取测试结果", "查看测试结果", "分析测试结果", "testresults"}
	if len(required) == 0 {
		for _, marker := range markers {
			if strings.Contains(normalized, marker) {
				required = append(required, "get_test_strategy_results")
				break
			}
		}
	}
	requestsData := strings.Contains(normalized, "获取") || strings.Contains(normalized, "查询") || strings.Contains(normalized, "get") || strings.Contains(normalized, "query")
	mentionsInstrument := strings.Contains(normalized, "币") || strings.Contains(normalized, "合约") || strings.Contains(normalized, "coin") || strings.Contains(normalized, "contract") || strings.Contains(normalized, "symbol") || strings.Contains(normalized, "usdt") || strings.Contains(normalized, "usdc")
	mentionsData := strings.Contains(normalized, "数据") || strings.Contains(normalized, "data")
	if strings.Contains(normalized, "get_features") || (requestsData && mentionsInstrument && mentionsData) {
		required = append(required, "get_features")
	}
	return required
}

func appendUniqueTool(values []string, name string) []string {
	for _, value := range values {
		if value == name {
			return values
		}
	}
	return append(values, name)
}

func (builder *Builder) validationError(err error, raw json.RawMessage) error {
	message := err.Error()
	if builder.options.RepairGuidance != nil {
		if guidance := strings.TrimSpace(builder.options.RepairGuidance(message, string(raw))); guidance != "" {
			message += "; required_fix: " + guidance
		}
	}
	return fmt.Errorf("%s", message)
}

func historyFromMetadata(metadata map[string]any) []llm.Message {
	if metadata == nil {
		return nil
	}
	history, ok := metadata[HistoryMetadataKey].([]llm.Message)
	if !ok {
		return nil
	}
	return append([]llm.Message(nil), history...)
}

func compactRepairHistory(messages []llm.Message) []llm.Message {
	lastFeedback := -1
	for index, message := range messages {
		if message.Role == llm.RoleUser && strings.HasPrefix(message.Content, "AGENT_FEEDBACK\n") {
			lastFeedback = index
		}
	}
	if lastFeedback < 0 {
		return messages
	}
	skip := make(map[int]bool)
	for index := 0; index < lastFeedback; index++ {
		message := messages[index]
		if message.Role != llm.RoleUser || !strings.HasPrefix(message.Content, "AGENT_FEEDBACK\n") {
			continue
		}
		skip[index] = true
		if index > 0 && messages[index-1].Role == llm.RoleAssistant {
			skip[index-1] = true
		}
	}
	result := make([]llm.Message, 0, len(messages)-len(skip))
	for index, message := range messages {
		if !skip[index] {
			result = append(result, message)
		}
	}
	return result
}
