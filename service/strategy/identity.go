package strategy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
)

const (
	SystemCloseStrategyName = "system_roi_10pct_fallback"
	SystemCloseStrategyType = "system"
	SystemCloseStrategyCode = "ROI > 10 || ROI < -10"
)

type RuleIdentity struct {
	Name string
	Type string
	Hash string
}

type TemplateIdentity struct {
	ID   int64
	Name string
}

type strategyRuleJSON struct {
	Name   string `json:"name"`
	Enable bool   `json:"enable"`
	Code   string `json:"code"`
	Type   string `json:"type"`
}

func RuleHash(code string) string {
	value := strings.TrimSpace(code)
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func StrategySnapshotHash(technologyJSON, strategyJSON string) string {
	technology := strings.TrimSpace(technologyJSON)
	strategy := strings.TrimSpace(strategyJSON)
	if technology == "" && strategy == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(technology + "\n" + strategy))
	return hex.EncodeToString(sum[:])
}

func ResolveRuleIdentity(strategyJSON, code, preferredType string) RuleIdentity {
	code = strings.TrimSpace(code)
	preferredType = strings.TrimSpace(preferredType)
	identity := RuleIdentity{Type: preferredType, Hash: RuleHash(code)}
	if code == "" {
		return identity
	}
	var rules []strategyRuleJSON
	if err := json.Unmarshal([]byte(strategyJSON), &rules); err != nil {
		return identity
	}
	var fallback *strategyRuleJSON
	for index := range rules {
		rule := &rules[index]
		if strings.TrimSpace(rule.Code) != code {
			continue
		}
		if fallback == nil {
			fallback = rule
		}
		if preferredType != "" && strings.EqualFold(strings.TrimSpace(rule.Type), preferredType) {
			fallback = rule
			break
		}
	}
	if fallback == nil {
		return identity
	}
	identity.Name = truncateStrategyIdentity(fallback.Name)
	identity.Type = strings.TrimSpace(fallback.Type)
	return identity
}

func ResolveTemplateIdentity(o orm.Ormer, templateID int64, templateName, technologyJSON, strategyJSON string) (TemplateIdentity, error) {
	identity := TemplateIdentity{ID: templateID, Name: truncateStrategyIdentity(templateName)}
	if templateID > 0 {
		if identity.Name != "" {
			return identity, nil
		}
		var row models.StrategyTemplates
		if err := o.QueryTable(new(models.StrategyTemplates)).Filter("id", templateID).One(&row); err == nil {
			identity.Name = truncateStrategyIdentity(row.Name)
			return identity, nil
		} else if err != orm.ErrNoRows {
			return TemplateIdentity{}, err
		}
	}
	if strings.TrimSpace(strategyJSON) == "" {
		return identity, nil
	}
	var rows []models.StrategyTemplates
	query := o.QueryTable(new(models.StrategyTemplates)).Filter("strategy", strategyJSON)
	if strings.TrimSpace(technologyJSON) != "" {
		query = query.Filter("technology", technologyJSON)
	}
	if _, err := query.OrderBy("id").Limit(2).All(&rows); err != nil {
		return TemplateIdentity{}, err
	}
	if len(rows) == 0 {
		return identity, nil
	}
	return TemplateIdentity{ID: rows[0].ID, Name: truncateStrategyIdentity(rows[0].Name)}, nil
}

func truncateStrategyIdentity(value string) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) > 128 {
		runes = runes[:128]
	}
	return string(runes)
}
