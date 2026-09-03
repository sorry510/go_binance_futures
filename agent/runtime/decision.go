package runtime

import (
	"encoding/json"
	"fmt"
	"strings"
)

type toolDecision struct {
	Tool      string          `json:"tool"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

type decision struct {
	Action    string          `json:"action"`
	Summary   string          `json:"summary,omitempty"`
	Tool      string          `json:"tool,omitempty"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
	Tools     []toolDecision  `json:"tools,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     string          `json:"error,omitempty"`
}

func parseDecision(content string) (decision, error) {
	var parsed decision
	payload := extractJSONObject(content)
	if payload == "" {
		return parsed, fmt.Errorf("agent response is not a JSON object")
	}
	if err := unmarshalDecision(payload, &parsed); err != nil {
		return parsed, fmt.Errorf("decode agent decision: %w", err)
	}
	parsed.Action = strings.ToLower(strings.TrimSpace(parsed.Action))
	parsed.Tool = strings.TrimSpace(parsed.Tool)
	switch parsed.Action {
	case "tool":
		if parsed.Tool == "" {
			return parsed, fmt.Errorf("tool decision requires tool")
		}
	case "parallel_tools":
		if len(parsed.Tools) < 2 {
			return parsed, fmt.Errorf("parallel_tools decision requires at least two tools")
		}
		for index := range parsed.Tools {
			parsed.Tools[index].Tool = strings.TrimSpace(parsed.Tools[index].Tool)
			if parsed.Tools[index].Tool == "" {
				return parsed, fmt.Errorf("parallel_tools[%d] requires tool", index)
			}
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

func unmarshalDecision(payload string, parsed *decision) error {
	err := json.Unmarshal([]byte(payload), parsed)
	if err == nil {
		return nil
	}

	// Some models repeatedly reverse or duplicate the object/array terminators
	// after the last array element (for example, `..."supertrend":[{...}}]`).
	// Only accept a repair when the decoder points at that exact byte and one
	// unambiguous single-token correction makes the complete decision valid
	// JSON. Every other malformed response remains a protocol error.
	syntaxError, ok := err.(*json.SyntaxError)
	if !ok || !strings.Contains(syntaxError.Error(), "invalid character '}' after array element") {
		return err
	}
	offset := int(syntaxError.Offset) - 1
	if offset < 0 || offset >= len(payload) || payload[offset] != '}' {
		return err
	}
	candidates := []string{payload[:offset] + payload[offset+1:]}
	if offset+1 < len(payload) && payload[offset+1] == ']' {
		candidates = append(candidates, payload[:offset]+"]}"+payload[offset+2:])
	}

	valid := make([]decision, 0, 1)
	for _, candidate := range candidates {
		var repaired decision
		if repairErr := json.Unmarshal([]byte(candidate), &repaired); repairErr == nil {
			valid = append(valid, repaired)
		}
	}
	if len(valid) != 1 {
		return err
	}
	*parsed = valid[0]
	return nil
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
