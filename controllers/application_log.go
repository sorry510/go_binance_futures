package controllers

import (
	"strings"

	"go_binance_futures/service/applog"

	"github.com/beego/beego/v2/server/web"
)

// ApplicationLogController exposes only a bounded read of the current
// application's fixed log file, protected by the existing JWT middleware.
type ApplicationLogController struct {
	web.Controller
}

func (ctrl *ApplicationLogController) Get() {
	limit, err := ctrl.GetInt("limit", 200)
	if err != nil || limit < 1 || limit > 1000 {
		_ = ctrl.JSONResp(map[string]interface{}{"code": 400, "msg": "limit must be between 1 and 1000"})
		return
	}
	level := strings.ToUpper(strings.TrimSpace(ctrl.GetString("level")))
	switch level {
	case "", "ALL", "ERROR", "WARN", "INFO", "DEBUG":
	default:
		_ = ctrl.JSONResp(map[string]interface{}{"code": 400, "msg": "invalid log level"})
		return
	}
	keyword := strings.TrimSpace(ctrl.GetString("keyword"))
	if len(keyword) > 128 {
		_ = ctrl.JSONResp(map[string]interface{}{"code": 400, "msg": "keyword is too long"})
		return
	}
	result, err := applog.Recent(limit, level, keyword)
	if err != nil {
		// Do not expose local paths or OS details in the HTTP response.
		_ = ctrl.JSONResp(map[string]interface{}{"code": 500, "msg": "application log unavailable"})
		return
	}
	_ = ctrl.JSONResp(map[string]interface{}{
		"code": 200, "msg": "success",
		"data": map[string]interface{}{
			"lines":         result.Lines,
			"scanned_bytes": result.ScannedBytes,
			"has_earlier":   result.HasEarlier,
		},
	})
}
