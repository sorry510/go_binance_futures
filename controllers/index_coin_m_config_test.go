package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	beegocontext "github.com/beego/beego/v2/server/web/context"
	_ "github.com/mattn/go-sqlite3"
)

var controllerConfigModelOnce sync.Once

func registerControllerConfigModel() {
	controllerConfigModelOnce.Do(func() { orm.RegisterModel(new(models.Config)) })
}

func TestCM1GlobalTogglePatchValidationAndIsolation(t *testing.T) {
	base := models.Config{ID: 1, CoinMQuantEnable: 0, DeliveryEnable: 1,
		FutureEnable: 1, SpotEnable: 1, SpotQuantEnable: 1}
	for _, tt := range []struct {
		body      string
		want      int8
		shouldErr bool
	}{
		{body: "{\"coin_m_quant_enable\":1}", want: 1},
		{body: "{\"coin_m_quant_enable\":0}", want: 0},
		{body: "{\"coin_m_quant_enable\":2}", shouldErr: true},
		{body: "{\"coin_m_quant_enable\":-1}", shouldErr: true},
		{body: "{\"coin_m_quant_enable\":\"1\"}", shouldErr: true},
		{body: "{\"coin_m_quant_enable\":true}", shouldErr: true},
		{body: "{\"coin_m_quant_enable\":null}", shouldErr: true},
		{body: "{\"coinMQuantEnable\":1}", shouldErr: true},
		{body: "{\"coin_m_quant_enable\":1,\"delivery_enable\":0}", shouldErr: true},
		{body: "{\"coin_m_quant_enable\":1,\"future_enable\":0}", shouldErr: true},
		{body: "{\"coin_m_quant_enable\":1,\"spot_quant_enable\":0}", shouldErr: true},
	} {
		got, err := applyServiceConfigPatch(base, []byte(tt.body))
		if tt.shouldErr {
			if err == nil || !(strings.Contains(err.Error(), "coin_m_quant_enable") || strings.Contains(err.Error(), "coinMQuantEnable")) {
				t.Errorf("%s: expected isolated CM toggle validation error, got %v", tt.body, err)
			}
			continue
		}
		if err != nil || got.CoinMQuantEnable != tt.want {
			t.Errorf("%s: CM toggle=%d, err=%v", tt.body, got.CoinMQuantEnable, err)
		}
		if got.DeliveryEnable != 1 || got.FutureEnable != 1 || got.SpotEnable != 1 || got.SpotQuantEnable != 1 {
			t.Errorf("%s: patch changed unrelated trading switches: %+v", tt.body, got)
		}
	}
	got, err := applyServiceConfigPatch(base, []byte("{\"delivery_enable\":0}"))
	if err != nil || got.CoinMQuantEnable != 0 {
		t.Fatalf("legacy delivery_enable unexpectedly changed new CM toggle: %+v %v", got, err)
	}
}

func TestCM1GlobalTogglePersistOnlyOneColumn(t *testing.T) {
	_ = orm.RegisterDriver("sqlite3", orm.DRSqlite)
	// Default ORM alias is shared by existing controller tests and must never
	// point at an actual app.conf database.
	if _, err := orm.GetDB("default"); err != nil {
		if err := orm.RegisterDataBase("default", "sqlite3",
			"file:controllers_stable_default_test?mode=memory&cache=shared"); err != nil {
			t.Fatal(err)
		}
	}
	const alias = "controller_cm1_config_patch"
	if err := orm.RegisterDataBase(alias, "sqlite3", filepath.Join(t.TempDir(), "cm1.db")); err != nil {
		t.Fatal(err)
	}
	registerControllerConfigModel()
	o := orm.NewOrmUsingDB(alias)
	if _, err := o.Raw("CREATE TABLE config (id BIGINT PRIMARY KEY, coin_m_quant_enable INTEGER NOT NULL DEFAULT 0, " +
		"delivery_enable INTEGER NOT NULL DEFAULT 0, future_enable INTEGER NOT NULL DEFAULT 0, " +
		"spot_quant_enable INTEGER NOT NULL DEFAULT 0, spot_enable INTEGER NOT NULL DEFAULT 0)").Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Raw("INSERT INTO config(id,coin_m_quant_enable,delivery_enable,future_enable,spot_quant_enable,spot_enable) VALUES(1,0,1,1,1,1)").Exec(); err != nil {
		t.Fatal(err)
	}
	// Simulate a stale config object: updating the whole struct would clear
	// previously enabled, unrelated trading modules.
	stale := models.Config{ID: 1}
	patch := []byte("{\"coin_m_quant_enable\":1}")
	value, err := applyServiceConfigPatch(stale, patch)
	if err != nil {
		t.Fatal(err)
	}
	if err := persistServiceConfigPatch(o, &value, patch); err != nil {
		t.Fatal(err)
	}
	var cm, delivery, futures, spotQuant, spot int
	if err := o.Raw("SELECT coin_m_quant_enable,delivery_enable,future_enable,spot_quant_enable,spot_enable FROM config WHERE id=1").
		QueryRow(&cm, &delivery, &futures, &spotQuant, &spot); err != nil {
		t.Fatal(err)
	}
	if cm != 1 || delivery != 1 || futures != 1 || spotQuant != 1 || spot != 1 {
		t.Fatalf("CM toggle overwrote legacy configs: CM=%d delivery=%d UM=%d spotQuant=%d spot=%d",
			cm, delivery, futures, spotQuant, spot)
	}
	patch = []byte("{\"coin_m_quant_enable\":0}")
	value, err = applyServiceConfigPatch(value, patch)
	if err != nil || persistServiceConfigPatch(o, &value, patch) != nil {
		t.Fatal("could not persist OFF toggle", err)
	}
	if err := o.Raw("SELECT coin_m_quant_enable FROM config WHERE id=1").QueryRow(&cm); err != nil || cm != 0 {
		t.Fatalf("new CM switch did not turn OFF: %d %v", cm, err)
	}
}

func TestCM1ConfigFieldJSONNameAndContentType(t *testing.T) {
	v := models.Config{CoinMQuantEnable: 1, DeliveryEnable: 0}
	b, err := json.Marshal(&v)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(b, &fields); err != nil || fields["coin_m_quant_enable"] != float64(1) {
		t.Fatalf("new CM JSON config key missing: %s (%v)", b, err)
	}
	for _, accept := range []string{"application/json", "application/xml", "text/plain"} {
		req := httptest.NewRequest(http.MethodGet, "/service/config", nil)
		req.Header.Set("Accept", accept)
		rec := httptest.NewRecorder()
		ctx := beegocontext.NewContext()
		ctx.Reset(rec, req)
		ctrl := new(IndexController)
		ctrl.Init(ctx, "IndexController", "", nil)
		ctrl.respondServiceConfig(map[string]interface{}{
			"code": 200, "data": map[string]interface{}{"coin_m_quant_enable": 1}, "msg": "success",
		})
		if !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
			t.Fatalf("CM config Accept=%s returned non-JSON response", accept)
		}
		var response struct {
			Data map[string]json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil ||
			string(response.Data["coin_m_quant_enable"]) != "1" {
			t.Fatalf("CM config response incorrect Accept=%s body=%s error=%v", accept, rec.Body.String(), err)
		}
	}
}
