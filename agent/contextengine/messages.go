package contextengine

import (
	"fmt"
	"strings"

	"go_binance_futures/llm"
)

func InitialMessageBlocks(messages []llm.Message) []ContextBlock {
	blocks := make([]ContextBlock, 0, len(messages))
	last := len(messages) - 1
	for index, message := range messages {
		blockType := BlockHistory
		priority := DefaultPriority(BlockHistory)
		required := false
		if index == last {
			blockType = BlockTask
			priority = DefaultPriority(BlockTask)
			required = true
		}
		blocks = append(blocks, ContextBlock{
			ID: fmt.Sprintf("input-%03d", index+1), Type: blockType, Source: "skill_input", Role: message.Role,
			Priority: priority, Required: required, Freshness: FreshnessUnknown, Content: message.Content,
		})
	}
	return blocks
}

func RuntimeMessageBlock(id string, message llm.Message) ContextBlock {
	blockType := BlockHistory
	priority := 450
	required := false
	content := strings.TrimSpace(message.Content)
	if strings.HasPrefix(content, "AGENT_FEEDBACK\n") {
		blockType = BlockTask
		priority = 950
		required = true
	} else if strings.HasPrefix(content, "TOOL_RESULT\n") {
		blockType = BlockTool
		priority = 650
	}
	return ContextBlock{ID: id, Type: blockType, Source: "runtime", Role: message.Role, Priority: priority, Required: required, Freshness: FreshnessUnknown, Content: message.Content}
}
