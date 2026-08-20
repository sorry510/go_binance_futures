package controllers

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/llm"
	"go_binance_futures/technology"
	"go_binance_futures/types"
	"go_binance_futures/utils"
	"strings"
	"sync"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/expr-lang/expr"
)

const (
	strategyTemplateAITaskRetention  = 30 * time.Minute
	maxStrategyTemplateAIPromptSize  = 12 * 1024
	maxStrategyTemplateAIContextSize = 256 * 1024
)

const strategyTemplateAISystemPrompt = `You generate portable custom futures strategy template JSON for go_binance_futures.

Return exactly one compact JSON object. Do not use Markdown fences, comments, explanations, trading advice, position sizing, leverage, or order instructions.

The root object must contain exactly:
{"name":string,"technology":object,"strategy":array}

technology must contain all of these array fields, even when empty:
ma, ema, macd, adx, mfi, obv, cci, roc, kdj, rsi, kc, boll, donchian, atr, supertrend.

Every enabled indicator has name, kline_interval, enable. Names must be unique valid expr identifiers and must not start with kline_. Supported intervals: 1m,3m,5m,15m,30m,1h,2h,4h,6h,8h,12h,1d,3d,1w,1M.
Fields by family:
- ma/ema/mfi/cci/roc/rsi/atr/donchian/adx: period
- macd: fast_period, slow_period, signal_period; fast_period < slow_period
- obv: no period
- kdj: period, k_period, d_period
- kc/supertrend: period, multiplier
- boll: period, std_dev_multiplier
Periods are positive and at most 150; rsi/mfi/roc must be below 150; adx at most 75; multipliers must be non-negative and supertrend multiplier must be positive.

Each strategy item must contain exactly name, type, code, fullScreen, enable. fullScreen and enable are booleans. type is one of long, short, close_long, close_short. Names must be unique. Each expr-lang code must compile and its final expression must be Boolean.

Runtime contract:
- Array index 0 is the current/forming K-line; index 1 is the latest closed K-line.
- Every enabled interval exposes kline_INTERVAL with High, Low, Open, Close, Amount, Qps arrays.
- ma/ema/mfi/cci/roc/rsi/atr expose .Data.
- macd exposes .DIF, .DEA, .Histogram.
- adx exposes .ADX, .PlusDI, .MinusDI.
- obv exposes .Data; kdj exposes .K, .D, .J.
- kc/boll/donchian expose .High, .Mid, .Low.
- supertrend exposes .Data and .Trend where 1 is bullish and -1 is bearish.
- Available globals include MarketCondition (string), BasicTrend, NowTime, NowPrice, NowSymbolPercentChange, NowSymbolClose/Open/Low/High, BTCUSDT/ETHUSDT/SOLUSDT/BNBUSDT market objects, IsAsc, IsDesc, KdjSimple. long/short rules additionally receive Positions. close_long/close_short rules receive ROI and Position, but not Positions.
- A live Donchian breakout must compare kline close[0] with channel High[1] or Low[1], because channel[0] includes the forming K-line.
- Close expressions are evaluated only after the symbol's outer profit/loss threshold is crossed. Do not claim an inner threshold can bypass that gate.

Prefer two distinct intervals and three or four enabled indicators. Unless the user explicitly narrows the request, generate one enabled rule for each of long, short, close_long, close_short. Keep rules symmetric, readable, efficient, and conservative. Do not claim profitability.`

type strategyTemplateAIGenerationRequest struct {
	Prompt          string `json:"prompt"`
	PreviousJSON    string `json:"previousJson"`
	ValidationError string `json:"validationError"`
}

type strategyTemplateSyntheticMarket struct {
	PercentChange float64
	Close         float64
	Open          float64
	Low           float64
	High          float64
}

type strategyTemplateAIProgressEvent struct {
	Progress int       `json:"progress"`
	Stage    string    `json:"stage"`
	Message  string    `json:"message"`
	Time     time.Time `json:"time"`
}

type strategyTemplateAIGenerationTask struct {
	TaskID          string                            `json:"taskId"`
	Status          string                            `json:"status"`
	Progress        int                               `json:"progress"`
	Stage           string                            `json:"stage"`
	Events          []strategyTemplateAIProgressEvent `json:"events"`
	JSON            string                            `json:"json,omitempty"`
	ValidationError string                            `json:"validationError,omitempty"`
	Error           string                            `json:"error,omitempty"`
	CreatedAt       time.Time                         `json:"createdAt"`
	UpdatedAt       time.Time                         `json:"updatedAt"`
	CompletedAt     *time.Time                        `json:"completedAt,omitempty"`
}

var strategyTemplateAITaskStore = struct {
	sync.RWMutex
	tasks map[string]*strategyTemplateAIGenerationTask
}{
	tasks: make(map[string]*strategyTemplateAIGenerationTask),
}

func (ctrl *StrategyTemplateController) StartAIGeneration() {
	var request strategyTemplateAIGenerationRequest
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &request); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "请求格式错误: "+err.Error()))
		return
	}
	request.Prompt = strings.TrimSpace(request.Prompt)
	if request.Prompt == "" {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "提示词不能为空"))
		return
	}
	if len(request.Prompt) > maxStrategyTemplateAIPromptSize {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "提示词不能超过 12 KB"))
		return
	}
	if len(request.PreviousJSON)+len(request.ValidationError) > maxStrategyTemplateAIContextSize {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "上一次生成内容和错误信息不能超过 256 KB"))
		return
	}

	task := startStrategyTemplateAITask(request)
	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"data": task,
		"msg":  "accepted",
	})
}

func (ctrl *StrategyTemplateController) GetAIGenerationTask() {
	taskID := ctrl.Ctx.Input.Param(":taskId")
	task, exists := getStrategyTemplateAITask(taskID)
	if !exists {
		ctrl.Ctx.Resp(utils.ResJson(404, nil, "task not found"))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"data": task,
		"msg":  "success",
	})
}

func startStrategyTemplateAITask(request strategyTemplateAIGenerationRequest) strategyTemplateAIGenerationTask {
	now := time.Now().UTC()
	taskID := newStrategyTemplateAITaskID()
	task := &strategyTemplateAIGenerationTask{
		TaskID:    taskID,
		Status:    "queued",
		Progress:  0,
		Stage:     "queued",
		CreatedAt: now,
		UpdatedAt: now,
	}
	strategyTemplateAITaskStore.Lock()
	cleanupStrategyTemplateAITasksLocked(now)
	strategyTemplateAITaskStore.tasks[taskID] = task
	appendStrategyTemplateAIEventLocked(task, 0, "queued", "生成任务已创建")
	result := cloneStrategyTemplateAITask(task)
	strategyTemplateAITaskStore.Unlock()

	go runStrategyTemplateAITask(taskID, request)
	return result
}

func runStrategyTemplateAITask(taskID string, request strategyTemplateAIGenerationRequest) {
	updateStrategyTemplateAITask(taskID, 5, "initializing_llm", "正在初始化 LLM 客户端")
	client, err := llm.NewFromConfig()
	if err != nil {
		failStrategyTemplateAITask(taskID, "LLM 配置不可用: "+err.Error())
		return
	}

	updateStrategyTemplateAITask(taskID, 15, "building_prompt", "正在整理策略需求和框架约束")
	userPrompt := buildStrategyTemplateAIUserPrompt(request)
	updateStrategyTemplateAITask(taskID, 25, "calling_llm", "已发送请求，等待 AI 生成 JSON")

	response, err := generateStrategyTemplateWithProgress(taskID, client, userPrompt)
	if err != nil {
		logs.Error("strategy template AI task %s: %s", taskID, err.Error())
		failStrategyTemplateAITask(taskID, "AI 生成失败: "+truncateStrategyTemplateAIError(err.Error()))
		return
	}

	updateStrategyTemplateAITask(taskID, 75, "extracting_json", "已收到 AI 响应，正在提取 JSON")
	generatedJSON := extractStrategyTemplateJSON(response.Content)
	if strings.TrimSpace(generatedJSON) == "" {
		failStrategyTemplateAITask(taskID, "AI 未返回可用内容")
		return
	}

	updateStrategyTemplateAITask(taskID, 86, "validating_json", "正在校验 JSON 结构、指标参数和 expr 规则")
	validationError := ""
	if err := validateGeneratedStrategyTemplateJSON([]byte(generatedJSON)); err != nil {
		validationError = err.Error()
	}
	if isLLMResponseTruncated(response.FinishReason) {
		validationError = joinStrategyTemplateAIValidationErrors(validationError, "AI 响应可能因 max_tokens 限制被截断，请提高配置后重新生成")
	}
	generatedJSON = formatStrategyTemplateJSON(generatedJSON)
	finishStrategyTemplateAITask(taskID, generatedJSON, validationError)
}

func generateStrategyTemplateWithProgress(taskID string, client llm.Client, userPrompt string) (*llm.Response, error) {
	type result struct {
		response *llm.Response
		err      error
	}
	resultChannel := make(chan result, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		response, err := client.Generate(ctx, llm.Request{
			System: strategyTemplateAISystemPrompt,
			Messages: []llm.Message{{
				Role:    llm.RoleUser,
				Content: userPrompt,
			}},
		})
		resultChannel <- result{response: response, err: err}
	}()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	waitedSeconds := 0
	for {
		select {
		case generated := <-resultChannel:
			return generated.response, generated.err
		case <-ticker.C:
			waitedSeconds += 5
			progress := 25 + waitedSeconds/5*3
			if progress > 68 {
				progress = 68
			}
			updateStrategyTemplateAITask(taskID, progress, "waiting_llm", fmt.Sprintf("AI 正在生成，已等待 %d 秒", waitedSeconds))
		}
	}
}

func buildStrategyTemplateAIUserPrompt(request strategyTemplateAIGenerationRequest) string {
	var builder strings.Builder
	builder.WriteString("User requirements:\n")
	builder.WriteString(request.Prompt)
	if strings.TrimSpace(request.PreviousJSON) != "" || strings.TrimSpace(request.ValidationError) != "" {
		builder.WriteString("\n\nRepair context from the previous attempt. Generate a complete replacement JSON, not a patch.\n")
		if strings.TrimSpace(request.ValidationError) != "" {
			builder.WriteString("Validation/import error:\n")
			builder.WriteString(request.ValidationError)
			builder.WriteString("\n")
		}
		if strings.TrimSpace(request.PreviousJSON) != "" {
			builder.WriteString("Previous JSON:\n")
			builder.WriteString(request.PreviousJSON)
		}
	}
	return builder.String()
}

func extractStrategyTemplateJSON(content string) string {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		lines := strings.Split(content, "\n")
		if len(lines) >= 3 {
			lines = lines[1:]
			if strings.TrimSpace(lines[len(lines)-1]) == "```" {
				lines = lines[:len(lines)-1]
			}
			content = strings.TrimSpace(strings.Join(lines, "\n"))
		}
	}
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start >= 0 && end >= start {
		return strings.TrimSpace(content[start : end+1])
	}
	return content
}

func formatStrategyTemplateJSON(value string) string {
	var buffer bytes.Buffer
	if err := json.Indent(&buffer, []byte(value), "", "  "); err != nil {
		return value
	}
	return buffer.String()
}

func validateGeneratedStrategyTemplateJSON(data []byte) error {
	if _, err := parseStrategyTemplateImport(data); err != nil {
		return err
	}
	if err := validateGeneratedTechnologyKeys(data); err != nil {
		return err
	}
	return nil
}

func validateStrategyTemplateRuleExpressions(config technology.TechnologyConfig, rules []strategyTemplateImportRule) error {
	if len(rules) == 0 {
		return fmt.Errorf("strategy 至少需要一条策略规则")
	}
	for index, rule := range rules {
		env := buildSyntheticStrategyTemplateEnv(config, rule.Type)
		program, err := expr.Compile(rule.Code, expr.Env(env), expr.AsBool())
		if err != nil {
			return fmt.Errorf("strategy 第 %d 项 %q 的 expr 编译失败: %w", index+1, rule.Name, err)
		}
		output, err := expr.Run(program, env)
		if err != nil {
			return fmt.Errorf("strategy 第 %d 项 %q 的 expr 运行失败: %w", index+1, rule.Name, err)
		}
		if _, ok := output.(bool); !ok {
			return fmt.Errorf("strategy 第 %d 项 %q 的 expr 最终结果必须是布尔值", index+1, rule.Name)
		}
	}
	return nil
}

func validateGeneratedTechnologyKeys(data []byte) error {
	var payload struct {
		Technology map[string]json.RawMessage `json:"technology"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	required := []string{"ma", "ema", "macd", "adx", "mfi", "obv", "cci", "roc", "kdj", "rsi", "kc", "boll", "donchian", "atr", "supertrend"}
	for _, key := range required {
		value, exists := payload.Technology[key]
		if !exists {
			return fmt.Errorf("technology 缺少数组字段 %s", key)
		}
		var items []json.RawMessage
		if err := json.Unmarshal(value, &items); err != nil {
			return fmt.Errorf("technology.%s 必须是数组", key)
		}
	}
	return nil
}

func buildSyntheticStrategyTemplateEnv(config technology.TechnologyConfig, ruleType string) map[string]interface{} {
	series := syntheticStrategyTemplateSeries()
	market := strategyTemplateSyntheticMarket{PercentChange: 1.0, Close: 101.0, Open: 100.0, Low: 99.0, High: 102.0}
	env := map[string]interface{}{
		"SystemStartTime":        int64(1_700_000_000_000),
		"MarketCondition":        "3",
		"NowTime":                int64(1_700_000_060_000),
		"NowPrice":               101.0,
		"NowSymbolPercentChange": 1.0,
		"NowSymbolClose":         101.0,
		"NowSymbolOpen":          100.0,
		"NowSymbolLow":           99.0,
		"NowSymbolHigh":          102.0,
		"BasicTrend":             0.5,
		"KdjSimple":              line.KdjSimple,
		"IsAsc":                  utils.IsAsc,
		"IsDesc":                 utils.IsDesc,
		"BTCUSDT":                market,
		"ETHUSDT":                market,
		"SOLUSDT":                market,
		"BNBUSDT":                market,
	}
	if ruleType == "long" || ruleType == "short" {
		env["Positions"] = []types.FuturesPosition{}
	} else {
		env["ROI"] = 8.0
		env["Position"] = types.FuturesPositionCode{
			Symbol: "BTCUSDT", Side: "LONG", Amount: 0.01, Leverage: 3,
			EntryPrice: 100.0, MarkPrice: 101.0, UnrealizedProfit: 1.0,
			CreateTime: 1_700_000_000_000, SourceType: "local",
		}
	}
	intervals := make(map[string]struct{})
	addItems := func(kind string, items []technology.IndicatorConfig) {
		for _, item := range items {
			if !item.Enable {
				continue
			}
			intervals[item.KlineInterval] = struct{}{}
			switch kind {
			case "macd":
				env[item.Name] = line.MACDConfigData{KlineInterval: item.KlineInterval, FastPeriod: item.FastPeriod, SlowPeriod: item.SlowPeriod, SignalPeriod: item.SignalPeriod, DIF: series, DEA: series, Histogram: series}
			case "adx":
				env[item.Name] = line.ADXConfigData{KlineInterval: item.KlineInterval, Period: item.Period, ADX: series, PlusDI: series, MinusDI: series}
			case "kdj":
				env[item.Name] = line.KDJConfigData{KlineInterval: item.KlineInterval, Period: item.Period, KPeriod: item.KPeriod, DPeriod: item.DPeriod, K: series, D: series, J: series}
			case "supertrend":
				env[item.Name] = line.SupertrendConfigData{KlineInterval: item.KlineInterval, Period: item.Period, Multiplier: item.Multiplier, Data: series, Trend: series}
			case "obv":
				env[item.Name] = line.OBVConfigData{KlineInterval: item.KlineInterval, Data: series}
			case "kc", "boll", "donchian":
				env[item.Name] = line.ConfigData{KlineInterval: item.KlineInterval, Period: item.Period, Multiplier: item.Multiplier, StdDevMultiplier: item.StdDevMultiplier, High: series, Mid: series, Low: series}
			default:
				env[item.Name] = line.ConfigData{KlineInterval: item.KlineInterval, Period: item.Period, Data: series}
			}
		}
	}
	addItems("ma", config.MA)
	addItems("ema", config.EMA)
	addItems("macd", config.MACD)
	addItems("adx", config.ADX)
	addItems("mfi", config.MFI)
	addItems("obv", config.OBV)
	addItems("cci", config.CCI)
	addItems("roc", config.ROC)
	addItems("kdj", config.KDJ)
	addItems("rsi", config.RSI)
	addItems("kc", config.KC)
	addItems("boll", config.BOLL)
	addItems("donchian", config.Donchian)
	addItems("atr", config.ATR)
	addItems("supertrend", config.Supertrend)
	for interval := range intervals {
		env["kline_"+interval] = line.KLinePrice{High: series, Low: series, Close: series, Open: series, Amount: series, Qps: series}
	}
	return env
}

func syntheticStrategyTemplateSeries() []float64 {
	values := make([]float64, 150)
	for index := range values {
		values[index] = 100 + float64(150-index)/10
	}
	return values
}

func updateStrategyTemplateAITask(taskID string, progress int, stage, message string) {
	strategyTemplateAITaskStore.Lock()
	defer strategyTemplateAITaskStore.Unlock()
	task := strategyTemplateAITaskStore.tasks[taskID]
	if task == nil || (task.Status != "queued" && task.Status != "running") {
		return
	}
	task.Status = "running"
	appendStrategyTemplateAIEventLocked(task, progress, stage, message)
}

func appendStrategyTemplateAIEventLocked(task *strategyTemplateAIGenerationTask, progress int, stage, message string) {
	now := time.Now().UTC()
	task.Progress = progress
	task.Stage = stage
	task.UpdatedAt = now
	task.Events = append(task.Events, strategyTemplateAIProgressEvent{Progress: progress, Stage: stage, Message: message, Time: now})
}

func finishStrategyTemplateAITask(taskID, generatedJSON, validationError string) {
	strategyTemplateAITaskStore.Lock()
	defer strategyTemplateAITaskStore.Unlock()
	task := strategyTemplateAITaskStore.tasks[taskID]
	if task == nil {
		return
	}
	task.JSON = generatedJSON
	task.ValidationError = validationError
	task.Status = "succeeded"
	message := "JSON 已生成并通过框架校验"
	if validationError != "" {
		message = "JSON 已生成，但框架校验发现问题"
	}
	appendStrategyTemplateAIEventLocked(task, 100, "completed", message)
	completedAt := task.UpdatedAt
	task.CompletedAt = &completedAt
}

func failStrategyTemplateAITask(taskID, errorMessage string) {
	strategyTemplateAITaskStore.Lock()
	defer strategyTemplateAITaskStore.Unlock()
	task := strategyTemplateAITaskStore.tasks[taskID]
	if task == nil {
		return
	}
	task.Status = "failed"
	task.Error = errorMessage
	appendStrategyTemplateAIEventLocked(task, 100, "failed", errorMessage)
	completedAt := task.UpdatedAt
	task.CompletedAt = &completedAt
}

func getStrategyTemplateAITask(taskID string) (strategyTemplateAIGenerationTask, bool) {
	strategyTemplateAITaskStore.RLock()
	defer strategyTemplateAITaskStore.RUnlock()
	task, exists := strategyTemplateAITaskStore.tasks[taskID]
	if !exists {
		return strategyTemplateAIGenerationTask{}, false
	}
	return cloneStrategyTemplateAITask(task), true
}

func cloneStrategyTemplateAITask(task *strategyTemplateAIGenerationTask) strategyTemplateAIGenerationTask {
	cloned := *task
	cloned.Events = append([]strategyTemplateAIProgressEvent(nil), task.Events...)
	return cloned
}

func cleanupStrategyTemplateAITasksLocked(now time.Time) {
	for taskID, task := range strategyTemplateAITaskStore.tasks {
		if task.Status == "queued" || task.Status == "running" || now.Sub(task.UpdatedAt) <= strategyTemplateAITaskRetention {
			continue
		}
		delete(strategyTemplateAITaskStore.tasks, taskID)
	}
}

func newStrategyTemplateAITaskID() string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err == nil {
		return hex.EncodeToString(value)
	}
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func isLLMResponseTruncated(finishReason string) bool {
	finishReason = strings.ToLower(strings.TrimSpace(finishReason))
	return finishReason == "length" || finishReason == "max_tokens"
}

func joinStrategyTemplateAIValidationErrors(current, next string) string {
	if current == "" {
		return next
	}
	return current + "；" + next
}

func truncateStrategyTemplateAIError(message string) string {
	const maxLength = 1000
	message = strings.TrimSpace(message)
	if len(message) <= maxLength {
		return message
	}
	return message[:maxLength] + "..."
}
