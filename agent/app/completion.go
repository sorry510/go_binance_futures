package app

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	agentruntime "go_binance_futures/agent/runtime"
	marketregime "go_binance_futures/agent/skills/marketregime"
	symbolanalysis "go_binance_futures/agent/skills/symbolanalysis"
	"go_binance_futures/agent/task"
	marketservice "go_binance_futures/service/market"
	symbolservice "go_binance_futures/service/symbol"
	symbolanalysisservice "go_binance_futures/service/symbolanalysis"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/core/logs"
)

func persistTaskCompletion(req agentruntime.Request, item *task.Task, result *agentruntime.Result, runErr error) error {
	if item == nil {
		return nil
	}
	if err := persistChatCompletion(item, result); err != nil {
		logs.Error("persist chat completion:", err)
	}
	if req.Skill == marketregime.Name {
		if job, _ := req.Metadata["scheduler_job"].(string); job == "market_regime" {
			return persistMarketRegimeCompletion(item, result)
		}
		return nil
	}
	if req.Skill != symbolanalysis.Name {
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

func persistMarketRegimeCompletion(item *task.Task, result *agentruntime.Result) error {
	if item.Status != task.StatusSucceeded {
		return nil
	}
	raw := item.Result
	if result != nil && len(result.Raw) > 0 {
		raw = result.Raw
	}
	var analysis marketregime.Analysis
	if err := json.Unmarshal(raw, &analysis); err != nil {
		return err
	}
	cfg, err := utils.GetSystemConfig()
	if err != nil {
		return err
	}
	return marketservice.SaveMarketCondition(context.Background(), cfg.ID, analysis.MarketCondition)
}

func EnsureCompletion(item *task.Task) error {
	if item == nil || !task.IsTerminalStatus(item.Status) {
		return nil
	}
	if err := persistChatCompletion(item, nil); err != nil {
		logs.Error("ensure chat completion:", err)
	}
	if item.Skill == marketregime.Name {
		return nil
	}
	if item.Skill != symbolanalysis.Name {
		return nil
	}
	var result *agentruntime.Result
	if len(item.Result) > 0 {
		result = &agentruntime.Result{TaskID: item.ID, Skill: item.Skill, Raw: append(json.RawMessage(nil), item.Result...)}
	}
	return persistTaskCompletion(agentruntime.Request{Skill: item.Skill, Input: item.Input}, item, result, nil)
}
