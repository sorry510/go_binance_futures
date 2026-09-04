package mcpclient

import (
	"context"
	"encoding/json"
	"strings"

	"go_binance_futures/agent/security"
	"go_binance_futures/models"
)

func (g *Gateway) credentialRedactions(ctx context.Context, server models.AgentMCPServer) ([]string, error) {
	if server.AuthType == AuthNone || strings.TrimSpace(server.SecretRef) == "" {
		return nil, nil
	}
	resolver := g.ResolveSecret
	if resolver == nil {
		resolver = ResolveEnvironmentSecret
	}
	raw, err := resolver(ctx, server.SecretRef)
	if err != nil {
		return nil, err
	}
	if server.AuthType != AuthOAuth2 {
		return compactCredentialValues([]string{raw}), nil
	}
	var credential OAuthCredential
	if err := json.Unmarshal([]byte(raw), &credential); err != nil {
		return nil, err
	}
	return compactCredentialValues([]string{
		credential.AccessToken,
		credential.RefreshToken,
		credential.ClientSecret,
	}), nil
}

func compactCredentialValues(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}

func redactCredentialText(value string, credentials []string) string {
	for _, credential := range credentials {
		if len(credential) >= 6 {
			value = strings.ReplaceAll(value, credential, "[REDACTED]")
		} else if value == credential {
			value = "[REDACTED]"
		}
	}
	return security.RedactText(value)
}

func redactCredentialAny(value any, credentials []string) any {
	switch current := value.(type) {
	case map[string]any:
		for key, child := range current {
			current[key] = redactCredentialAny(child, credentials)
		}
		return current
	case []any:
		for index, child := range current {
			current[index] = redactCredentialAny(child, credentials)
		}
		return current
	case string:
		return redactCredentialText(current, credentials)
	default:
		return current
	}
}

func redactCredentialObject(value any, credentials []string) error {
	if len(credentials) == 0 || value == nil {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return err
	}
	decoded = redactCredentialAny(decoded, credentials)
	safeRaw, err := json.Marshal(decoded)
	if err != nil {
		return err
	}
	return json.Unmarshal(safeRaw, value)
}

type credentialRedactedError struct {
	message string
	cause   error
}

func (err credentialRedactedError) Error() string { return err.message }
func (err credentialRedactedError) Unwrap() error { return err.cause }

func redactCredentialError(err error, credentials []string) error {
	if err == nil {
		return nil
	}
	return credentialRedactedError{message: redactCredentialText(err.Error(), credentials), cause: err}
}
