package controllers

import (
	"math"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	loginMaxAttempts   = 5
	loginAttemptWindow = 15 * time.Minute
	loginLockDuration  = 15 * time.Minute
	loginMaxIPs        = 10000
)

type loginAttempt struct {
	count        int
	windowStart  time.Time
	blockedUntil time.Time
}

// loginLimiter is deliberately process-local: it needs no database, Redis, or
// additional goroutine. An attempt is reserved BEFORE checking credentials so
// a burst of parallel requests cannot bypass the limit.
type loginLimiter struct {
	mu          sync.Mutex
	attempts    map[string]loginAttempt
	lastCleanup time.Time
}

var webLoginLimiter = &loginLimiter{attempts: make(map[string]loginAttempt)}

func (l *loginLimiter) cleanup(now time.Time) {
	if !l.lastCleanup.IsZero() && now.Sub(l.lastCleanup) < time.Minute {
		return
	}
	for ip, attempt := range l.attempts {
		if !now.Before(attempt.blockedUntil) &&
			!now.Before(attempt.windowStart.Add(loginAttemptWindow)) {
			delete(l.attempts, ip)
		}
	}
	l.lastCleanup = now
}

func (l *loginLimiter) reserve(ip string, now time.Time) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cleanup(now)

	attempt, exists := l.attempts[ip]
	if now.Before(attempt.blockedUntil) {
		return attempt.blockedUntil.Sub(now)
	}
	if !exists && len(l.attempts) >= loginMaxIPs {
		// Fail closed rather than allow unlimited attempts from rotating IPs.
		return time.Minute
	}
	if attempt.windowStart.IsZero() || !now.Before(attempt.windowStart.Add(loginAttemptWindow)) {
		attempt = loginAttempt{windowStart: now}
	}
	if attempt.count >= loginMaxAttempts {
		attempt.blockedUntil = now.Add(loginLockDuration)
		l.attempts[ip] = attempt
		return loginLockDuration
	}
	attempt.count++
	l.attempts[ip] = attempt
	return 0
}

func (l *loginLimiter) failed(ip string, now time.Time) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	attempt, ok := l.attempts[ip]
	if !ok || attempt.count < loginMaxAttempts {
		return 0
	}
	if attempt.blockedUntil.IsZero() {
		attempt.blockedUntil = now.Add(loginLockDuration)
		l.attempts[ip] = attempt
	}
	return attempt.blockedUntil.Sub(now)
}

func (l *loginLimiter) succeeded(ip string) {
	l.mu.Lock()
	delete(l.attempts, ip)
	l.mu.Unlock()
}

// Never trust a user-supplied X-Forwarded-For or X-Real-IP. If a trusted
// reverse proxy is used, per-client limits belong at that proxy boundary.
func loginRemoteIP(r *http.Request) string {
	remote := strings.TrimSpace(r.RemoteAddr)
	host, _, err := net.SplitHostPort(remote)
	if err == nil {
		remote = host
	}
	if ip := net.ParseIP(remote); ip != nil {
		return ip.String()
	}
	return "unknown"
}

func retryAfterSeconds(wait time.Duration) int {
	if wait <= 0 {
		return 1
	}
	return int(math.Ceil(wait.Seconds()))
}
