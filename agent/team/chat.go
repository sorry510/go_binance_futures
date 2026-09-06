package team

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var chatSymbolPattern = regexp.MustCompile(`(?i)([A-Z0-9]{2,20}USDT)\b`)

// BuildChatInput converts a chat message into the fixed Team input contract.
// Symbol selection stays deterministic: explicit UI selection, literal XXXUSDT
// in the message, or a previous Team input for continuation-style prompts.
func BuildChatInput(content string, previousInputs []string, explicitSymbol string) (Input, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return Input{}, fmt.Errorf("chat content is required")
	}
	symbol := strings.ToUpper(strings.TrimSpace(explicitSymbol))
	if symbol != "" && !strings.HasSuffix(symbol, "USDT") {
		return Input{}, fmt.Errorf("selected symbol must be a USDT futures contract")
	}
	if symbol == "" {
		if match := chatSymbolPattern.FindStringSubmatch(content); len(match) >= 2 {
			symbol = strings.ToUpper(match[1])
		}
	}
	if symbol == "" && shouldReusePreviousSymbol(content) {
		for _, raw := range previousInputs {
			var previous Input
			if json.Unmarshal([]byte(raw), &previous) == nil {
				previous.Symbol = strings.ToUpper(strings.TrimSpace(previous.Symbol))
				if strings.HasSuffix(previous.Symbol, "USDT") {
					symbol = previous.Symbol
					break
				}
			}
		}
	}
	if symbol == "" {
		return Input{}, fmt.Errorf("请选择 USDT 合约，或在消息中明确指定，例如 BTCUSDT")
	}
	return Input{Symbol: symbol, Prompt: content}, nil
}

func shouldReusePreviousSymbol(content string) bool {
	normalized := strings.ToLower(strings.TrimSpace(content))
	for _, marker := range []string{"刚才", "之前", "上面", "这个币", "该币", "它", "继续", "刚刚", "previous", "above", "this coin", "continue"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}
