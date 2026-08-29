package controllers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	agentruntime "go_binance_futures/agent/runtime"
	"go_binance_futures/agent/skill"
	strategybuilder "go_binance_futures/agent/skills/strategybuilder"
	"go_binance_futures/agent/task"
	agenttools "go_binance_futures/agent/tools"
	domaintools "go_binance_futures/agent/tools/domain"
	"go_binance_futures/llm"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/core/logs"
)

const (
	strategyTemplateAITaskRetention  = 30 * time.Minute
	maxStrategyTemplateAIPromptSize  = 12 * 1024
	maxStrategyTemplateAIContextSize = 256 * 1024
	maxStrategyTemplateAIRounds      = 10
)

type strategyTemplateAIGenerationRequest struct {
	Prompt          string `json:"prompt"`
	PreviousJSON    string `json:"previousJson"`
	ValidationError string `json:"validationError"`
	ConversationID  string `json:"conversationId"`
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

var newStrategyBuilderLLMClient = llm.NewFromConfig

var strategyTemplateAITaskStore = struct {
	sync.RWMutex
	tasks map[string]*strategyTemplateAIGenerationTask
}{tasks: make(map[string]*strategyTemplateAIGenerationTask)}

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
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": task, "msg": "accepted"})
}

func (ctrl *StrategyTemplateController) GetAIGenerationTask() {
	task, exists := getStrategyTemplateAITask(ctrl.Ctx.Input.Param(":taskId"))
	if !exists {
		ctrl.Ctx.Resp(utils.ResJson(404, nil, "task not found"))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": task, "msg": "success"})
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
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": map[string]interface{}{"action": action, "template": stored}, "msg": "success"})
}

func startStrategyTemplateAITask(request strategyTemplateAIGenerationRequest) (strategyTemplateAIGenerationTask, error) {
	now := time.Now().UTC()
	strategyTemplateAITaskStore.Lock()
	cleanupStrategyTemplateAITasksLocked(now)
	taskID := strings.TrimSpace(request.ConversationID)
	var item *strategyTemplateAIGenerationTask
	if taskID != "" {
		item = strategyTemplateAITaskStore.tasks[taskID]
		if item == nil {
			strategyTemplateAITaskStore.Unlock()
			return strategyTemplateAIGenerationTask{}, fmt.Errorf("续聊任务不存在或已过期")
		}
		if item.Status == "queued" || item.Status == "running" {
			strategyTemplateAITaskStore.Unlock()
			return strategyTemplateAIGenerationTask{}, fmt.Errorf("当前生成任务仍在运行")
		}
		if item.Imported {
			strategyTemplateAITaskStore.Unlock()
			return strategyTemplateAIGenerationTask{}, fmt.Errorf("该对话已成功导入，不能继续生成")
		}
		item.Status, item.Stage, item.Progress, item.Round = "queued", "queued", 0, 0
		item.Error, item.ValidationError = "", ""
		item.CompletedAt = nil
		appendStrategyTemplateAIEventLocked(item, 0, "queued", "已保留历史对话，开始新一轮生成")
	} else {
		taskID = newStrategyTemplateAITaskID()
		item = &strategyTemplateAIGenerationTask{TaskID: taskID, Status: "queued", Stage: "queued", MaxRounds: maxStrategyTemplateAIRounds, CreatedAt: now, UpdatedAt: now}
		strategyTemplateAITaskStore.tasks[taskID] = item
		appendStrategyTemplateAIEventLocked(item, 0, "queued", "生成任务已创建")
	}
	item.MaxRounds = maxStrategyTemplateAIRounds
	result := cloneStrategyTemplateAITask(item)
	strategyTemplateAITaskStore.Unlock()
	go runStrategyTemplateAITask(taskID, request)
	return result, nil
}

func runStrategyTemplateAITask(taskID string, request strategyTemplateAIGenerationRequest) {
	updateStrategyTemplateAITask(taskID, 5, "initializing_llm", "正在初始化 LLM 客户端")
	client, err := newStrategyBuilderLLMClient()
	if err != nil {
		failStrategyTemplateAITask(taskID, "LLM 配置不可用: "+err.Error())
		return
	}
	history, exists := getStrategyTemplateAIMessages(taskID)
	if !exists {
		return
	}
	input := strategybuilder.Input{Prompt: request.Prompt, PreviousJSON: request.PreviousJSON, ValidationError: request.ValidationError}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		failStrategyTemplateAITask(taskID, "构造 Agent 输入失败: "+err.Error())
		return
	}
	appendStrategyTemplateAIMessage(taskID, llm.RoleUser, strategybuilder.BuildUserPrompt(input))

	skills := skill.NewRegistry()
	builder := strategybuilder.New(strategybuilder.Options{
		Validate:               validateGeneratedStrategyTemplateJSON,
		RepairGuidance:         buildStrategyTemplateAIRepairGuidance,
		RequireMarketCondition: strategybuilder.RequiresMarketConditionForConversation(request.Prompt, history),
	})
	if err := skills.Register(builder); err != nil {
		failStrategyTemplateAITask(taskID, "注册 Strategy Builder Skill 失败: "+err.Error())
		return
	}
	toolRegistry := agenttools.NewRegistry()
	if err := domaintools.RegisterReadOnly(toolRegistry, domaintools.DefaultDependencies()); err != nil {
		failStrategyTemplateAITask(taskID, "注册 Agent Tool 失败: "+err.Error())
		return
	}

	runner, err := agentruntime.NewRunner(agentruntime.Config{
		Client: client, Skills: skills, Tools: toolRegistry,
		Timeout: 15 * time.Minute, DefaultMaxRounds: maxStrategyTemplateAIRounds,
		MaxContextBytes: maxStrategyTemplateAIContextSize, MaxToolResultBytes: 256 * 1024, MaxToolCalls: maxStrategyTemplateAIRounds,
		Retry:     agentruntime.RetryPolicy{MaxAttempts: 2, Delay: time.Second},
		EventHook: func(event task.Event) { handleStrategyTemplateAIRuntimeEvent(taskID, event) },
		MessageHook: func(_ string, message llm.Message) {
			appendStrategyTemplateAIMessage(taskID, message.Role, message.Content)
		},
		ValidationHook: func(_ string, raw json.RawMessage, validationErr error) {
			if validationErr != nil {
				recordStrategyTemplateAICandidate(taskID, formatStrategyTemplateJSON(string(raw)), validationErr.Error())
			}
		},
	})
	if err != nil {
		failStrategyTemplateAITask(taskID, "初始化 Agent Runtime 失败: "+err.Error())
		return
	}

	result, err := runner.Run(context.Background(), agentruntime.Request{
		Skill: strategybuilder.Name, Input: string(inputJSON), ConversationID: taskID,
		Metadata: map[string]any{strategybuilder.HistoryMetadataKey: history},
	})
	if err != nil {
		logs.Error("strategy builder runtime task %s: %s", taskID, err.Error())
		failStrategyTemplateAITask(taskID, truncateStrategyTemplateAIError(err.Error()))
		return
	}
	finishStrategyTemplateAITask(taskID, formatStrategyTemplateJSON(string(result.Raw)), result.Summary)
}

func handleStrategyTemplateAIRuntimeEvent(taskID string, event task.Event) {
	progress := strategyTemplateAIRoundProgress(event.Round)
	switch event.Stage {
	case "building_input":
		updateStrategyTemplateAITask(taskID, 8, "building_prompt", "正在整理策略需求和框架约束")
	case "waiting_llm":
		updateStrategyTemplateAIRound(taskID, event.Round, progress, fmt.Sprintf("Agent 第 %d/%d 轮：正在等待 AI 响应", event.Round, maxStrategyTemplateAIRounds))
	case "retrying_llm":
		updateStrategyTemplateAITask(taskID, minStrategyTemplateAIProgress(progress+1, 94), "retrying_llm", "LLM 连接中断，正在自动重试")
	case "waiting_tool":
		updateStrategyTemplateAIToolEvent(taskID, minStrategyTemplateAIProgress(progress+2, 94), "calling_tool", event.Tool, event.Message)
	case "tool_result":
		stage := "tool_result"
		if event.Status == "error" {
			stage = "tool_error"
		}
		message := fmt.Sprintf("%s 已返回结果，耗时 %d ms", event.Tool, event.DurationMs)
		updateStrategyTemplateAIToolEvent(taskID, minStrategyTemplateAIProgress(progress+3, 95), stage, event.Tool, message)
	case "validating":
		updateStrategyTemplateAITask(taskID, minStrategyTemplateAIProgress(progress+4, 95), "validating_json", "正在校验 JSON、指标参数和 expr 规则")
	case "repairing_final":
		updateStrategyTemplateAITask(taskID, minStrategyTemplateAIProgress(progress+5, 96), "repairing_json", truncateStrategyTemplateAIEventMessage(event.Message))
	case "repairing_required_tools", "repairing_decision", "repairing_response":
		updateStrategyTemplateAITask(taskID, minStrategyTemplateAIProgress(progress+2, 95), "repairing_response", truncateStrategyTemplateAIEventMessage(event.Message))
	}
}

func updateStrategyTemplateAITask(taskID string, progress int, stage, message string) {
	strategyTemplateAITaskStore.Lock()
	defer strategyTemplateAITaskStore.Unlock()
	item := strategyTemplateAITaskStore.tasks[taskID]
	if item == nil || (item.Status != "queued" && item.Status != "running") {
		return
	}
	item.Status = "running"
	appendStrategyTemplateAIEventLocked(item, progress, stage, message)
}

func updateStrategyTemplateAIToolEvent(taskID string, progress int, stage, tool, message string) {
	strategyTemplateAITaskStore.Lock()
	defer strategyTemplateAITaskStore.Unlock()
	item := strategyTemplateAITaskStore.tasks[taskID]
	if item == nil || (item.Status != "queued" && item.Status != "running") {
		return
	}
	item.Status = "running"
	appendStrategyTemplateAIToolEventLocked(item, progress, stage, tool, message)
}

func updateStrategyTemplateAIRound(taskID string, round, progress int, message string) {
	strategyTemplateAITaskStore.Lock()
	defer strategyTemplateAITaskStore.Unlock()
	item := strategyTemplateAITaskStore.tasks[taskID]
	if item == nil || (item.Status != "queued" && item.Status != "running") {
		return
	}
	item.Status, item.Round = "running", round
	appendStrategyTemplateAIEventLocked(item, progress, "agent_round", message)
}

func appendStrategyTemplateAIMessage(taskID, role, content string) {
	strategyTemplateAITaskStore.Lock()
	defer strategyTemplateAITaskStore.Unlock()
	item := strategyTemplateAITaskStore.tasks[taskID]
	if item == nil {
		return
	}
	appendStrategyTemplateAIMessageLocked(item, role, content)
}

func appendStrategyTemplateAIMessageLocked(item *strategyTemplateAIGenerationTask, role, content string) {
	last := len(item.Messages) - 1
	if last >= 0 && item.Messages[last].Role == role {
		item.Messages[last].Content += "\n\n" + content
	} else {
		item.Messages = append(item.Messages, llm.Message{Role: role, Content: content})
	}
	item.UpdatedAt = time.Now().UTC()
}

func getStrategyTemplateAIMessages(taskID string) ([]llm.Message, bool) {
	strategyTemplateAITaskStore.RLock()
	defer strategyTemplateAITaskStore.RUnlock()
	item := strategyTemplateAITaskStore.tasks[taskID]
	if item == nil {
		return nil, false
	}
	return append([]llm.Message(nil), item.Messages...), true
}

func recordStrategyTemplateAICandidate(taskID, generatedJSON, validationError string) {
	strategyTemplateAITaskStore.Lock()
	defer strategyTemplateAITaskStore.Unlock()
	item := strategyTemplateAITaskStore.tasks[taskID]
	if item == nil {
		return
	}
	item.JSON = generatedJSON
	item.ValidationError = validationError
	item.UpdatedAt = time.Now().UTC()
}

func appendStrategyTemplateAIEventLocked(item *strategyTemplateAIGenerationTask, progress int, stage, message string) {
	appendStrategyTemplateAIToolEventLocked(item, progress, stage, "", message)
}

func appendStrategyTemplateAIToolEventLocked(item *strategyTemplateAIGenerationTask, progress int, stage, tool, message string) {
	now := time.Now().UTC()
	if stage != "queued" && progress < item.Progress {
		progress = item.Progress
	}
	item.Progress, item.Stage, item.UpdatedAt = progress, stage, now
	item.Events = append(item.Events, strategyTemplateAIProgressEvent{Progress: progress, Stage: stage, Tool: tool, Message: message, Time: now})
}

func finishStrategyTemplateAITask(taskID, generatedJSON, summary string) {
	strategyTemplateAITaskStore.Lock()
	defer strategyTemplateAITaskStore.Unlock()
	item := strategyTemplateAITaskStore.tasks[taskID]
	if item == nil {
		return
	}
	item.JSON, item.ValidationError, item.Error, item.Status = generatedJSON, "", "", "succeeded"
	message := "JSON 已生成并通过框架校验"
	if strings.TrimSpace(summary) != "" {
		message += "：" + truncateStrategyTemplateAIEventMessage(summary)
	}
	appendStrategyTemplateAIEventLocked(item, 100, "completed", message)
	completedAt := item.UpdatedAt
	item.CompletedAt = &completedAt
}

func ensureStrategyTemplateAITaskCanImport(taskID string) error {
	strategyTemplateAITaskStore.RLock()
	defer strategyTemplateAITaskStore.RUnlock()
	item := strategyTemplateAITaskStore.tasks[taskID]
	if item == nil {
		return fmt.Errorf("AI 生成任务不存在或已过期")
	}
	if item.Status == "queued" || item.Status == "running" {
		return fmt.Errorf("AI 生成任务仍在运行，暂时不能导入")
	}
	if item.Imported {
		return fmt.Errorf("该 AI 生成任务已经成功导入")
	}
	return nil
}

func recordStrategyTemplateAIImportError(taskID, generatedJSON, errorMessage string) {
	payload, _ := json.Marshal(map[string]string{"error": errorMessage, "json": generatedJSON})
	appendStrategyTemplateAIMessage(taskID, llm.RoleUser, "IMPORT_ERROR\n"+string(payload)+"\nThe user may provide a revised prompt. Preserve prior evidence and generate a complete corrected JSON when asked.")
	strategyTemplateAITaskStore.Lock()
	defer strategyTemplateAITaskStore.Unlock()
	item := strategyTemplateAITaskStore.tasks[taskID]
	if item == nil {
		return
	}
	item.JSON, item.ValidationError = generatedJSON, errorMessage
	appendStrategyTemplateAIEventLocked(item, 100, "import_failed", "导入失败："+truncateStrategyTemplateAIEventMessage(errorMessage))
}

func markStrategyTemplateAITaskImported(taskID string) {
	strategyTemplateAITaskStore.Lock()
	defer strategyTemplateAITaskStore.Unlock()
	item := strategyTemplateAITaskStore.tasks[taskID]
	if item == nil {
		return
	}
	item.Imported, item.Status = true, "succeeded"
	item.Error, item.ValidationError, item.Messages = "", "", nil
	appendStrategyTemplateAIEventLocked(item, 100, "imported", "策略模板已成功导入，对话上下文已结束")
	completedAt := item.UpdatedAt
	item.CompletedAt = &completedAt
}

func failStrategyTemplateAITask(taskID, errorMessage string) {
	strategyTemplateAITaskStore.Lock()
	defer strategyTemplateAITaskStore.Unlock()
	item := strategyTemplateAITaskStore.tasks[taskID]
	if item == nil {
		return
	}
	item.Status, item.Error = "failed", errorMessage
	appendStrategyTemplateAIEventLocked(item, 100, "failed", errorMessage)
	completedAt := item.UpdatedAt
	item.CompletedAt = &completedAt
}

func getStrategyTemplateAITask(taskID string) (strategyTemplateAIGenerationTask, bool) {
	strategyTemplateAITaskStore.RLock()
	defer strategyTemplateAITaskStore.RUnlock()
	item, exists := strategyTemplateAITaskStore.tasks[taskID]
	if !exists {
		return strategyTemplateAIGenerationTask{}, false
	}
	return cloneStrategyTemplateAITask(item), true
}

func cloneStrategyTemplateAITask(item *strategyTemplateAIGenerationTask) strategyTemplateAIGenerationTask {
	cloned := *item
	cloned.Events = append([]strategyTemplateAIProgressEvent(nil), item.Events...)
	cloned.Messages = nil
	return cloned
}

func cleanupStrategyTemplateAITasksLocked(now time.Time) {
	for taskID, item := range strategyTemplateAITaskStore.tasks {
		if item.Status == "queued" || item.Status == "running" || !item.Imported || now.Sub(item.UpdatedAt) <= strategyTemplateAITaskRetention {
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

func minStrategyTemplateAIProgress(value, maximum int) int {
	if value > maximum {
		return maximum
	}
	return value
}

func strategyTemplateAIRoundProgress(round int) int {
	if round < 1 {
		return 8
	}
	if round > maxStrategyTemplateAIRounds {
		round = maxStrategyTemplateAIRounds
	}
	return 8 + round*82/maxStrategyTemplateAIRounds
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
