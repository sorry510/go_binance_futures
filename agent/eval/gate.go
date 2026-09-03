package eval

import "fmt"

type GatePolicy struct {
	MinimumScore       float64 `json:"minimum_score"`
	MaxScoreRegression float64 `json:"max_score_regression"`
}

type GateResult struct {
	Passed   bool     `json:"passed"`
	Failures []string `json:"failures,omitempty"`
}

func Gate(reports []Report, comparisons []Comparison, policy GatePolicy) GateResult {
	if policy.MinimumScore <= 0 {
		policy.MinimumScore = 80
	}
	if policy.MaxScoreRegression <= 0 {
		policy.MaxScoreRegression = 5
	}
	result := GateResult{Passed: true}
	for _, report := range reports {
		if !report.Passed || report.Score < policy.MinimumScore || len(report.CriticalFailures) > 0 {
			result.Passed = false
			result.Failures = append(result.Failures, fmt.Sprintf("%s score=%.2f critical=%v", report.CaseName, report.Score, report.CriticalFailures))
		}
	}
	for _, comparison := range comparisons {
		if comparison.ScoreDelta < -policy.MaxScoreRegression {
			result.Passed = false
			result.Failures = append(result.Failures, fmt.Sprintf("%s regressed %.2f points", comparison.CaseName, -comparison.ScoreDelta))
		}
	}
	return result
}

func RequireGate(reports []Report, comparisons []Comparison, policy GatePolicy) error {
	result := Gate(reports, comparisons, policy)
	if result.Passed {
		return nil
	}
	return fmt.Errorf("agent eval gate failed: %v", result.Failures)
}
