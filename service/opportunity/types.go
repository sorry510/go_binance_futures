package opportunity

import "go_binance_futures/models"

const (
	StatusNew      = "new"
	StatusReviewed = "reviewed"
	StatusExpired  = "expired"
)

const (
	AnalysisPending     = "pending"
	AnalysisSucceeded   = "succeeded"
	AnalysisPartial     = "partial"
	AnalysisDataMissing = "data_missing"
	AnalysisFailed      = "failed"
)

const (
	DirectionLong    = "long"
	DirectionShort   = "short"
	DirectionNeutral = "neutral"
)

type CreateInput struct {
	Symbol          string
	Direction       string
	SourceType      string
	SourceID        string
	AnalysisTaskID  string
	Summary         string
	Confidence      float64
	MarketCondition int
	AnalysisStatus  string
	AnalysisError   string
	ExpiresAt       int64
}

type AnalysisUpdate struct {
	Direction       string
	TaskID          string
	Summary         string
	Confidence      float64
	MarketCondition int
	Status          string
	Error           string
}

type ListOptions struct {
	Status         string
	AnalysisStatus string
	Symbol         string
	SourceType     string
	Page           int
	Limit          int
}

type ListResult struct {
	Page  int                       `json:"page"`
	Limit int                       `json:"limit"`
	Total int64                     `json:"total"`
	List  []models.AgentOpportunity `json:"list"`
}
