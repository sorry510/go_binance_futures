package controllers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/models"
	"go_binance_futures/technology"
	"go_binance_futures/utils"
	"io"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

const maxStrategyTemplateImportSize int64 = 2 * 1024 * 1024

type strategyTemplateImportPayload struct {
	Name       string          `json:"name"`
	Technology json.RawMessage `json:"technology"`
	Strategy   json.RawMessage `json:"strategy"`
}

type strategyTemplateImportRule struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Code       string `json:"code"`
	FullScreen *bool  `json:"fullScreen"`
	Enable     *bool  `json:"enable"`
}

type parsedStrategyTemplateImport struct {
	Name       string
	Technology string
	Strategy   string
}

func (ctrl *StrategyTemplateController) Import() {
	data := ctrl.Ctx.Input.RequestBody
	action, stored, err := importStrategyTemplateData(data)
	if err != nil {
		ctrl.strategyTemplateImportError(err.Error())
		return
	}

	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"action":   action,
			"template": stored,
		},
		"msg": "success",
	})
}

func importStrategyTemplateData(data []byte) (string, models.StrategyTemplates, error) {
	var stored models.StrategyTemplates
	if len(bytes.TrimSpace(data)) == 0 {
		return "", stored, errors.New("请输入策略模板 JSON")
	}
	if int64(len(data)) > maxStrategyTemplateImportSize {
		return "", stored, errors.New("JSON 内容不能超过 2 MB")
	}

	template, err := parseStrategyTemplateImport(data)
	if err != nil {
		return "", stored, err
	}

	o := orm.NewOrm()
	err = o.QueryTable("strategy_templates").Filter("Name", template.Name).One(&stored)
	now := time.Now().UnixMilli()
	action := "updated"

	switch {
	case errors.Is(err, orm.ErrNoRows):
		stored = models.StrategyTemplates{
			Name:       template.Name,
			Technology: template.Technology,
			Strategy:   template.Strategy,
			CreateTime: now,
			UpdateTime: now,
		}
		stored.ID, err = o.Insert(&stored)
		action = "created"
	case err != nil:
		return "", stored, fmt.Errorf("查询同名策略模板失败: %w", err)
	default:
		stored.Technology = template.Technology
		stored.Strategy = template.Strategy
		stored.UpdateTime = now
		_, err = o.Update(&stored, "Technology", "Strategy", "UpdateTime")
	}

	if err != nil {
		return "", stored, fmt.Errorf("保存策略模板失败: %w", err)
	}
	return action, stored, nil
}

func (ctrl *StrategyTemplateController) strategyTemplateImportError(message string) {
	ctrl.Ctx.Resp(utils.ResJson(400, nil, message))
}

func parseStrategyTemplateImport(data []byte) (parsedStrategyTemplateImport, error) {
	var result parsedStrategyTemplateImport
	var payload strategyTemplateImportPayload
	if err := decodeStrictStrategyTemplateJSON(data, &payload); err != nil {
		return result, fmt.Errorf("JSON 格式错误: %s", describeStrategyTemplateJSONError(data, err))
	}

	payload.Name = strings.TrimSpace(payload.Name)
	if payload.Name == "" {
		return result, errors.New("模板名称 name 不能为空")
	}
	if len(payload.Technology) == 0 || bytes.Equal(bytes.TrimSpace(payload.Technology), []byte("null")) {
		return result, errors.New("technology 必须是技术指标配置对象")
	}
	if len(payload.Strategy) == 0 || bytes.Equal(bytes.TrimSpace(payload.Strategy), []byte("null")) {
		return result, errors.New("strategy 必须是策略规则数组")
	}

	var technologyConfig technology.TechnologyConfig
	if err := decodeStrictStrategyTemplateJSON(payload.Technology, &technologyConfig); err != nil {
		return result, fmt.Errorf("technology 格式错误: %s", describeStrategyTemplateJSONError(payload.Technology, err))
	}
	if err := line.ValidateTechnologyConfig(technologyConfig); err != nil {
		return result, fmt.Errorf("technology 配置错误: %w", err)
	}

	var strategyRules []strategyTemplateImportRule
	if err := decodeStrictStrategyTemplateJSON(payload.Strategy, &strategyRules); err != nil {
		return result, fmt.Errorf("strategy 格式错误: %s", describeStrategyTemplateJSONError(payload.Strategy, err))
	}
	if err := validateStrategyTemplateImportRules(strategyRules); err != nil {
		return result, err
	}
	if err := validateStrategyTemplateRuleExpressions(technologyConfig, strategyRules); err != nil {
		return result, err
	}

	technologyJSON, err := compactStrategyTemplateJSON(payload.Technology)
	if err != nil {
		return result, fmt.Errorf("technology 格式错误: %w", err)
	}
	strategyJSON, err := compactStrategyTemplateJSON(payload.Strategy)
	if err != nil {
		return result, fmt.Errorf("strategy 格式错误: %w", err)
	}

	result.Name = payload.Name
	result.Technology = technologyJSON
	result.Strategy = strategyJSON
	return result, nil
}

func validateStrategyTemplateImportRules(rules []strategyTemplateImportRule) error {
	supportedTypes := map[string]struct{}{
		"long": {}, "short": {}, "close_long": {}, "close_short": {},
	}
	usedNames := make(map[string]struct{})

	for index, rule := range rules {
		position := index + 1
		name := strings.TrimSpace(rule.Name)
		if name == "" {
			return fmt.Errorf("strategy 第 %d 项的 name 不能为空", position)
		}
		if _, exists := usedNames[name]; exists {
			return fmt.Errorf("strategy 第 %d 项的 name %q 重复", position, name)
		}
		usedNames[name] = struct{}{}
		if _, supported := supportedTypes[rule.Type]; !supported {
			return fmt.Errorf("strategy 第 %d 项的 type %q 无效，仅支持 long、short、close_long、close_short", position, rule.Type)
		}
		if strings.TrimSpace(rule.Code) == "" {
			return fmt.Errorf("strategy 第 %d 项的 code 不能为空", position)
		}
		if rule.FullScreen == nil {
			return fmt.Errorf("strategy 第 %d 项缺少布尔字段 fullScreen", position)
		}
		if rule.Enable == nil {
			return fmt.Errorf("strategy 第 %d 项缺少布尔字段 enable", position)
		}
	}

	return nil
}

func decodeStrictStrategyTemplateJSON(data []byte, target interface{}) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("只能包含一个 JSON 根值")
		}
		return err
	}
	return nil
}

func compactStrategyTemplateJSON(data []byte) (string, error) {
	var buffer bytes.Buffer
	if err := json.Compact(&buffer, data); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func describeStrategyTemplateJSONError(data []byte, err error) string {
	var syntaxError *json.SyntaxError
	if errors.As(err, &syntaxError) {
		lineNumber, columnNumber := strategyTemplateJSONPosition(data, syntaxError.Offset)
		return fmt.Sprintf("第 %d 行第 %d 列: %s", lineNumber, columnNumber, syntaxError.Error())
	}
	var typeError *json.UnmarshalTypeError
	if errors.As(err, &typeError) {
		lineNumber, columnNumber := strategyTemplateJSONPosition(data, typeError.Offset)
		return fmt.Sprintf("第 %d 行第 %d 列: 字段 %s 的类型错误，%s", lineNumber, columnNumber, typeError.Field, typeError.Error())
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return "JSON 内容不完整"
	}
	return err.Error()
}

func strategyTemplateJSONPosition(data []byte, offset int64) (int, int) {
	lineNumber, columnNumber := 1, 1
	limit := int(offset) - 1
	if limit > len(data) {
		limit = len(data)
	}
	for index := 0; index < limit; index++ {
		if data[index] == '\n' {
			lineNumber++
			columnNumber = 1
			continue
		}
		columnNumber++
	}
	return lineNumber, columnNumber
}
