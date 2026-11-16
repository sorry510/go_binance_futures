package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	beegocontext "github.com/beego/beego/v2/server/web/context"
)

func newTestLimiter() *loginLimiter {
	return &loginLimiter{attempts: make(map[string]loginAttempt)}
}

func TestLoginLimiterFifthFailedAttemptBlocks(t *testing.T) {
	l := newTestLimiter()
	start := time.Unix(1800000000, 0)
	for i := 0; i < loginMaxAttempts; i++ {
		if wait := l.reserve("192.0.2.10", start.Add(time.Duration(i)*time.Second)); wait != 0 {
			t.Fatalf("attempt %d unexpectedly limited: %v", i+1, wait)
		}
		wait := l.failed("192.0.2.10", start.Add(time.Duration(i)*time.Second))
		if i < loginMaxAttempts-1 && wait != 0 {
			t.Fatalf("blocked early on attempt %d: %v", i+1, wait)
		}
		if i == loginMaxAttempts-1 && wait != loginLockDuration {
			t.Fatalf("fifth failure did not lock for %v, got %v", loginLockDuration, wait)
		}
	}
	if wait := l.reserve("192.0.2.10", start.Add(20*time.Second)); wait <= 0 {
		t.Fatal("blocked IP was allowed")
	}
	if wait := l.reserve("192.0.2.11", start.Add(20*time.Second)); wait != 0 {
		t.Fatal("independent IP was affected")
	}
	if wait := l.reserve("192.0.2.10", start.Add(4*time.Second).Add(loginLockDuration)); wait != 0 {
		t.Fatalf("IP did not recover after lock expiration: %v", wait)
	}
}

func TestLoginLimiterSuccessClearsFailures(t *testing.T) {
	l := newTestLimiter()
	start := time.Unix(1800000000, 0)
	for i := 0; i < 4; i++ {
		if wait := l.reserve("198.51.100.2", start); wait != 0 {
			t.Fatal("unexpected lock")
		}
		if wait := l.failed("198.51.100.2", start); wait != 0 {
			t.Fatal("unexpected early lock")
		}
	}
	if wait := l.reserve("198.51.100.2", start); wait != 0 {
		t.Fatal("correct fifth login was blocked")
	}
	l.succeeded("198.51.100.2")
	if wait := l.reserve("198.51.100.2", start); wait != 0 {
		t.Fatal("successful login did not clear attempts")
	}
	if wait := l.failed("198.51.100.2", start); wait != 0 {
		t.Fatal("unexpected lock after success")
	}
}

func TestLoginLimiterWindowAndParallelRequests(t *testing.T) {
	l := newTestLimiter()
	at := time.Unix(1800000000, 0)
	for i := 0; i < loginMaxAttempts-1; i++ {
		_ = l.reserve("203.0.113.10", at)
		_ = l.failed("203.0.113.10", at)
	}
	if wait := l.reserve("203.0.113.10", at.Add(loginAttemptWindow)); wait != 0 {
		t.Fatalf("attempt window did not reset: %v", wait)
	}

	l = newTestLimiter()
	const concurrent = 50
	var wg sync.WaitGroup
	allowed := make(chan bool, concurrent)
	for i := 0; i < concurrent; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allowed <- l.reserve("203.0.113.11", at) == 0
		}()
	}
	wg.Wait()
	close(allowed)
	n := 0
	for ok := range allowed {
		if ok {
			n++
		}
	}
	if n != loginMaxAttempts {
		t.Fatalf("parallel requests allowed %d login attempts, want %d", n, loginMaxAttempts)
	}
}

func TestLoginRemoteIPDoesNotTrustForwardedHeaders(t *testing.T) {
	for _, tc := range []struct{ remote, expected string }{
		{"198.51.100.5:12345", "198.51.100.5"},
		{"[2001:db8::5]:12345", "2001:db8::5"},
		{"garbage", "unknown"},
	} {
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = tc.remote
		req.Header.Set("X-Forwarded-For", "203.0.113.7")
		req.Header.Set("X-Real-IP", "203.0.113.8")
		if got := loginRemoteIP(req); got != tc.expected {
			t.Fatalf("remote %q -> %q, want %q", tc.remote, got, tc.expected)
		}
	}
}

func TestLoginLimiterMaxEntriesFailsClosed(t *testing.T) {
	l := newTestLimiter()
	at := time.Unix(1800000000, 0)
	for i := 0; i < loginMaxIPs; i++ {
		l.attempts[strings.Repeat("x", 5)+string(rune(i+1000))] = loginAttempt{count: 1, windowStart: at}
	}
	l.lastCleanup = at
	if got := l.reserve("new-ip", at); got != time.Minute {
		t.Fatalf("full limiter should fail closed, got %v", got)
	}
}

func loginRequestForTest(remote, body string) (*httptest.ResponseRecorder, map[string]interface{}) {
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
	req.RemoteAddr = remote
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	ctx := beegocontext.NewContext()
	ctx.Reset(rec, req)
	ctx.Input.RequestBody = []byte(body)
	ctrl := new(LoginController)
	ctrl.Init(ctx, "LoginController", "Post", nil)
	ctrl.Post()
	var payload map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &payload)
	return rec, payload
}

func TestLoginControllerHTTP429AndRecovery(t *testing.T) {
	oldUser, oldPass, oldLimiter := username, password, webLoginLimiter
	username, password = "login_test_operator", "correct_test_password"
	webLoginLimiter = newTestLimiter()
	defer func() {
		username, password, webLoginLimiter = oldUser, oldPass, oldLimiter
	}()

	badBytes, _ := json.Marshal(User{UserName: "login_test_operator", Password: "wrong"})
	goodBytes, _ := json.Marshal(User{UserName: "login_test_operator", Password: "correct_test_password"})
	bad, good := string(badBytes), string(goodBytes)
	for i := 0; i < loginMaxAttempts; i++ {
		rec, payload := loginRequestForTest("192.0.2.9:4321", bad)
		if i < loginMaxAttempts-1 && (rec.Code != http.StatusOK || payload["code"] != float64(101)) {
			t.Fatalf("failure %d: HTTP %d, body=%s", i+1, rec.Code, rec.Body.String())
		}
		if i == loginMaxAttempts-1 {
			if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") == "" ||
				payload["code"] != float64(http.StatusTooManyRequests) {
				t.Fatalf("fifth failure not throttled: HTTP %d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
			}
		}
	}
	req, blocked := loginRequestForTest("192.0.2.9:9999", good)
	if req.Code != http.StatusTooManyRequests || blocked["code"] != float64(http.StatusTooManyRequests) {
		t.Fatalf("locked IP logged in with correct password: HTTP=%d", req.Code)
	}
	rec, payload := loginRequestForTest("192.0.2.10:4321", good)
	if rec.Code != http.StatusOK || payload["code"] != float64(200) {
		t.Fatalf("other IP could not login: HTTP=%d body=%s", rec.Code, rec.Body.String())
	}
}

// Invalid JSON is also a failed login attempt; otherwise an attacker could
// bypass the limiter while forcing repeated JSON parsing.
func TestLoginControllerMalformedRequestsAlsoCount(t *testing.T) {
	oldLimiter := webLoginLimiter
	webLoginLimiter = newTestLimiter()
	defer func() { webLoginLimiter = oldLimiter }()
	for i := 0; i < loginMaxAttempts; i++ {
		rec, payload := loginRequestForTest("203.0.113.3:8080", "{invalid")
		if i < loginMaxAttempts-1 && (rec.Code != http.StatusOK || payload["code"] != float64(101)) {
			t.Fatalf("bad JSON attempt %d: HTTP=%d body=%s", i+1, rec.Code, rec.Body.String())
		}
		if i == loginMaxAttempts-1 && rec.Code != http.StatusTooManyRequests {
			t.Fatalf("malformed JSON did not trigger limit: HTTP=%d body=%s", rec.Code, rec.Body.String())
		}
	}
}

func TestLoginControllerSuccessResetsPriorFailures(t *testing.T) {
	oldUser, oldPass, oldLimiter := username, password, webLoginLimiter
	username, password = "operator_test", "correct_test_password"
	webLoginLimiter = newTestLimiter()
	defer func() { username, password, webLoginLimiter = oldUser, oldPass, oldLimiter }()
	good, _ := json.Marshal(User{UserName: username, Password: password})
	bad, _ := json.Marshal(User{UserName: username, Password: "bad"})
	for i := 0; i < 4; i++ {
		rec, _ := loginRequestForTest("198.51.100.25:1234", string(bad))
		if rec.Code != http.StatusOK {
			t.Fatalf("early limit: %d", rec.Code)
		}
	}
	rec, payload := loginRequestForTest("198.51.100.25:1234", string(good))
	if rec.Code != http.StatusOK || payload["code"] != float64(200) {
		t.Fatalf("fifth valid login not allowed: %d", rec.Code)
	}
	rec, payload = loginRequestForTest("198.51.100.25:1234", string(bad))
	if rec.Code != http.StatusOK || payload["code"] != float64(101) {
		t.Fatalf("successful login did not clear failures: %d", rec.Code)
	}
}
