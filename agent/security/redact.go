package security

import (
	"encoding/json"
	"regexp"
	"strings"
)

var sensitiveTextPattern = regexp.MustCompile(`(?i)(api[_-]?key|authorization|password|db[_-]?password|database[_-]?password|access[_-]?token|refresh[_-]?token|client[_-]?secret|token|secret|dsn)(["\']?\s*[:=]\s*["\']?)([^"\'\s,;}]+)`)
var bearerPattern = regexp.MustCompile(`(?i)bearer\s+[^\s,;]+`)
var uriCredentialPattern = regexp.MustCompile(`(?i)([a-z][a-z0-9+.-]*://[^:/\s]+:)([^@\s]+)(@)`)
var mysqlCredentialPattern = regexp.MustCompile(`([^:\s/@]+:)([^@\s]+)(@(?:tcp|unix)\()`)

func RedactText(value string) string {
	value = sensitiveTextPattern.ReplaceAllString(value, "$1$2[REDACTED]")
	value = bearerPattern.ReplaceAllString(value, "Bearer [REDACTED]")
	value = uriCredentialPattern.ReplaceAllString(value, "$1[REDACTED]$3")
	return mysqlCredentialPattern.ReplaceAllString(value, "$1[REDACTED]$3")
}

func RedactPayload(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var decoded any
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return RedactText(value)
	}
	decoded = redactValue(decoded)
	encoded, err := json.Marshal(decoded)
	if err != nil {
		return RedactText(value)
	}
	return string(encoded)
}
func redactValue(value any) any {
	switch current := value.(type) {
	case map[string]any:
		for key, child := range current {
			if IsSensitiveKey(key) {
				current[key] = "[REDACTED]"
				continue
			}
			current[key] = redactValue(child)
		}
		return current
	case []any:
		for index, child := range current {
			current[index] = redactValue(child)
		}
		return current
	case string:
		return RedactText(current)
	default:
		return current
	}
}

func IsSensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.NewReplacer("_", "", "-", "").Replace(strings.TrimSpace(key)))
	switch normalized {
	case "apikey", "authorization", "password", "dbpassword", "databasepassword", "dsn", "token", "accesstoken", "refreshtoken", "secret", "clientsecret":
		return true
	default:
		return false
	}
}
