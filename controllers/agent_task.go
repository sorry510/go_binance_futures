package controllers

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"time"

	agentmanager "go_binance_futures/agent/manager"
	agentruntime "go_binance_futures/agent/runtime"
	"go_binance_futures/agent/skill"
	symbolanalysis "go_binance_futures/agent/skills/symbolanalysis"
	"go_binance_futures/agent/task"
	agenttools "go_binance_futures/agent/tools"
	domaintools "go_binance_futures/agent/tools/domain"
	symbolservice "go_binance_futures/service/symbol"
	symbolanalysisservice "go_binance_futures/service/symbolanalysis"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
)

type AgentController struct {
	web.Controller
}

type agentTaskRequest struct {
	Skill string          `json:"skill"`
	Input json.RawMessage `json:"input"`
}

var defaultAgentManagerOnce sync.Once
var defaultAgentManager *agentmanager.Manager
var defaultAgentManagerErr error

func getDefaultAgentManager() (*agentmanager.Manager, error) {
	defaultAgentManagerOnce.Do(func() {
		skills := skill.NewRegistry()
		if err := skills.Register(symbolanalysis.New()); err != nil {
			defaultAgentManagerErr = err
			return
		}
		tools := agenttools.NewRegistry()
		if err := domaintools.RegisterReadOnly(tools, domaintools.DefaultDependencies()); err != nil {
			defaultAgentManagerErr = err
			return
		}
		defaultAgentManager, defaultAgentManagerErr = agentmanager.New(agentmanager.Config{
			Skills:         skills,
			Tools:          tools,
			CompletionHook: persistAgentTaskCompletion,
			RuntimeConfig: agentruntime.Config{
				Timeout:            3 * time.Minute,
				DefaultMaxRounds:   8,
				MaxContextBytes:    256 * 1024,
				MaxToolResultBytes: 256 * 1024,
				MaxToolCalls:       12,
				Retry:              agentruntime.RetryPolicy{MaxAttempts: 2, Delay: time.Second},
			},
		})
	})
	return defaultAgentManager, defaultAgentManagerErr
}

func (ctrl *AgentController) StartTask() {
	var request agentTaskRequest
	if err := json.Unmarshal(ctrl.Ctx.Input.RequestBody, &request); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "请求格式错误: "+err.Error()))
		return
	}
	request.Skill = strings.TrimSpace(request.Skill)
	if request.Skill == "" || len(request.Input) == 0 || string(request.Input) == "null" {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "skill 和 input 不能为空"))
		return
	}
	manager, err := getDefaultAgentManager()
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(500, nil, "初始化 Agent Manager 失败: "+err.Error()))
		return
	}
	item, err := manager.Start(agentruntime.Request{Skill: request.Skill, Input: string(request.Input)})
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": item, "msg": "accepted"})
}

func (ctrl *AgentController) GetTask() {
	manager, err := getDefaultAgentManager()
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(500, nil, "初始化 Agent Manager 失败: "+err.Error()))
		return
	}
	item, err := manager.Get(ctrl.Ctx.Request.Context(), ctrl.Ctx.Input.Param(":taskId"))
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(404, nil, "task not found"))
		return
	}
	if item.Skill == symbolanalysis.Name && (item.Status == task.StatusSucceeded || item.Status == task.StatusFailed || item.Status == task.StatusCancelled) {
		var result *agentruntime.Result
		if len(item.Result) > 0 {
			result = &agentruntime.Result{TaskID: item.ID, Skill: item.Skill, Raw: append(json.RawMessage(nil), item.Result...)}
		}
		_ = persistAgentTaskCompletion(agentruntime.Request{Skill: item.Skill, Input: item.Input}, item, result, nil)
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": item, "msg": "success"})
}

func persistAgentTaskCompletion(req agentruntime.Request, item *task.Task, result *agentruntime.Result, runErr error) error {
	if req.Skill != symbolanalysis.Name || item == nil {
		return nil
	}
	var input symbolanalysis.Input
	if err := json.Unmarshal([]byte(req.Input), &input); err != nil {
		return err
	}
	input.Symbol = strings.ToUpper(strings.TrimSpace(input.Symbol))
	analysisPrice := 0.0
	if snapshot, err := (symbolservice.Service{}).Snapshot(context.Background(), input.Symbol); err == nil {
		analysisPrice, _ = strconv.ParseFloat(snapshot.Close, 64)
	}
	var raw json.RawMessage
	if result != nil {
		raw = append(json.RawMessage(nil), result.Raw...)
	}
	errorMessage := item.Error
	if errorMessage == "" && runErr != nil {
		errorMessage = runErr.Error()
	}
	completedAt := time.Now().UTC()
	if item.CompletedAt != nil {
		completedAt = item.CompletedAt.UTC()
	}
	err := (symbolanalysisservice.HistoryService{}).Save(context.Background(), symbolanalysisservice.HistorySaveRequest{
		TaskID: item.ID, Symbol: input.Symbol, Prompt: input.Prompt,
		Status: string(item.Status), Result: raw, Error: errorMessage,
		Provider: item.Provider, Model: item.Model, AnalysisPrice: analysisPrice,
		CreatedAt: item.CreatedAt.UnixMilli(), CompletedAt: completedAt.UnixMilli(),
	})
	if err != nil {
		logs.Error("persist symbol analysis history:", err)
	}
	return err
}

func (ctrl *AgentController) GetSymbolAnalysisHistory() {
	page, _ := strconv.Atoi(ctrl.GetString("page", "1"))
	limit, _ := strconv.Atoi(ctrl.GetString("limit", "20"))
	result, err := (symbolanalysisservice.HistoryService{}).List(
		ctrl.Ctx.Request.Context(),
		symbolanalysisservice.HistoryListOptions{
			Symbol: ctrl.GetString("symbol"), Status: ctrl.GetString("status"),
			Page: page, Limit: limit,
		},
	)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": result, "msg": "success"})
}
