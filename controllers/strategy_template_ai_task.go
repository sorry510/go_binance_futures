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
	"strconv"
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
	maxStrategyTemplateAIRounds      = 15 // 最大15循环
)

const strategyTemplateAISystemPrompt = `Generate valid custom futures strategy-template JSON for go_binance_futures. You have at most 10 agent rounds. Each response must be one compact JSON object without Markdown or extra text.

Agent protocol:
- Tool: {"action":"tool","summary":"reason/current findings","tool":"NAME","arguments":{...}}
- Final: {"action":"final","summary":"design/evidence summary","json":{"name":"...","technology":{},"strategy":[]}}
- Tools: get_features arguments are sort,symbol_type,symbol,enable,margin_type,pin,page,limit (default/max limit 20); get_test_strategy_results arguments are symbol,position_side,start_time,end_time,type,page,limit (default/max limit 100). TOOL_RESULT arrives in the next user message; include relevant findings in the next summary. Avoid duplicate calls without a new reason.
- A request for test results MUST call get_test_strategy_results first, preserving symbol/side/time/type filters and defaulting to page=1,limit=100. A request for a coin/contract/symbol's data MUST call get_features first with that symbol and requested filters. If both are requested, call both in separate rounds.

Final json contract:
- Root is exactly {"name":string,"technology":object,"strategy":array}.
- technology always contains arrays: ma,ema,macd,adx,mfi,obv,cci,roc,kdj,rsi,kc,boll,donchian,atr,supertrend.
- Every enabled indicator has name,kline_interval,enable. name is a unique expr identifier, not reserved and not prefixed kline_. Intervals: 1m,3m,5m,15m,30m,1h,2h,4h,6h,8h,12h,1d,3d,1w,1M.
- Config fields: ma/ema/adx/mfi/cci/roc/rsi/donchian/atr use period; macd uses fast_period,slow_period,signal_period with fast<slow and slow+signal-1<=150; obv has no period; kdj uses period,k_period,d_period; kc/supertrend use period,multiplier; boll uses period,std_dev_multiplier. Periods are 1..150, except rsi/mfi/roc <=149 and adx <=75; KDJ smoothing periods are 1..150; KC/BOLL multipliers are >=0 and Supertrend multiplier is >0.
- Every strategy item is exactly {name,type,code,fullScreen,enable}; names are unique, booleans are booleans, type is long|short|close_long|close_short, and expr code must compile to Boolean.

Indicator terms and generated values (each configured indicator also has KlineInterval):
- ma (Simple Moving Average): arithmetic mean of Close; smooth lagging trend. Exposes Period,Data.
- ema (Exponential Moving Average): recent Close has greater weight than MA, so it reacts faster. Exposes Period,Data.
- macd (Moving Average Convergence Divergence): DIF=fast EMA-slow EMA, DEA=signal EMA of DIF, Histogram=DIF-DEA; crosses/zero line show momentum direction and Histogram its expansion/contraction. Exposes FastPeriod,SlowPeriod,SignalPeriod,DIF,DEA,Histogram.
- adx (Average Directional Index/DMI): ADX measures trend strength, not direction; PlusDI>MinusDI is positive direction and the reverse is negative; 20-25 is only a common reference. Exposes Period,ADX,PlusDI,MinusDI.
- mfi (Money Flow Index): 0..100 price-volume oscillator using typical price and quote turnover Amount; 20/80 are reference zones, not reversal guarantees. Exposes Period,Data.
- obv (On-Balance Volume): cumulative Amount added on rising closes and subtracted on falling closes; it restarts at zero within the fetched 150-candle window, so use direction/slope/differences, not absolute level. Exposes Data.
- cci (Commodity Channel Index): normalized deviation of typical price from its average; +100/-100 are common references and zero is returned when mean deviation is zero. Exposes Period,Data.
- roc (Rate of Change): percentage Close change versus period candles earlier; sign and zero crosses show momentum direction. Exposes Period,Data.
- kdj (Stochastic KDJ): smoothed RSV where J=3*K-2*D; K/D crosses and J extremes describe range position/momentum, not guaranteed reversals. Exposes Period,KPeriod,DPeriod,K,D,J.
- rsi (Relative Strength Index): Wilder-smoothed 0..100 momentum oscillator; 30/70 are references and flat input returns 50. Exposes Period,Data.
- kc (Keltner Channel): EMA midpoint and EMA +/- multiplier*ATR volatility channel. Exposes Period,Multiplier,High,Mid,Low.
- boll (Bollinger Bands): SMA midpoint and population-standard-deviation bands; width shows volatility and a band touch alone is not a reversal. Exposes Period,StdDevMultiplier,High,Mid,Low.
- donchian (Donchian Channel): period highest High, lowest Low, and midpoint; defines recent range/breakout boundaries. Exposes Period,High,Mid,Low.
- atr (Average True Range): Wilder-smoothed true range measuring absolute volatility, not direction. Exposes Period,Data.
- supertrend: ATR trailing trend line; Data is the active line and Trend is 1 bullish/-1 bearish. Exposes Period,Multiplier,Data,Trend.

Runtime data and variables:
- All arrays are newest-to-oldest, max 150; [0] is the forming K-line and [1] the latest closed K-line. Each enabled interval creates kline_INTERVAL with High,Low,Open,Close price arrays, Amount quote turnover (volume*average price), and Qps quote turnover/second.
- SystemStartTime (int64): system-start Unix milliseconds. NowTime (int64): current Unix milliseconds.
- MarketCondition (string): regime code 1 strong bull, 2 bull, 3 sideways, 4 bear, 5 strong bear, 6 bullish divergence, 7 bearish divergence, 8 broad rise, 9 broad decline, 10 high-volatility sideways, 11 low-volatility consolidation.
- BasicTrend (float64): 24h percentage-point weighted trend BTC*0.60+ETH*0.30+SOL*0.05+BNB*0.05; positive is bullish, negative bearish.
- NowPrice and NowSymbolClose (float64): selected symbol's latest stored/current close. NowSymbolPercentChange (float64): its 24h change in percentage points. NowSymbolOpen/Low/High (float64): its 24h open/low/high.
- BTCUSDT,ETHUSDT,SOLUSDT,BNBUSDT: benchmark objects with PercentChange (24h percentage points), Close,Open,Low,High.
- IsAsc(array)/IsDesc(array): strict increase/decrease in the supplied order; remember arrays are newest-to-oldest. KdjSimple(short,long,count): newest short is above long with exactly one upward cross and no recross in the inspected window; usable for any numeric arrays.
- Entry long/short only: Positions is the current local futures-position list. Each item has Symbol string; Side LONG|SHORT|BOTH; Amount numeric string (short normally negative); MarginType isolated|cross; Leverage int64; IsolatedWallet,EntryPrice,MarkPrice,UnrealizedProfit numeric strings; SourceType local|api; CreateTime milliseconds. Use float(...) for numeric strings. ROI and Position are unavailable.
- Exit close_long/close_short only: ROI float64 leveraged return in percentage points; Position has Symbol string, Side LONG|SHORT|BOTH, Amount float64 (short normally negative), Leverage int64, EntryPrice/MarkPrice/UnrealizedProfit float64, Mock bool, CreateTime milliseconds, SourceType local|api. Time exits require SourceType=local and CreateTime>0. Positions is unavailable.

Critical rules: for a live Donchian breakout compare kline close[0] with channel High[1]/Low[1], since channel[0] includes the forming candle. Close expr runs only after the symbol's outer profit/loss threshold is crossed. Unless narrowed by the user, produce enabled long,short,close_long,close_short rules. Never claim profitability or provide sizing, leverage, or order instructions.`

type strategyTemplateAIGenerationRequest struct {
	Prompt          string `json:"prompt"`
	PreviousJSON    string `json:"previousJson"`
	ValidationError string `json:"validationError"`
	ConversationID  string `json:"conversationId"`
}

type strategyTemplateAIAgentDecision struct {
	Action    string          `json:"action"`
	Summary   string          `json:"summary"`
	Tool      string          `json:"tool"`
	Arguments json.RawMessage `json:"arguments"`
	JSON      json.RawMessage `json:"json"`
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
	Tool     string    `json:"tool,omitempty"`
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
	Round           int                               `json:"round"`
	MaxRounds       int                               `json:"maxRounds"`
	Imported        bool                              `json:"imported"`
	CreatedAt       time.Time                         `json:"createdAt"`
	UpdatedAt       time.Time                         `json:"updatedAt"`
	CompletedAt     *time.Time                        `json:"completedAt,omitempty"`
	Messages        []llm.Message                     `json:"-"`
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

	task, err := startStrategyTemplateAITask(request)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
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

func (ctrl *StrategyTemplateController) ImportAIGeneratedTemplate() {
	taskID := strings.TrimSpace(ctrl.Ctx.Input.Param(":taskId"))
	if taskID == "" {
		ctrl.strategyTemplateImportError("taskId 不能为空")
		return
	}
	if err := ensureStrategyTemplateAITaskCanImport(taskID); err != nil {
		ctrl.strategyTemplateImportError(err.Error())
		return
	}

	data := ctrl.Ctx.Input.RequestBody
	action, stored, err := importStrategyTemplateData(data)
	if err != nil {
		recordStrategyTemplateAIImportError(taskID, string(data), err.Error())
		ctrl.strategyTemplateImportError(err.Error())
		return
	}
	markStrategyTemplateAITaskImported(taskID)
	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"action":   action,
			"template": stored,
		},
		"msg": "success",
	})
}

func startStrategyTemplateAITask(request strategyTemplateAIGenerationRequest) (strategyTemplateAIGenerationTask, error) {
	now := time.Now().UTC()
	strategyTemplateAITaskStore.Lock()
	cleanupStrategyTemplateAITasksLocked(now)

	taskID := strings.TrimSpace(request.ConversationID)
	var task *strategyTemplateAIGenerationTask
	if taskID != "" {
		task = strategyTemplateAITaskStore.tasks[taskID]
		if task == nil {
			strategyTemplateAITaskStore.Unlock()
			return strategyTemplateAIGenerationTask{}, fmt.Errorf("续聊任务不存在或已过期")
		}
		if task.Status == "queued" || task.Status == "running" {
			strategyTemplateAITaskStore.Unlock()
			return strategyTemplateAIGenerationTask{}, fmt.Errorf("当前生成任务仍在运行")
		}
		if task.Imported {
			strategyTemplateAITaskStore.Unlock()
			return strategyTemplateAIGenerationTask{}, fmt.Errorf("该对话已成功导入，不能继续生成")
		}
		task.Status = "queued"
		task.Progress = 0
		task.Stage = "queued"
		task.Round = 0
		task.Error = ""
		task.CompletedAt = nil
		appendStrategyTemplateAIEventLocked(task, 0, "queued", "已保留历史对话，开始新一轮生成")
	} else {
		taskID = newStrategyTemplateAITaskID()
		task = &strategyTemplateAIGenerationTask{
			TaskID:    taskID,
			Status:    "queued",
			Progress:  0,
			Stage:     "queued",
			MaxRounds: maxStrategyTemplateAIRounds,
			CreatedAt: now,
			UpdatedAt: now,
		}
		strategyTemplateAITaskStore.tasks[taskID] = task
		appendStrategyTemplateAIEventLocked(task, 0, "queued", "生成任务已创建")
	}
	task.MaxRounds = maxStrategyTemplateAIRounds
	result := cloneStrategyTemplateAITask(task)
	strategyTemplateAITaskStore.Unlock()

	go runStrategyTemplateAITask(taskID, request)
	return result, nil
}

func runStrategyTemplateAITask(taskID string, request strategyTemplateAIGenerationRequest) {
	updateStrategyTemplateAITask(taskID, 5, "initializing_llm", "正在初始化 LLM 客户端")
	client, err := llm.NewFromConfig()
	if err != nil {
		failStrategyTemplateAITask(taskID, "LLM 配置不可用: "+err.Error())
		return
	}

	updateStrategyTemplateAITask(taskID, 8, "building_prompt", "正在整理策略需求和框架约束")
	userPrompt := buildStrategyTemplateAIUserPrompt(request)
	appendStrategyTemplateAIMessage(taskID, llm.RoleUser, userPrompt)
	requiredTools := requiredStrategyTemplateAITools(request.Prompt)
	calledTools := make(map[string]bool, len(requiredTools))

	lastError := "AI 未在限定轮数内返回合法策略 JSON"
	for round := 1; round <= maxStrategyTemplateAIRounds; round++ {
		progress := 8 + round*7
		updateStrategyTemplateAIRound(taskID, round, progress, fmt.Sprintf("Agent 第 %d/%d 轮：正在分析上下文并决定下一步", round, maxStrategyTemplateAIRounds))
		messages, exists := getStrategyTemplateAIMessages(taskID)
		if !exists {
			return
		}
		response, err := generateStrategyTemplateWithProgress(taskID, client, messages, round)
		if err != nil {
			logs.Error("strategy template AI task %s round %d: %s", taskID, round, err.Error())
			failStrategyTemplateAITask(taskID, "Agent 第 "+strconv.Itoa(round)+" 轮调用失败: "+truncateStrategyTemplateAIError(err.Error()))
			return
		}
		if response == nil {
			failStrategyTemplateAITask(taskID, fmt.Sprintf("Agent 第 %d 轮调用失败: LLM 返回空响应", round))
			return
		}
		appendStrategyTemplateAIMessage(taskID, llm.RoleAssistant, response.Content)

		if isLLMResponseTruncated(response.FinishReason) {
			lastError = "AI 响应因 max_tokens 限制被截断，请输出更紧凑的响应或提高配置"
			recordStrategyTemplateAIRepair(taskID, lastError, "repairing_response")
			continue
		}

		decision, err := parseStrategyTemplateAIAgentDecision(response.Content)
		if err != nil {
			lastError = "Agent 响应协议错误: " + err.Error()
			recordStrategyTemplateAIRepair(taskID, lastError, "repairing_response")
			continue
		}

		switch decision.Action {
		case "tool":
			if strings.TrimSpace(decision.Tool) == "" {
				lastError = "Agent 选择调用工具但未提供 tool"
				recordStrategyTemplateAIRepair(taskID, lastError, "tool_error")
				continue
			}
			message := fmt.Sprintf("Agent 第 %d 轮决定调用 %s", round, decision.Tool)
			if strings.TrimSpace(decision.Summary) != "" {
				message += "：" + truncateStrategyTemplateAIEventMessage(decision.Summary)
			}
			updateStrategyTemplateAIToolEvent(taskID, progress+2, "calling_tool", decision.Tool, message)
			toolResult, toolErr := executeStrategyTemplateAITool(decision.Tool, decision.Arguments)
			if toolErr != nil {
				lastError = fmt.Sprintf("工具 %s 调用失败: %s", decision.Tool, truncateStrategyTemplateAIError(toolErr.Error()))
				appendStrategyTemplateAIMessage(taskID, llm.RoleUser, buildStrategyTemplateAIToolResultMessage(decision.Tool, "", toolErr))
				updateStrategyTemplateAIToolEvent(taskID, progress+3, "tool_error", decision.Tool, lastError)
				continue
			}
			appendStrategyTemplateAIMessage(taskID, llm.RoleUser, buildStrategyTemplateAIToolResultMessage(decision.Tool, toolResult, nil))
			updateStrategyTemplateAIToolEvent(taskID, progress+3, "tool_result", decision.Tool, fmt.Sprintf("%s 已返回数据，下一轮将总结并继续分析", decision.Tool))
			calledTools[decision.Tool] = true
			lastError = fmt.Sprintf("Agent 第 %d 轮调用了 %s，但尚未返回最终 JSON", round, decision.Tool)
			continue
		case "final":
			if missingTools := missingStrategyTemplateAITools(requiredTools, calledTools); len(missingTools) > 0 {
				lastError = "用户已明确要求使用工具，返回最终 JSON 前必须先成功调用: " + strings.Join(missingTools, ", ")
				recordStrategyTemplateAIRepair(taskID, lastError, "repairing_response")
				continue
			}
			generatedJSON := strings.TrimSpace(string(decision.JSON))
			if generatedJSON == "" || generatedJSON == "null" {
				lastError = "Agent final 响应缺少 json 对象"
				recordStrategyTemplateAIRepair(taskID, lastError, "repairing_json")
				continue
			}
			generatedJSON = formatStrategyTemplateJSON(generatedJSON)
			updateStrategyTemplateAITask(taskID, minStrategyTemplateAIProgress(progress+4, 95), "validating_json", fmt.Sprintf("Agent 第 %d 轮正在校验 JSON、指标参数和 expr 规则", round))
			if err := validateGeneratedStrategyTemplateJSON([]byte(generatedJSON)); err != nil {
				lastError = err.Error()
				recordStrategyTemplateAICandidate(taskID, generatedJSON, lastError)
				recordStrategyTemplateAIRepair(taskID, "JSON 校验失败: "+lastError, "repairing_json")
				continue
			}
			finishStrategyTemplateAITask(taskID, generatedJSON, decision.Summary)
			return
		default:
			lastError = fmt.Sprintf("Agent 返回未知 action %q，仅支持 tool 或 final", decision.Action)
			recordStrategyTemplateAIRepair(taskID, lastError, "repairing_response")
		}
	}

	failStrategyTemplateAITask(taskID, fmt.Sprintf("Agent 已达到最大 %d 轮，最后错误: %s", maxStrategyTemplateAIRounds, truncateStrategyTemplateAIError(lastError)))
}

func generateStrategyTemplateWithProgress(taskID string, client llm.Client, messages []llm.Message, round int) (*llm.Response, error) {
	type result struct {
		response *llm.Response
		err      error
	}
	resultChannel := make(chan result, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		response, err := client.Generate(ctx, llm.Request{
			System:   strategyTemplateAISystemPrompt,
			Messages: messages,
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
			progress := 8 + round*7 + waitedSeconds/5
			progress = minStrategyTemplateAIProgress(progress, 94)
			updateStrategyTemplateAITask(taskID, progress, "waiting_llm", fmt.Sprintf("Agent 第 %d/%d 轮等待 AI 响应，已等待 %d 秒", round, maxStrategyTemplateAIRounds, waitedSeconds))
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

func requiredStrategyTemplateAITools(prompt string) []string {
	normalized := strings.Join(strings.Fields(strings.ToLower(prompt)), "")
	required := make([]string, 0, 2)
	testResultMarkers := []string{
		"调用测试结果", "查询测试结果", "获取测试结果", "查看测试结果", "分析测试结果", "testresults",
	}
	for _, marker := range testResultMarkers {
		if strings.Contains(normalized, marker) {
			required = append(required, "get_test_strategy_results")
			break
		}
	}

	requestsData := strings.Contains(normalized, "获取") || strings.Contains(normalized, "查询") || strings.Contains(normalized, "get") || strings.Contains(normalized, "query")
	mentionsInstrument := strings.Contains(normalized, "币") || strings.Contains(normalized, "合约") || strings.Contains(normalized, "coin") || strings.Contains(normalized, "contract") || strings.Contains(normalized, "symbol") || strings.Contains(normalized, "usdt") || strings.Contains(normalized, "usdc")
	mentionsData := strings.Contains(normalized, "数据") || strings.Contains(normalized, "data")
	if requestsData && mentionsInstrument && mentionsData {
		required = append(required, "get_features")
	}
	return required
}

func missingStrategyTemplateAITools(required []string, called map[string]bool) []string {
	missing := make([]string, 0, len(required))
	for _, tool := range required {
		if !called[tool] {
			missing = append(missing, tool)
		}
	}
	return missing
}

func parseStrategyTemplateAIAgentDecision(content string) (strategyTemplateAIAgentDecision, error) {
	var decision strategyTemplateAIAgentDecision
	value := extractStrategyTemplateJSON(content)
	if strings.TrimSpace(value) == "" {
		return decision, fmt.Errorf("AI 未返回可用内容")
	}
	if err := json.Unmarshal([]byte(value), &decision); err != nil {
		return decision, fmt.Errorf("无法解析响应 JSON: %w", err)
	}
	decision.Action = strings.ToLower(strings.TrimSpace(decision.Action))
	decision.Tool = strings.TrimSpace(decision.Tool)
	if decision.Action != "" {
		return decision, nil
	}

	var root map[string]json.RawMessage
	if err := json.Unmarshal([]byte(value), &root); err == nil && root["name"] != nil && root["technology"] != nil && root["strategy"] != nil {
		decision.Action = "final"
		decision.Summary = "模型直接返回了策略模板 JSON"
		decision.JSON = json.RawMessage(value)
		return decision, nil
	}
	return decision, fmt.Errorf("响应缺少 action 字段")
}

func buildStrategyTemplateAIToolResultMessage(toolName, result string, toolErr error) string {
	payload := map[string]interface{}{
		"tool": toolName,
		"ok":   toolErr == nil,
	}
	if toolErr != nil {
		payload["error"] = truncateStrategyTemplateAIError(toolErr.Error())
	} else {
		var value interface{}
		if err := json.Unmarshal([]byte(result), &value); err == nil {
			payload["result"] = value
		} else {
			payload["result"] = result
		}
	}
	data, _ := json.Marshal(payload)
	return "TOOL_RESULT\n" + string(data) + "\nSummarize the relevant facts in your next summary field, then choose another tool or return a final strategy JSON."
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

func updateStrategyTemplateAIToolEvent(taskID string, progress int, stage, tool, message string) {
	strategyTemplateAITaskStore.Lock()
	defer strategyTemplateAITaskStore.Unlock()
	task := strategyTemplateAITaskStore.tasks[taskID]
	if task == nil || (task.Status != "queued" && task.Status != "running") {
		return
	}
	task.Status = "running"
	appendStrategyTemplateAIToolEventLocked(task, progress, stage, tool, message)
}

func updateStrategyTemplateAIRound(taskID string, round, progress int, message string) {
	strategyTemplateAITaskStore.Lock()
	defer strategyTemplateAITaskStore.Unlock()
	task := strategyTemplateAITaskStore.tasks[taskID]
	if task == nil || (task.Status != "queued" && task.Status != "running") {
		return
	}
	task.Status = "running"
	task.Round = round
	appendStrategyTemplateAIEventLocked(task, progress, "agent_round", message)
}

func appendStrategyTemplateAIMessage(taskID, role, content string) {
	strategyTemplateAITaskStore.Lock()
	defer strategyTemplateAITaskStore.Unlock()
	task := strategyTemplateAITaskStore.tasks[taskID]
	if task == nil {
		return
	}
	lastIndex := len(task.Messages) - 1
	if lastIndex >= 0 && task.Messages[lastIndex].Role == role {
		task.Messages[lastIndex].Content += "\n\n" + content
	} else {
		task.Messages = append(task.Messages, llm.Message{Role: role, Content: content})
	}
	task.UpdatedAt = time.Now().UTC()
}

func getStrategyTemplateAIMessages(taskID string) ([]llm.Message, bool) {
	strategyTemplateAITaskStore.RLock()
	defer strategyTemplateAITaskStore.RUnlock()
	task := strategyTemplateAITaskStore.tasks[taskID]
	if task == nil {
		return nil, false
	}
	return append([]llm.Message(nil), task.Messages...), true
}

func recordStrategyTemplateAIRepair(taskID, errorMessage, stage string) {
	payload, _ := json.Marshal(map[string]string{"error": errorMessage})
	appendStrategyTemplateAIMessage(taskID, llm.RoleUser, "AGENT_FEEDBACK\n"+string(payload)+"\nCorrect the error using the full conversation. Return one tool or final response envelope.")
	strategyTemplateAITaskStore.Lock()
	defer strategyTemplateAITaskStore.Unlock()
	task := strategyTemplateAITaskStore.tasks[taskID]
	if task == nil || task.Status != "running" {
		return
	}
	task.ValidationError = errorMessage
	appendStrategyTemplateAIEventLocked(task, minStrategyTemplateAIProgress(task.Progress+1, 95), stage, truncateStrategyTemplateAIEventMessage(errorMessage))
}

func recordStrategyTemplateAICandidate(taskID, generatedJSON, validationError string) {
	strategyTemplateAITaskStore.Lock()
	defer strategyTemplateAITaskStore.Unlock()
	task := strategyTemplateAITaskStore.tasks[taskID]
	if task == nil {
		return
	}
	task.JSON = generatedJSON
	task.ValidationError = validationError
	task.UpdatedAt = time.Now().UTC()
}

func appendStrategyTemplateAIEventLocked(task *strategyTemplateAIGenerationTask, progress int, stage, message string) {
	appendStrategyTemplateAIToolEventLocked(task, progress, stage, "", message)
}

func appendStrategyTemplateAIToolEventLocked(task *strategyTemplateAIGenerationTask, progress int, stage, tool, message string) {
	now := time.Now().UTC()
	if stage != "queued" && progress < task.Progress {
		progress = task.Progress
	}
	task.Progress = progress
	task.Stage = stage
	task.UpdatedAt = now
	task.Events = append(task.Events, strategyTemplateAIProgressEvent{Progress: progress, Stage: stage, Tool: tool, Message: message, Time: now})
}

func finishStrategyTemplateAITask(taskID, generatedJSON, summary string) {
	strategyTemplateAITaskStore.Lock()
	defer strategyTemplateAITaskStore.Unlock()
	task := strategyTemplateAITaskStore.tasks[taskID]
	if task == nil {
		return
	}
	task.JSON = generatedJSON
	task.ValidationError = ""
	task.Error = ""
	task.Status = "succeeded"
	message := "JSON 已生成并通过框架校验"
	if strings.TrimSpace(summary) != "" {
		message += "：" + truncateStrategyTemplateAIEventMessage(summary)
	}
	appendStrategyTemplateAIEventLocked(task, 100, "completed", message)
	completedAt := task.UpdatedAt
	task.CompletedAt = &completedAt
}

func ensureStrategyTemplateAITaskCanImport(taskID string) error {
	strategyTemplateAITaskStore.RLock()
	defer strategyTemplateAITaskStore.RUnlock()
	task := strategyTemplateAITaskStore.tasks[taskID]
	if task == nil {
		return fmt.Errorf("AI 生成任务不存在或已过期")
	}
	if task.Status == "queued" || task.Status == "running" {
		return fmt.Errorf("AI 生成任务仍在运行，暂时不能导入")
	}
	if task.Imported {
		return fmt.Errorf("该 AI 生成任务已经成功导入")
	}
	return nil
}

func recordStrategyTemplateAIImportError(taskID, generatedJSON, errorMessage string) {
	payload, _ := json.Marshal(map[string]string{
		"error": errorMessage,
		"json":  generatedJSON,
	})
	appendStrategyTemplateAIMessage(taskID, llm.RoleUser, "IMPORT_ERROR\n"+string(payload)+"\nThe user may provide a revised prompt. Preserve prior evidence and generate a complete corrected JSON when asked.")
	strategyTemplateAITaskStore.Lock()
	defer strategyTemplateAITaskStore.Unlock()
	task := strategyTemplateAITaskStore.tasks[taskID]
	if task == nil {
		return
	}
	task.JSON = generatedJSON
	task.ValidationError = errorMessage
	appendStrategyTemplateAIEventLocked(task, 100, "import_failed", "导入失败："+truncateStrategyTemplateAIEventMessage(errorMessage))
}

func markStrategyTemplateAITaskImported(taskID string) {
	strategyTemplateAITaskStore.Lock()
	defer strategyTemplateAITaskStore.Unlock()
	task := strategyTemplateAITaskStore.tasks[taskID]
	if task == nil {
		return
	}
	task.Imported = true
	task.Status = "succeeded"
	task.Error = ""
	task.ValidationError = ""
	task.Messages = nil
	appendStrategyTemplateAIEventLocked(task, 100, "imported", "策略模板已成功导入，对话上下文已结束")
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
	cloned.Messages = nil
	return cloned
}

func cleanupStrategyTemplateAITasksLocked(now time.Time) {
	for taskID, task := range strategyTemplateAITaskStore.tasks {
		if task.Status == "queued" || task.Status == "running" || !task.Imported || now.Sub(task.UpdatedAt) <= strategyTemplateAITaskRetention {
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

func minStrategyTemplateAIProgress(value, maximum int) int {
	if value > maximum {
		return maximum
	}
	return value
}

func truncateStrategyTemplateAIEventMessage(message string) string {
	const maxLength = 300
	value := []rune(strings.TrimSpace(message))
	if len(value) <= maxLength {
		return string(value)
	}
	return string(value[:maxLength]) + "..."
}

func truncateStrategyTemplateAIError(message string) string {
	const maxLength = 1000
	message = strings.TrimSpace(message)
	if len(message) <= maxLength {
		return message
	}
	return message[:maxLength] + "..."
}
