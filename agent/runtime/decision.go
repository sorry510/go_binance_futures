package runtime

import (
	"encoding/json"
	"fmt"
	"strings"
)

type decision struct {
	Action    string          `json:"action"`
	Summary   string          `json:"summary,omitempty"`
	Tool      string          `json:"tool,omitempty"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     string          `json:"error,omitempty"`
}

func parseDecision(content string) (decision, error) {
	var parsed decision
	payload := extractJSONObject(content)
	if payload == "" {
		return parsed, fmt.Errorf("agent response is not a JSON object")
	}
	if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
		return parsed, fmt.Errorf("decode agent decision: %w", err)
	}
	parsed.Action = strings.ToLower(strings.TrimSpace(parsed.Action))
	parsed.Tool = strings.TrimSpace(parsed.Tool)
	switch parsed.Action {
	case "tool":
		if parsed.Tool == "" {
			return parsed, fmt.Errorf("tool decision requires tool")
		}
	case "final":
		if len(parsed.Result) == 0 || string(parsed.Result) == "null" {
			return parsed, fmt.Errorf("final decision requires result")
		}
	case "error":
		if strings.TrimSpace(parsed.Error) == "" {
			return parsed, fmt.Errorf("error decision requires error message")
		}
	default:
		return parsed, fmt.Errorf("unsupported agent action %q", parsed.Action)
	}
	return parsed, nil
}
func extractJSONObject(content string) string {
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "```") {
		lines := strings.Split(trimmed, "\n")
		if len(lines) >= 3 {
			lines = lines[1:]
			if strings.TrimSpace(lines[len(lines)-1]) == "```" {
				lines = lines[:len(lines)-1]
			}
			trimmed = strings.TrimSpace(strings.Join(lines, "\n"))
		}
	}
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start < 0 || end < start {
		return ""
	}
	return strings.TrimSpace(trimmed[start : end+1])
}
