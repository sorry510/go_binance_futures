package systemhealth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go_binance_futures/agent/scheduler"
	"go_binance_futures/appversion"
	"go_binance_futures/binanceproxy"
	"go_binance_futures/models"
	"go_binance_futures/service/binanceapiusage"
	marketintelligence "go_binance_futures/service/marketintelligence"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/config"
)

const (
	StatusHealthy  = "healthy"
	StatusWarning  = "warning"
	StatusError    = "error"
	StatusDisabled = "disabled"
	StatusUnknown  = "unknown"
)

type Check struct {
	Status        string `json:"status"`
	Message       string `json:"message,omitempty"`
	LastSuccessAt int64  `json:"last_success_at,omitempty"`
	LastErrorAt   int64  `json:"last_error_at,omitempty"`
	LastError     string `json:"last_error,omitempty"`
	Count         int64  `json:"count,omitempty"`
}

type DatabaseCheck struct {
	Check
	Version         int64 `json:"version"`
	RequiredVersion int64 `json:"required_version"`
}

type MCPServerStatus struct {
	Name          string `json:"name"`
	Status        string `json:"status"`
	LastSuccessAt int64  `json:"last_success_at,omitempty"`
	LastErrorAt   int64  `json:"last_error_at,omitempty"`
	LastError     string `json:"last_error,omitempty"`
}

type SchedulerJobStatus struct {
	Name            string `json:"name"`
	Skill           string `json:"skill"`
	Enabled         bool   `json:"enabled"`
	Running         bool   `json:"running"`
	IntervalSeconds int64  `json:"interval_seconds"`
	LastStatus      string `json:"last_status,omitempty"`
	LastError       string `json:"last_error,omitempty"`
	LastRunAt       int64  `json:"last_run_at,omitempty"`
	NextRunAt       int64  `json:"next_run_at,omitempty"`
	RunCount        uint64 `json:"run_count,omitempty"`
	SkipCount       uint64 `json:"skip_count,omitempty"`
}

type AgentCheck struct {
	Check
	Tasks24h        int64 `json:"tasks_24h"`
	Failed24h       int64 `json:"failed_24h"`
	MaxRoundsFailed int64 `json:"max_rounds_failed_24h"`
}

type TradeCheck struct {
	Check
	ExecutionUncertain int64 `json:"execution_uncertain"`
	ProtectionFailed   int64 `json:"protection_failed"`
	ReconcileRequired  int64 `json:"reconcile_required"`
}

type Report struct {
	GeneratedAt        int64                             `json:"generated_at"`
	Overall            string                            `json:"overall"`
	Database           DatabaseCheck                     `json:"database"`
	BinanceREST        Check                             `json:"binance_rest"`
	FuturesWS          Check                             `json:"futures_ws"`
	AnnouncementWS     Check                             `json:"announcement_ws"`
	MarketIntelligence Check                             `json:"market_intelligence"`
	MarketSources      []marketintelligence.SourceStatus `json:"market_sources"`
	MCP                Check                             `json:"mcp"`
	MCPServers         []MCPServerStatus                 `json:"mcp_servers"`
	LLM                Check                             `json:"llm"`
	Scheduler          Check                             `json:"scheduler"`
	SchedulerJobs      []SchedulerJobStatus              `json:"scheduler_jobs"`
	Agent              AgentCheck                        `json:"agent"`
	Trade              TradeCheck                        `json:"trade"`
}

type Options struct {
	Now                          time.Time
	CheckBinanceREST             bool
	BinanceBaseURL               string
	BinanceProxyURL              string
	IgnoreConfiguredBinanceProxy bool
	SchedulerRuntimeAvailable    bool
	SchedulerJobs                []scheduler.JobStatus
}

type Service struct{}

func (Service) Report(ctx context.Context, options Options) (Report, error) {
	now := options.Now
	if now.IsZero() {
		now = time.Now()
	}
	report := Report{GeneratedAt: now.UnixMilli(), Overall: StatusHealthy, MarketSources: []marketintelligence.SourceStatus{}, MCPServers: []MCPServerStatus{}, SchedulerJobs: []SchedulerJobStatus{}}
	o := orm.NewOrm()
	var one int
	if err := o.Raw("SELECT 1").QueryRow(&one); err != nil {
		report.Database = DatabaseCheck{Check: Check{Status: StatusError, Message: "database connection failed", LastError: err.Error()}, RequiredVersion: appversion.DatabaseSchemaVersion}
		report.BinanceREST = checkBinanceREST(ctx, options)
		report.markOverall()
		return report, nil
	}
	cfg, err := utils.GetSystemConfig()
	if err != nil {
		report.Database = DatabaseCheck{Check: Check{Status: StatusError, Message: "load database schema version failed", LastError: err.Error()}, RequiredVersion: appversion.DatabaseSchemaVersion}
	} else {
		status := StatusHealthy
		message := fmt.Sprintf("schema v%d", cfg.Version)
		if cfg.Version < appversion.DatabaseSchemaVersion {
			status = StatusError
			message = fmt.Sprintf("schema v%d is older than required v%d", cfg.Version, appversion.DatabaseSchemaVersion)
		}
		report.Database = DatabaseCheck{Check: Check{Status: status, Message: message}, Version: cfg.Version, RequiredVersion: appversion.DatabaseSchemaVersion}
	}
	if options.CheckBinanceREST {
		report.BinanceREST = checkBinanceREST(ctx, options)
	} else {
		report.BinanceREST = Check{Status: StatusUnknown, Message: "network check not requested"}
	}
	report.FuturesWS = checkFuturesWS(o, cfg, now)
	marketSources, sourceErr := (marketintelligence.Service{}).SourceStatuses(ctx)
	if sourceErr != nil {
		report.MarketIntelligence = Check{Status: StatusError, Message: "read market intelligence source status failed", LastError: sourceErr.Error()}
		report.AnnouncementWS = Check{Status: StatusUnknown, Message: "announcement websocket status unavailable"}
	} else {
		report.MarketSources = marketSources
		report.MarketIntelligence, report.AnnouncementWS = summarizeMarketSources(report.MarketSources)
	}
	report.MCP, report.MCPServers = checkMCP(o)
	report.LLM = checkLLM(o, now)
	report.Scheduler, report.SchedulerJobs = checkScheduler(o, cfg, now, options)
	report.Agent = checkAgent(o, now)
	report.Trade = checkTrade(o)
	report.markOverall()
	return report, nil
}

func checkBinanceREST(ctx context.Context, options Options) Check {
	base := strings.TrimRight(strings.TrimSpace(options.BinanceBaseURL), "/")
	if base == "" {
		testnet, _ := config.Bool("binance::testnet")
		if testnet {
			base, _ = config.String("binance::testnet_binance_futures_base_url")
			base = strings.TrimRight(strings.TrimSpace(base), "/")
		}
		if base == "" {
			base = "https://fapi.binance.com"
		}
	}
	proxyURL := strings.TrimSpace(options.BinanceProxyURL)
	if proxyURL == "" && !options.IgnoreConfiguredBinanceProxy {
		proxyURL, _ = config.String("binance::proxy_url")
	}
	pool, err := binanceproxy.New(proxyURL)
	if err != nil {
		return Check{Status: StatusError, Message: "invalid Binance proxy", LastError: err.Error()}
	}
	transport := pool.HTTPClient().Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	environment := "mainnet"
	testnet, _ := config.Bool("binance::testnet")
	if testnet {
		environment = "testnet"
	}
	client := binanceapiusage.WrapClient(
		&http.Client{Transport: transport, Timeout: 5 * time.Second},
		binanceapiusage.TransportConfig{Product: "futures", Environment: environment, Source: "system_health"},
	)
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/fapi/v1/time", nil)
	if err != nil {
		return Check{Status: StatusError, Message: "build Binance REST request failed", LastError: err.Error()}
	}
	resp, err := client.Do(req)
	if err != nil {
		return Check{Status: StatusError, Message: "Binance REST unavailable", LastError: err.Error(), LastErrorAt: time.Now().UnixMilli()}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Check{Status: StatusError, Message: fmt.Sprintf("Binance REST HTTP %d", resp.StatusCode), LastErrorAt: time.Now().UnixMilli()}
	}
	return Check{Status: StatusHealthy, Message: fmt.Sprintf("reachable in %d ms", time.Since(start).Milliseconds()), LastSuccessAt: time.Now().UnixMilli()}
}

func checkFuturesWS(o orm.Ormer, cfg models.Config, now time.Time) Check {
	if cfg.WsFuturesEnable != 1 {
		return Check{Status: StatusDisabled, Message: "futures market websocket is disabled"}
	}
	var latest int64
	if err := o.Raw("SELECT COALESCE(MAX(updateTime), 0) FROM symbols").QueryRow(&latest); err != nil {
		return Check{Status: StatusError, Message: "read futures websocket freshness failed", LastError: err.Error()}
	}
	if latest <= 0 {
		return Check{Status: StatusWarning, Message: "no futures market update has been recorded"}
	}
	age := now.UnixMilli() - latest
	if age < 0 {
		age = 0
	}
	status := StatusHealthy
	if age > 15*time.Minute.Milliseconds() {
		status = StatusError
	} else if age > 5*time.Minute.Milliseconds() {
		status = StatusWarning
	}
	return Check{Status: status, Message: fmt.Sprintf("last market update %s ago", humanAge(age)), LastSuccessAt: latest}
}

func summarizeMarketSources(sources []marketintelligence.SourceStatus) (Check, Check) {
	market := Check{Status: StatusDisabled, Message: "no market intelligence source status"}
	announcement := Check{Status: StatusUnknown, Message: "announcement websocket has no status"}
	healthy, errors, other := 0, 0, 0
	for _, source := range sources {
		switch source.Status {
		case "healthy":
			healthy++
		case "error":
			errors++
		default:
			other++
		}
		if source.Source == "binance_announcement" {
			status := StatusUnknown
			switch source.Status {
			case "healthy":
				status = StatusHealthy
			case "error":
				status = StatusError
			case "disabled":
				status = StatusDisabled
			}
			announcement = Check{Status: status, Message: source.Status, LastSuccessAt: source.LastSuccessAt, LastErrorAt: source.LastErrorAt, LastError: source.LastError}
		}
	}
	if len(sources) > 0 {
		switch {
		case errors > 0:
			market = Check{Status: StatusWarning, Message: fmt.Sprintf("%d healthy, %d error, %d other market sources", healthy, errors, other), Count: int64(errors)}
		case other > 0:
			market = Check{Status: StatusWarning, Message: fmt.Sprintf("%d healthy, %d non-healthy market sources", healthy, other), Count: int64(other)}
		default:
			market = Check{Status: StatusHealthy, Message: fmt.Sprintf("%d sources healthy", healthy)}
		}
	}
	apiKey, _ := config.String("binance::api_key")
	if strings.TrimSpace(apiKey) == "" {
		announcement = Check{Status: StatusDisabled, Message: "Binance announcement credentials are not configured"}
	}
	return market, announcement
}

func checkMCP(o orm.Ormer) (Check, []MCPServerStatus) {
	var rows []models.AgentMCPServer
	if _, err := o.QueryTable(new(models.AgentMCPServer)).Filter("enabled", 1).OrderBy("name").All(&rows); err != nil {
		return Check{Status: StatusError, Message: "read MCP status failed", LastError: err.Error()}, []MCPServerStatus{}
	}
	items := make([]MCPServerStatus, 0, len(rows))
	errors := 0
	for _, row := range rows {
		items = append(items, MCPServerStatus{Name: row.Name, Status: row.Status, LastSuccessAt: row.LastSuccessAt, LastErrorAt: row.LastErrorAt, LastError: row.LastError})
		if row.Status != "healthy" {
			errors++
		}
	}
	if len(rows) == 0 {
		return Check{Status: StatusDisabled, Message: "no enabled MCP servers"}, items
	}
	if errors > 0 {
		return Check{Status: StatusWarning, Message: fmt.Sprintf("%d/%d MCP servers are not healthy", errors, len(rows)), Count: int64(errors)}, items
	}
	return Check{Status: StatusHealthy, Message: fmt.Sprintf("%d MCP servers healthy", len(rows))}, items
}

func checkLLM(o orm.Ormer, now time.Time) Check {
	var candidates int64
	if err := o.Raw("SELECT COUNT(*) FROM llm_configs WHERE deleted=0 AND (enabled=1 OR router_candidate=1)").QueryRow(&candidates); err != nil {
		return Check{Status: StatusError, Message: "read LLM configuration failed", LastError: err.Error()}
	}
	if candidates == 0 {
		return Check{Status: StatusError, Message: "no enabled LLM routing candidate"}
	}
	cutoff := now.Add(-24 * time.Hour).UnixMilli()
	var failures int64
	if err := o.Raw("SELECT COUNT(*) FROM agent_observations WHERE type='llm_call' AND created_at>=? AND status!='success'", cutoff).QueryRow(&failures); err != nil {
		return Check{Status: StatusError, Message: "read LLM observation failures failed", LastError: err.Error()}
	}
	var latest struct {
		Status    string `orm:"column(status)"`
		Error     string `orm:"column(error)"`
		CreatedAt int64  `orm:"column(created_at)"`
	}
	err := o.Raw("SELECT status, error, created_at FROM agent_observations WHERE type='llm_call' ORDER BY created_at DESC LIMIT 1").QueryRow(&latest)
	if err != nil && err != orm.ErrNoRows {
		return Check{Status: StatusError, Message: "read latest LLM observation failed", LastError: err.Error()}
	}
	check := Check{Status: StatusHealthy, Message: fmt.Sprintf("%d routing candidates", candidates), Count: candidates}
	if failures > 0 {
		check.Message = fmt.Sprintf("%d routing candidates; %d LLM failures in 24h", candidates, failures)
	}
	if err == nil && latest.Status != "" && latest.Status != "success" {
		check.Status, check.LastError, check.LastErrorAt = StatusWarning, latest.Error, latest.CreatedAt
	}
	return check
}

func checkScheduler(o orm.Ormer, cfg models.Config, now time.Time, options Options) (Check, []SchedulerJobStatus) {
	if options.SchedulerRuntimeAvailable {
		items := make([]SchedulerJobStatus, 0, len(options.SchedulerJobs))
		problems, enabled := 0, 0
		for _, job := range options.SchedulerJobs {
			item := SchedulerJobStatus{Name: job.Name, Skill: job.Skill, Enabled: job.Enabled, Running: job.Running, IntervalSeconds: job.IntervalSeconds, LastStatus: job.LastStatus, LastError: job.LastError, LastRunAt: job.LastRunAt, NextRunAt: job.NextRunAt, RunCount: job.RunCount, SkipCount: job.SkipCount}
			items = append(items, item)
			if job.Enabled {
				enabled++
				if job.LastStatus == "failed" || job.LastError != "" {
					problems++
				}
			}
		}
		if enabled == 0 {
			return Check{Status: StatusDisabled, Message: "no enabled scheduler jobs"}, items
		}
		if problems > 0 {
			return Check{Status: StatusWarning, Message: fmt.Sprintf("%d scheduler jobs report errors", problems), Count: int64(problems)}, items
		}
		return Check{Status: StatusHealthy, Message: fmt.Sprintf("%d scheduler jobs enabled", enabled)}, items
	}
	definitions := schedulerDefinitions(cfg)
	items := make([]SchedulerJobStatus, 0, len(definitions))
	problems, enabled := 0, 0
	for _, item := range definitions {
		if item.Enabled {
			enabled++
			var latest struct {
				Status      string `orm:"column(status)"`
				Error       string `orm:"column(error)"`
				CompletedAt int64  `orm:"column(completed_at)"`
				CreatedAt   int64  `orm:"column(created_at)"`
			}
			err := o.Raw("SELECT status, error, completed_at, created_at FROM agent_tasks WHERE skill=? ORDER BY created_at DESC LIMIT 1", item.Skill).QueryRow(&latest)
			if err != nil && err != orm.ErrNoRows {
				return Check{Status: StatusError, Message: "read scheduler task freshness failed", LastError: err.Error()}, items
			}
			if err == nil {
				item.LastStatus, item.LastError = latest.Status, latest.Error
				item.LastRunAt = latest.CompletedAt
				if item.LastRunAt <= 0 {
					item.LastRunAt = latest.CreatedAt
				}
			}
			staleAfter := time.Duration(item.IntervalSeconds) * time.Second * 3
			if err == orm.ErrNoRows || item.LastRunAt <= 0 || now.Sub(time.UnixMilli(item.LastRunAt)) > staleAfter || item.LastStatus == "failed" {
				problems++
			}
		}
		items = append(items, item)
	}
	if enabled == 0 {
		return Check{Status: StatusDisabled, Message: "no enabled scheduler jobs"}, items
	}
	if problems > 0 {
		return Check{Status: StatusWarning, Message: fmt.Sprintf("%d scheduler jobs need attention (database freshness check)", problems), Count: int64(problems)}, items
	}
	return Check{Status: StatusHealthy, Message: fmt.Sprintf("%d scheduler jobs have recent task results", enabled)}, items
}

func schedulerDefinitions(cfg models.Config) []SchedulerJobStatus {
	regimeInterval := cfg.AgentMarketRegimeIntervalMin
	if regimeInterval <= 0 {
		regimeInterval = 60
	}
	scanInterval := cfg.AgentOpportunityScanIntervalMin
	if scanInterval <= 0 {
		scanInterval = 60
	}
	briefInterval := cfg.AgentDailyMarketBriefIntervalMin
	if briefInterval <= 0 {
		briefInterval = 1440
	}
	return []SchedulerJobStatus{
		{Name: "market_regime", Skill: "market_regime", Enabled: cfg.AgentMarketRegimeScheduleEnable == 1 && cfg.MarketConditionIsAuto == 1, IntervalSeconds: int64(regimeInterval * 60)},
		{Name: "opportunity_market_scan", Skill: "market_scan", Enabled: cfg.AgentOpportunityWatchEnable == 1, IntervalSeconds: int64(scanInterval * 60)},
		{Name: "daily_market_brief", Skill: "daily_market_brief", Enabled: cfg.AgentDailyMarketBriefScheduleEnable == 1, IntervalSeconds: int64(briefInterval * 60)},
	}
}

func checkAgent(o orm.Ormer, now time.Time) AgentCheck {
	cutoff := now.Add(-24 * time.Hour).UnixMilli()
	result := AgentCheck{}
	if err := o.Raw("SELECT COUNT(*) FROM agent_tasks WHERE created_at>=?", cutoff).QueryRow(&result.Tasks24h); err != nil {
		result.Status, result.Message, result.LastError = StatusError, "read Agent task count failed", err.Error()
		return result
	}
	if err := o.Raw("SELECT COUNT(*) FROM agent_tasks WHERE created_at>=? AND status='failed'", cutoff).QueryRow(&result.Failed24h); err != nil {
		result.Status, result.Message, result.LastError = StatusError, "read Agent failed task count failed", err.Error()
		return result
	}
	if err := o.Raw("SELECT COUNT(*) FROM agent_tasks WHERE created_at>=? AND status='failed' AND (round>=max_rounds OR LOWER(error) LIKE '%maximum%round%')", cutoff).QueryRow(&result.MaxRoundsFailed); err != nil {
		result.Status, result.Message, result.LastError = StatusError, "read Agent max-round failure count failed", err.Error()
		return result
	}
	result.Status = StatusHealthy
	result.Message = fmt.Sprintf("%d tasks in 24h", result.Tasks24h)
	if result.Failed24h > 0 {
		result.Status = StatusWarning
		result.Message = fmt.Sprintf("%d/%d tasks failed in 24h", result.Failed24h, result.Tasks24h)
		result.Count = result.Failed24h
	}
	return result
}

func checkTrade(o orm.Ormer) TradeCheck {
	result := TradeCheck{}
	queries := []struct {
		label string
		query string
		dest  *int64
	}{
		{"execution_uncertain", "SELECT COUNT(*) FROM agent_trade_proposals WHERE status='execution_uncertain'", &result.ExecutionUncertain},
		{"protection_failed", "SELECT COUNT(*) FROM agent_trade_proposals WHERE status='protection_failed'", &result.ProtectionFailed},
	}
	for _, item := range queries {
		if err := o.Raw(item.query).QueryRow(item.dest); err != nil {
			result.Status, result.Message, result.LastError = StatusError, "read trade safety "+item.label+" count failed", err.Error()
			return result
		}
	}
	var positions, orders int64
	if err := o.Raw("SELECT COUNT(*) FROM futures_managed_positions WHERE status='reconcile_required'").QueryRow(&positions); err != nil {
		result.Status, result.Message, result.LastError = StatusError, "read managed position reconcile count failed", err.Error()
		return result
	}
	if err := o.Raw("SELECT COUNT(*) FROM futures_managed_orders WHERE status='reconcile_required'").QueryRow(&orders); err != nil {
		result.Status, result.Message, result.LastError = StatusError, "read managed order reconcile count failed", err.Error()
		return result
	}
	result.ReconcileRequired = positions + orders
	result.Status, result.Message = StatusHealthy, "no trade safety issue"
	if result.ProtectionFailed > 0 {
		result.Status, result.Message = StatusWarning, fmt.Sprintf("%d protection failures need review", result.ProtectionFailed)
	}
	if result.ExecutionUncertain > 0 || result.ReconcileRequired > 0 {
		result.Status = StatusError
		result.Message = fmt.Sprintf("execution_uncertain=%d reconcile_required=%d protection_failed=%d", result.ExecutionUncertain, result.ReconcileRequired, result.ProtectionFailed)
	}
	result.Count = result.ExecutionUncertain + result.ProtectionFailed + result.ReconcileRequired
	return result
}

func (report *Report) markOverall() {
	report.Overall = StatusHealthy
	statuses := []string{report.Database.Status, report.BinanceREST.Status, report.FuturesWS.Status, report.AnnouncementWS.Status, report.MarketIntelligence.Status, report.MCP.Status, report.LLM.Status, report.Scheduler.Status, report.Agent.Status, report.Trade.Status}
	for _, status := range statuses {
		if status == StatusError {
			report.Overall = StatusError
			return
		}
		if (status == StatusWarning || status == StatusUnknown) && report.Overall == StatusHealthy {
			report.Overall = StatusWarning
		}
	}
}

func humanAge(ms int64) string {
	d := time.Duration(ms) * time.Millisecond
	if d < time.Minute {
		return fmt.Sprintf("%ds", int64(d/time.Second))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int64(d/time.Minute))
	}
	return fmt.Sprintf("%dh%dm", int64(d/time.Hour), int64((d%time.Hour)/time.Minute))
}
