package controllers

import (
	"encoding/json"
	"fmt"
	"math"

	"go_binance_futures/models"
	"go_binance_futures/notify"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/config"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
)

type IndexController struct {
	web.Controller
}

// respondServiceConfig sends JSON regardless of the request Accept header.
// Beego Context.Resp would negotiate XML/Protobuf for map payloads.
func (ctrl *IndexController) respondServiceConfig(payload interface{}) {
	_ = ctrl.JSONResp(payload)
}

func (ctrl *IndexController) GetServiceConfig() {
	var debug, _ = config.String("debug")
	systemConfig, err := utils.GetSystemConfig()
	if err != nil {
		logs.Error("GetSystemConfig:", err)
		ctrl.respondServiceConfig(utils.ResJson(500, nil, "cannot load configuration"))
		return
	}

	var coinMaxCount = systemConfig.FutureMaxCount
	var coinOrderType = systemConfig.FutureOrderType
	var coinAllowLong = systemConfig.FutureAllowLong
	var coinAllowShort = systemConfig.FutureAllowShort

	var tradeFutureTest = systemConfig.FutureTest
	var tradeFutureTestNoticeLimitMin = systemConfig.FutureTestNoticeLimitMin
	var tradeFutureEnable = systemConfig.FutureEnable
	var tradeSpotEnable = systemConfig.SpotEnable
	var tradeDeliveryEnable = systemConfig.DeliveryEnable
	var tradeStrategyCoin = systemConfig.FutureStrategyCoin
	var tradeNewEnable = systemConfig.FutureNewEnable

	var spotNewEnable = systemConfig.SpotNewEnable

	var noticeCoinEnable = systemConfig.NoticeCoinEnable

	var listenCoinEnable = systemConfig.ListenCoinEnable
	var listenFundingRate = systemConfig.ListenFundingRateEnable
	var lossMaxCount = systemConfig.LossMaxCount
	var futureTestAutoTradeCountLimit = systemConfig.FutureTestAutoTradeCountLimit
	var externalLinks, _ = config.String("external::links")

	ctrl.respondServiceConfig(map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"debug": debug,

			"futuresPositionConvertEnable": systemConfig.FuturesPositionConvertEnable,

			"coinMaxCount":   coinMaxCount,
			"coinOrderType":  coinOrderType,
			"coinAllowLong":  coinAllowLong,
			"coinAllowShort": coinAllowShort,

			"tradeFutureTest":               tradeFutureTest,
			"tradeFutureTestNoticeLimitMin": tradeFutureTestNoticeLimitMin,
			"tradeFutureEnable":             tradeFutureEnable,
			"tradeSpotEnable":               tradeSpotEnable,
			"spot_quant_enable":             systemConfig.SpotQuantEnable,
			"tradeDeliveryEnable":           tradeDeliveryEnable,
			"tradeStrategyCoin":             tradeStrategyCoin,
			"tradeNewEnable":                tradeNewEnable,

			"spotNewEnable": spotNewEnable,

			"noticeCoinEnable": noticeCoinEnable,

			"listenCoinEnable":                           listenCoinEnable,
			"listenFundingRate":                          listenFundingRate,
			"lossMaxCount":                               lossMaxCount,
			"lossAutoScale":                              systemConfig.LossAutoScale,
			"marketCondition":                            systemConfig.MarketCondition,
			"marketConditionIsAuto":                      systemConfig.MarketConditionIsAuto,
			"FutureTestAutoTradeCountLimit":              futureTestAutoTradeCountLimit,
			"FutureTestFeeRate":                          systemConfig.FutureTestFeeRate,
			"WsFuturesFastMoveEnable":                    systemConfig.WsFuturesFastMoveEnable,
			"WsFuturesFastMoveThreshold":                 systemConfig.WsFuturesFastMoveThreshold,
			"WsFuturesFastMoveRecover":                   systemConfig.WsFuturesFastMoveRecover,
			"WsFuturesFastMoveCooldownSec":               systemConfig.WsFuturesFastMoveCooldownSec,
			"WsFuturesFastMoveWindows":                   systemConfig.WsFuturesFastMoveWindows,
			"WsFuturesLiquidationEnable":                 systemConfig.WsFuturesLiquidationEnable,
			"WsFuturesLiquidationAlertWindowSec":         systemConfig.WsFuturesLiquidationAlertWindowSec,
			"WsFuturesLiquidationAlertNotionalThreshold": systemConfig.WsFuturesLiquidationAlertNotionalThreshold,
			"WsFuturesLiquidationAlertCooldownSec":       systemConfig.WsFuturesLiquidationAlertCooldownSec,
			"AgentAlertPipelineEnable":                   systemConfig.AgentAlertPipelineEnable,
			"AgentAlertAnalysisEnable":                   systemConfig.AgentAlertAnalysisEnable,
			"AgentAlertMinSeverity":                      systemConfig.AgentAlertMinSeverity,
			"AgentAlertCooldownSec":                      systemConfig.AgentAlertCooldownSec,
			"AgentAlertMaxConcurrent":                    systemConfig.AgentAlertMaxConcurrent,
			"AgentMarketRegimeScheduleEnable":            systemConfig.AgentMarketRegimeScheduleEnable,
			"AgentMarketRegimeIntervalMin":               systemConfig.AgentMarketRegimeIntervalMin,
			"AgentDailyMarketBriefScheduleEnable":        systemConfig.AgentDailyMarketBriefScheduleEnable,
			"AgentDailyMarketBriefIntervalMin":           systemConfig.AgentDailyMarketBriefIntervalMin,
			"AgentOpportunityWatchEnable":                systemConfig.AgentOpportunityWatchEnable,
			"AgentOpportunityScanIntervalMin":            systemConfig.AgentOpportunityScanIntervalMin,
			"AgentOpportunityMinConfidence":              systemConfig.AgentOpportunityMinConfidence,
			"AgentMaxStartsPerMinute":                    systemConfig.AgentMaxStartsPerMinute,
			"AgentMaxStartsPerHour":                      systemConfig.AgentMaxStartsPerHour,
			"AgentMaxRoundsPerTask":                      systemConfig.AgentMaxRoundsPerTask,
			"AgentMaxTokensPerTask":                      systemConfig.AgentMaxTokensPerTask,
			"AgentMaxToolCallsPerTask":                   systemConfig.AgentMaxToolCallsPerTask,
			"AgentTradeExecutionEnable":                  systemConfig.AgentTradeExecutionEnable,
			"AgentTradeAllowedSymbols":                   systemConfig.AgentTradeAllowedSymbols,
			"AgentTradeMaxRiskUSDT":                      systemConfig.AgentTradeMaxRiskUSDT,
			"AgentTradeMaxNotionalUSDT":                  systemConfig.AgentTradeMaxNotionalUSDT,
			"AgentTradeMaxTotalExposureUSDT":             systemConfig.AgentTradeMaxTotalExposureUSDT,
			"AgentTradeMaxLeverage":                      systemConfig.AgentTradeMaxLeverage,
			"AgentTradePriceFreshnessSec":                systemConfig.AgentTradePriceFreshnessSec,
			"AgentTradeMaxSlippageBps":                   systemConfig.AgentTradeMaxSlippageBps,
			"AgentTradeCooldownSec":                      systemConfig.AgentTradeCooldownSec,
			"AgentTradeProposalTTLMin":                   systemConfig.AgentTradeProposalTTLMin,

			"externalLinks": externalLinks,
		},
		"msg": "success",
	})
}

// applyServiceConfigPatch validates sensitive configuration fields before
// mutating the existing config. It must preserve fields absent from the patch.
func applyServiceConfigPatch(current models.Config, payload []byte) (models.Config, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		return current, fmt.Errorf("invalid config JSON: %w", err)
	}
	if len(fields) == 0 {
		return current, fmt.Errorf("config update cannot be empty")
	}
	if _, wrongName := fields["spotQuantEnable"]; wrongName {
		return current, fmt.Errorf("unknown spotQuantEnable field; use spot_quant_enable")
	}
	// The Spot Quant toggle must not piggyback on a patch that also changes
	// legacy Spot or Futures settings. Keep unrelated config API behavior as-is.
	if raw, ok := fields["spot_quant_enable"]; ok {
		if len(fields) != 1 {
			return current, fmt.Errorf("spot_quant_enable must be updated alone")
		}
		var enabled int
		if err := json.Unmarshal(raw, &enabled); err != nil || (enabled != 0 && enabled != 1) || string(raw) == "null" {
			return current, fmt.Errorf("spot_quant_enable must be 0 or 1")
		}
	}
	if raw, ok := fields["agent_opportunity_min_confidence"]; ok {
		var confidence float64
		if err := json.Unmarshal(raw, &confidence); err != nil {
			return current, fmt.Errorf("invalid opportunity confidence: %w", err)
		}
		// Zero is not a valid threshold: the opportunity watcher treats <=0
		// as its default 0.7, so accepting zero would mislead the UI.
		if math.IsNaN(confidence) || math.IsInf(confidence, 0) || confidence < 0.01 || confidence > 1 {
			return current, fmt.Errorf("opportunity confidence must be between 0.01 and 1")
		}
	}
	if err := json.Unmarshal(payload, &current); err != nil {
		return current, fmt.Errorf("invalid config value: %w", err)
	}
	return current, nil
}

// persistServiceConfigPatch updates only the Spot toggle column for that
// dedicated request. Existing callers editing other config fields keep their
// historical behavior; the new toggle cannot change Futures or legacy Spot.
func persistServiceConfigPatch(o orm.Ormer, current *models.Config, payload []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		return fmt.Errorf("invalid config JSON: %w", err)
	}
	if _, onlySpot := fields["spot_quant_enable"]; onlySpot {
		_, err := o.Update(current, "SpotQuantEnable")
		return err
	}
	_, err := o.Update(current)
	return err
}

func (ctrl *IndexController) EditServiceConfig() {
	systemConfig, err := utils.GetSystemConfig()
	if err != nil {
		ctrl.respondServiceConfig(utils.ResJson(500, nil, "cannot load configuration"))
		return
	}
	systemConfig, err = applyServiceConfigPatch(systemConfig, ctrl.Ctx.Input.RequestBody)
	if err != nil {
		ctrl.respondServiceConfig(utils.ResJson(400, nil, err.Error()))
		return
	}
	if err = persistServiceConfigPatch(orm.NewOrm(), &systemConfig, ctrl.Ctx.Input.RequestBody); err != nil {
		ctrl.respondServiceConfig(utils.ResJson(400, nil, "edit failed"))
		return
	}
	ctrl.respondServiceConfig(map[string]interface{}{
		"code": 200,
		"data": systemConfig,
		"msg":  "success",
	})
}

func (ctrl *IndexController) TestPusher() {
	var pusher = notify.GetNotifyChannel()
	pusher.SetModuleName("futures_test").TestPusher()
	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"msg":  "success",
	})
}

func (ctrl *IndexController) UpdateMarketCondition() {
	systemConfig, err := utils.GetSystemConfig()
	if err != nil {
		logs.Error("GetSystemConfig:", err.Error())
		ctrl.Ctx.Resp(utils.ResJson(500, nil, "internal error"))
		return
	}
	task := startMarketConditionUpdateTask(systemConfig)
	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"data": task,
		"msg":  "accepted",
	})
}

func (ctrl *IndexController) GetMarketConditionUpdateTask() {
	taskID := ctrl.Ctx.Input.Param(":taskId")
	task, exists := getMarketConditionUpdateTask(taskID)
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
