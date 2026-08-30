package security

import (
	"strings"
	"testing"
)

func TestRedactTextAndPayload(t *testing.T) {
	text := RedactText(`authorization=secret-token Bearer abc123 password=hunter2 database_password=db-pass postgres://agent:uri-pass@localhost/app agent:mysql-pass@tcp(localhost:3306)/app {"api_key":"json-key","client_secret":"json-secret"}`)
	for _, secret := range []string{"secret-token", "abc123", "hunter2", "db-pass", "uri-pass", "mysql-pass", "json-key", "json-secret"} {
		if strings.Contains(text, secret) {
			t.Fatalf("secret %q leaked in text: %s", secret, text)
		}
	}
	payload := RedactPayload(`{"api_key":"key-1","nested":{"refresh_token":"token-2"},"prompt":"authorization=prompt-secret","error":"postgres://agent:uri-secret@localhost/app","symbol":"BTCUSDT"}`)
	for _, secret := range []string{"key-1", "token-2", "prompt-secret", "uri-secret"} {
		if strings.Contains(payload, secret) {
			t.Fatalf("secret %q leaked in payload: %s", secret, payload)
		}
	}
	if !strings.Contains(payload, "BTCUSDT") || !strings.Contains(payload, "[REDACTED]") {
		t.Fatalf("unexpected redacted payload: %s", payload)
	}
}
