package governance

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type Limits struct {
	PerMinute int `json:"per_minute"`
	PerHour   int `json:"per_hour"`
}

type Provider func() Limits

type SkillStatus struct {
	Accepted uint64 `json:"accepted"`
	Rejected uint64 `json:"rejected"`
}

type Status struct {
	Limits        Limits                 `json:"limits"`
	RecentMinute int                    `json:"recent_minute"`
	RecentHour   int                    `json:"recent_hour"`
	Accepted     uint64                 `json:"accepted"`
	Rejected     uint64                 `json:"rejected"`
	Skills       map[string]SkillStatus `json:"skills"`
}
type Limiter struct {
	provider Provider
	mu       sync.Mutex
	calls    []int64
	accepted uint64
	rejected uint64
	skills   map[string]SkillStatus
}

func New(provider Provider) *Limiter {
	if provider == nil {
		provider = func() Limits { return Limits{PerMinute: 30, PerHour: 300} }
	}
	return &Limiter{provider: provider, skills: map[string]SkillStatus{}}
}

func (limiter *Limiter) Admit(skill string) error {
	if limiter == nil {
		return nil
	}
	skill = strings.TrimSpace(skill)
	limits := normalize(limiter.provider())
	now := time.Now().UnixMilli()
	minuteCutoff := now - time.Minute.Milliseconds()
	hourCutoff := now - time.Hour.Milliseconds()

	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	limiter.prune(hourCutoff)
	recentMinute := 0
	for _, value := range limiter.calls {
		if value > minuteCutoff {
			recentMinute++
		}
	}
	if recentMinute >= limits.PerMinute || len(limiter.calls) >= limits.PerHour {
		limiter.rejected++
		status := limiter.skills[skill]
		status.Rejected++
		limiter.skills[skill] = status
		return fmt.Errorf("agent start budget exceeded: minute=%d/%d hour=%d/%d", recentMinute, limits.PerMinute, len(limiter.calls), limits.PerHour)
	}
	limiter.calls = append(limiter.calls, now)
	limiter.accepted++
	status := limiter.skills[skill]
	status.Accepted++
	limiter.skills[skill] = status
	return nil
}

func (limiter *Limiter) Status() Status {
	if limiter == nil {
		return Status{}
	}
	limits := normalize(limiter.provider())
	now := time.Now().UnixMilli()
	minuteCutoff := now - time.Minute.Milliseconds()
	hourCutoff := now - time.Hour.Milliseconds()
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	limiter.prune(hourCutoff)
	recentMinute := 0
	for _, value := range limiter.calls {
		if value > minuteCutoff {
			recentMinute++
		}
	}
	skills := make(map[string]SkillStatus, len(limiter.skills))
	for name, value := range limiter.skills {
		skills[name] = value
	}
	return Status{
		Limits: limits, RecentMinute: recentMinute, RecentHour: len(limiter.calls),
		Accepted: limiter.accepted, Rejected: limiter.rejected, Skills: skills,
	}
}

func (limiter *Limiter) prune(cutoff int64) {
	kept := limiter.calls[:0]
	for _, value := range limiter.calls {
		if value > cutoff {
			kept = append(kept, value)
		}
	}
	limiter.calls = kept
}
func normalize(value Limits) Limits {
	if value.PerMinute <= 0 {
		value.PerMinute = 30
	}
	if value.PerHour <= 0 {
		value.PerHour = 300
	}
	if value.PerHour < value.PerMinute {
		value.PerHour = value.PerMinute
	}
	return value
}
