package symbolteam

import "encoding/json"

const (
	TechnicalSkillName  = "symbol_team_technical"
	FlowSkillName       = "symbol_team_flow"
	NewsSkillName       = "symbol_team_news"
	SupervisorSkillName = "symbol_team_supervisor"
)

const (
	RoleTechnical        = "technical_analyst"
	RoleFlow             = "flow_analyst"
	RoleNews             = "news_analyst"
	RoleStrategyReviewer = "strategy_reviewer"
	RoleSupervisor       = "supervisor"
)

type Evidence struct {
	Source  string `json:"source"`
	Finding string `json:"finding"`
}

type SharedInput struct {
	Symbol            string          `json:"symbol"`
	Prompt            string          `json:"prompt,omitempty"`
	SharedContext     json.RawMessage `json:"shared_context"`
	SharedContextHash string          `json:"shared_context_hash,omitempty"`
}

type KeyLevel struct {
	Kind  string  `json:"kind"`
	Price float64 `json:"price"`
}

type TechnicalAnalysisV1 struct {
	Version     string     `json:"version"`
	Symbol      string     `json:"symbol"`
	AsOf        string     `json:"as_of"`
	Trend       string     `json:"trend"`
	Structure   string     `json:"structure"`
	Volatility  string     `json:"volatility"`
	KeyLevels   []KeyLevel `json:"key_levels"`
	Confidence  float64    `json:"confidence"`
	DataMissing []string   `json:"data_missing"`
	Evidence    []Evidence `json:"evidence"`
}

type FlowAnalysisV1 struct {
	Version      string     `json:"version"`
	Symbol       string     `json:"symbol"`
	AsOf         string     `json:"as_of"`
	Bias         string     `json:"bias"`
	Funding      string     `json:"funding"`
	OpenInterest string     `json:"open_interest"`
	Taker        string     `json:"taker"`
	Depth        string     `json:"depth"`
	Liquidation  string     `json:"liquidation"`
	Confidence   float64    `json:"confidence"`
	DataMissing  []string   `json:"data_missing"`
	Evidence     []Evidence `json:"evidence"`
}

type NewsAnalysisV1 struct {
	Version     string     `json:"version"`
	Symbol      string     `json:"symbol"`
	AsOf        string     `json:"as_of"`
	Bias        string     `json:"bias"`
	Impact      string     `json:"impact"`
	Summary     string     `json:"summary"`
	Confidence  float64    `json:"confidence"`
	DataMissing []string   `json:"data_missing"`
	Evidence    []Evidence `json:"evidence"`
}

type MemberInput struct {
	Role        string          `json:"role"`
	TaskID      string          `json:"task_id,omitempty"`
	Status      string          `json:"status"`
	Result      json.RawMessage `json:"result,omitempty"`
	DataMissing []string        `json:"data_missing"`
}

type SupervisorInput struct {
	Symbol    string      `json:"symbol"`
	Prompt    string      `json:"prompt,omitempty"`
	Technical MemberInput `json:"technical"`
	Flow      MemberInput `json:"flow"`
	News      MemberInput `json:"news"`
}

type SupervisorEvidence struct {
	Role    string `json:"role"`
	Source  string `json:"source"`
	Finding string `json:"finding"`
}

type SupervisorResultV1 struct {
	Version       string               `json:"version"`
	Symbol        string               `json:"symbol"`
	AsOf          string               `json:"as_of"`
	Direction     string               `json:"direction"`
	Confidence    float64              `json:"confidence"`
	Summary       string               `json:"summary"`
	Consensus     []string             `json:"consensus"`
	Disagreements []string             `json:"disagreements"`
	DataMissing   []string             `json:"data_missing"`
	Evidence      []SupervisorEvidence `json:"evidence"`
}
