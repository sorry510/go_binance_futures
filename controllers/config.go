package controllers

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"go_binance_futures/service/apprestart"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/core/config"
	"github.com/beego/beego/v2/server/web"
)

const applicationConfigFile = "./conf/app.conf"
const maxConfigBytes = 256 << 10

var configWriteMutex sync.Mutex
var bootConfigHash, bootConfigPresent = initialConfigFingerprint()

func initialConfigFingerprint() ([32]byte, bool) {
	data, err := os.ReadFile(applicationConfigFile)
	if err != nil {
		return [32]byte{}, false
	}
	return sha256.Sum256(data), true
}

func restartRequired(data []byte) bool {
	return bootConfigPresent && sha256.Sum256(data) != bootConfigHash
}

type ConfigController struct {
	web.Controller
}

func (ctrl *ConfigController) Get() {
	content, err := os.ReadFile(applicationConfigFile)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "读取配置失败"))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"content":          string(content),
			"restart_required": restartRequired(content),
		},
		"msg": "success",
	})
}

func (ctrl *ConfigController) Edit() {
	var content ConfigContent
	if err := ctrl.BindJSON(&content); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "配置请求无效"))
		return
	}
	changed, err := saveIfChanged(applicationConfigFile, []byte(content.Code))
	if err != nil {
		// Never return file contents or secrets to the browser.
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "保存失败，请检查配置格式和文件权限"))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"changed":          changed,
			"restart_required": configRestartRequired(),
		},
		"msg": "success",
	})
}

// Restart is deliberately separate from saving: configuration changes are
// never allowed to restart a running trading process without a second click.
func (ctrl *ConfigController) Restart() {
	// No saved configuration changes means no process restart is needed.
	if !configRestartRequired() {
		ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "msg": "no changes to apply", "data": map[string]bool{"restarted": false}})
		return
	}
	if err := apprestart.Schedule(); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "无法启动重启，请检查程序运行环境"))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"msg":  "restart requested",
		"data": map[string]bool{"requested": true},
	})
}

func configRestartRequired() bool {
	content, err := os.ReadFile(applicationConfigFile)
	if err != nil {
		return false
	}
	return restartRequired(content)
}

// saveIfChanged does nothing when the bytes are unchanged. Changed configs
// are parsed before writing, written privately, and atomically renamed.
func saveIfChanged(path string, data []byte) (bool, error) {
	configWriteMutex.Lock()
	defer configWriteMutex.Unlock()
	previous, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	if bytes.Equal(previous, data) {
		return false, nil
	}
	if len(data) == 0 || len(data) > maxConfigBytes {
		return false, fmt.Errorf("invalid configuration length")
	}
	parsed, err := config.NewConfigData("ini", data)
	if err != nil {
		return false, err
	}
	appname, err := parsed.String("appname")
	if err != nil || appname == "" {
		return false, fmt.Errorf("missing application name")
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".app.conf-*.tmp")
	if err != nil {
		return false, err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return false, err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return false, err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return false, err
	}
	if err := tmp.Close(); err != nil {
		return false, err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return false, err
	}
	return true, nil
}

type ConfigContent struct {
	Code string `json:"code"`
}
