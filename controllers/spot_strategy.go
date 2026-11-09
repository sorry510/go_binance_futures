package controllers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"go_binance_futures/service/spotstrategy"

	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	"github.com/golang-jwt/jwt/v5"
)

// S1: configuration only. No route sends or cancels a Binance order.
type SpotStrategyController struct{ web.Controller }

// Never use Ctx.Resp: it negotiates YAML/XML/protobuf based on Accept and can
// panic on map values. Always return the same JSON with a real HTTP status.
func (c *SpotStrategyController) respond(status int, data any, msg string) {
	c.Ctx.Output.SetStatus(status)
	_ = c.JSONResp(map[string]any{"code": status, "data": data, "msg": msg})
}

func (c *SpotStrategyController) authorized() (string, bool) {
	claims, ok := c.Ctx.Input.GetData("user").(jwt.MapClaims)
	if !ok {
		c.respond(http.StatusUnauthorized, nil, "spot_strategy_auth_required")
		return "", false
	}
	user, ok := claims["Username"].(string)
	if !ok || strings.TrimSpace(user) == "" {
		c.respond(http.StatusUnauthorized, nil, "spot_strategy_auth_required")
		return "", false
	}
	return strings.TrimSpace(user), true
}

func (c *SpotStrategyController) checkJSONRequest() bool {
	contentType := strings.ToLower(strings.TrimSpace(c.Ctx.Request.Header.Get("Content-Type")))
	if contentType != "application/json" && !strings.HasPrefix(contentType, "application/json;") {
		c.respond(http.StatusUnsupportedMediaType, nil, "spot_strategy_json_required")
		return false
	}
	return true
}

func (c *SpotStrategyController) Get() {
	if _, ok := c.authorized(); !ok {
		return
	}
	if err := spotstrategy.ValidateScope(c.GetString("account_id"), c.GetString("owner")); err != nil {
		c.respond(http.StatusBadRequest, nil, err.Error())
		return
	}
	config, err := spotstrategy.LoadSetting(c.GetString("symbol"))
	if err != nil {
		c.respond(http.StatusBadRequest, nil, err.Error())
		return
	}
	c.respond(http.StatusOK, config, "success")
}

func decodeSpotStrategyPayload(payload []byte, out interface{}) error {
	if len(payload) == 0 || len(payload) > 32768 {
		return errors.New("invalid or oversized Spot strategy payload")
	}
	d := json.NewDecoder(bytes.NewReader(payload))
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		return err
	}
	var tail interface{}
	if err := d.Decode(&tail); err != io.EOF {
		return errors.New("unexpected trailing JSON payload")
	}
	return nil
}

func (c *SpotStrategyController) Put() {
	if _, ok := c.authorized(); !ok {
		return
	}
	if !c.checkJSONRequest() {
		return
	}
	var req spotstrategy.Patch
	if err := decodeSpotStrategyPayload(c.Ctx.Input.RequestBody, &req); err != nil {
		c.respond(http.StatusBadRequest, nil, err.Error())
		return
	}
	result, err := spotstrategy.UpdateSetting(req)
	if err != nil {
		c.respond(http.StatusBadRequest, nil, err.Error())
		return
	}
	c.respond(http.StatusOK, result, "success")
}

type spotModePatch struct {
	AccountID string `json:"account_id"`
	Owner     string `json:"owner"`
	Mode      string `json:"mode"`
}

func (c *SpotStrategyController) GetMode() {
	if _, ok := c.authorized(); !ok {
		return
	}
	if err := spotstrategy.ValidateScope(c.GetString("account_id"), c.GetString("owner")); err != nil {
		c.respond(http.StatusBadRequest, nil, err.Error())
		return
	}
	mode, err := spotstrategy.CurrentMode()
	if err != nil {
		c.respond(http.StatusServiceUnavailable, nil, "spot_mode_unavailable")
		return
	}
	c.respond(http.StatusOK, map[string]any{"mode": mode, "execution": "mock_only"}, "success")
}

func (c *SpotStrategyController) PutMode() {
	operator, ok := c.authorized()
	if !ok {
		return
	}
	if !c.checkJSONRequest() {
		return
	}
	var req spotModePatch
	if err := decodeSpotStrategyPayload(c.Ctx.Input.RequestBody, &req); err != nil {
		c.respond(http.StatusBadRequest, nil, err.Error())
		return
	}
	if err := spotstrategy.ValidateScope(req.AccountID, req.Owner); err != nil {
		c.respond(http.StatusBadRequest, nil, err.Error())
		return
	}
	old, readErr := spotstrategy.CurrentMode()
	if readErr != nil {
		c.respond(http.StatusServiceUnavailable, nil, "spot_mode_unavailable")
		return
	}
	err := spotstrategy.RequestMode(req.Mode)
	result := "unchanged"
	if err != nil {
		result = "blocked"
	}
	// S1 cannot change modes. Record the attempted action with the validated
	// operator, old/new mode and time, without logging credentials or payloads.
	// A durable transition audit belongs to S2 when switching is implemented.
	logs.Info("spot_mode_audit operator=%q old=%q requested=%q result=%s time_ms=%d",
		operator, old, req.Mode, result, time.Now().UnixMilli())
	if err != nil {
		c.respond(http.StatusConflict, nil, err.Error())
		return
	}
	c.respond(http.StatusOK, map[string]any{"mode": old, "execution": "mock_only"}, "success")
}
