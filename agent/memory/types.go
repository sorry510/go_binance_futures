package memory

import "time"

type Type string

const (
	TypeUserPreference   Type = "user_preference"
	TypeStrategyFact     Type = "strategy_fact"
	TypeMarketHypothesis Type = "market_hypothesis"
	TypeTaskSummary      Type = "task_summary"
	TypeLesson           Type = "lesson"
)

const (
	StatusCandidate = "candidate"
	StatusActive    = "active"
	StatusDisabled  = "disabled"
	StatusExpired   = "expired"
)

const DefaultUserScope = "local"

var DefaultMarketHypothesisTTL = 6 * time.Hour
var DefaultTaskSummaryTTL = 30 * 24 * time.Hour

type Scope struct {
	User     string `json:"user,omitempty"`
	Skill    string `json:"skill,omitempty"`
	Symbol   string `json:"symbol,omitempty"`
	Strategy string `json:"strategy,omitempty"`
}

type Memory struct {
	ID           int64      `json:"id"`
	Type         Type       `json:"type"`
	Scope        Scope      `json:"scope"`
	SourceTaskID string     `json:"source_task_id,omitempty"`
	Confidence   float64    `json:"confidence"`
	Status       string     `json:"status"`
	Content      string     `json:"content"`
	ContentHash  string     `json:"content_hash"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
}

type CreateInput struct {
	Type         Type    `json:"type"`
	Scope        Scope   `json:"scope"`
	SourceTaskID string  `json:"source_task_id,omitempty"`
	Confidence   float64 `json:"confidence"`
	Status       string  `json:"status,omitempty"`
	Content      string  `json:"content"`
	ExpiresAt    int64   `json:"expires_at,omitempty"`
}

type UpdateInput struct {
	Scope      Scope   `json:"scope"`
	Confidence float64 `json:"confidence"`
	Content    string  `json:"content"`
	ExpiresAt  int64   `json:"expires_at,omitempty"`
}

type ListOptions struct {
	Type           string
	Status         string
	User           string
	Skill          string
	Symbol         string
	Strategy       string
	SourceTaskID   string
	IncludeExpired bool
	Page           int
	Limit          int
}

type ListResult struct {
	Page  int      `json:"page"`
	Limit int      `json:"limit"`
	Total int64    `json:"total"`
	List  []Memory `json:"list"`
}

type QueryScope struct {
	User     string
	Skill    string
	Symbol   string
	Strategy string
	Limit    int
	Now      time.Time
}
