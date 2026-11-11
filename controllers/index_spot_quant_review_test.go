package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	beegocontext "github.com/beego/beego/v2/server/web/context"
	_ "github.com/mattn/go-sqlite3"
)

// Regression for 06-F3 / 06-F7: a dedicated Spot Quant toggle must persist
// without rewriting unrelated Futures or legacy Spot configuration columns.
func TestSpotQuantConfigUpdateOnlyChangesItsColumn(t *testing.T) {
	if err := orm.RegisterDriver("sqlite3", orm.DRSqlite); err != nil {
		t.Fatal(err)
	}
	if _, err := orm.GetDB("default"); err != nil {
		if err := orm.RegisterDataBase("default", "sqlite3", filepath.Join(t.TempDir(), "default.db")); err != nil {
			t.Fatal(err)
		}
	}
	const alias = "controller_spot_quant_patch_test"
	if err := orm.RegisterDataBase(alias, "sqlite3", filepath.Join(t.TempDir(), "spot-config.db")); err != nil {
		t.Fatal(err)
	}
	orm.RegisterModel(new(models.Config))
	o := orm.NewOrmUsingDB(alias)
	if _, err := o.Raw(`CREATE TABLE config (
		id BIGINT PRIMARY KEY,
		spot_quant_enable INTEGER NOT NULL DEFAULT 0,
		future_enable INTEGER NOT NULL DEFAULT 0,
		spot_enable INTEGER NOT NULL DEFAULT 0,
		spot_new_enable INTEGER NOT NULL DEFAULT 0
	)`).Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Raw(
		"INSERT INTO config(id,spot_quant_enable,future_enable,spot_enable,spot_new_enable) VALUES (1,0,1,1,1)",
	).Exec(); err != nil {
		t.Fatal(err)
	}
	// Deliberately stale concurrent values: the toggle must not overwrite
	// Futures/Spot data even if the controller loaded an older snapshot.
	old := models.Config{ID: 1, FutureEnable: 0, SpotEnable: 0, SpotNewEnable: 0}
	body := []byte(`{"spot_quant_enable":1}`)
	patched, err := applyServiceConfigPatch(old, body)
	if err != nil {
		t.Fatal(err)
	}
	if err := persistServiceConfigPatch(o, &patched, body); err != nil {
		t.Fatal(err)
	}
	var gotToggle, gotFuture, gotSpot, gotSpotNew int
	if err := o.Raw("SELECT spot_quant_enable,future_enable,spot_enable,spot_new_enable FROM config WHERE id=1").
		QueryRow(&gotToggle, &gotFuture, &gotSpot, &gotSpotNew); err != nil {
		t.Fatal(err)
	}
	if gotToggle != 1 || gotFuture != 1 || gotSpot != 1 || gotSpotNew != 1 {
		t.Fatalf("toggle update changed unrelated fields: toggle=%d futures=%d spot=%d spot_new=%d",
			gotToggle, gotFuture, gotSpot, gotSpotNew)
	}
}

// Regression for 06-F2/06-F4: configuration APIs must expose the same
// snake_case key on read/write, and JSON must not depend on Accept negotiation.
func TestSpotQuantServiceConfigJSONIndependentOfAccept(t *testing.T) {
	for _, accept := range []string{"application/json", "application/xml", "application/x-protobuf", "text/plain"} {
		t.Run(accept, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/service/config", nil)
			req.Header.Set("Accept", accept)
			rec := httptest.NewRecorder()
			ctx := beegocontext.NewContext()
			ctx.Reset(rec, req)
			ctrl := new(IndexController)
			ctrl.Init(ctx, "IndexController", "", nil)
			ctrl.respondServiceConfig(map[string]interface{}{
				"code": 200,
				"data": map[string]interface{}{"spot_quant_enable": 1},
				"msg":  "success",
			})
			if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
				t.Fatalf("Accept %q response content type=%q", accept, got)
			}
			var response struct {
				Data map[string]json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatalf("Accept %q invalid JSON %q: %v", accept, rec.Body.String(), err)
			}
			value, ok := response.Data["spot_quant_enable"]
			if !ok || string(value) != "1" {
				t.Fatalf("missing snake_case Spot key from JSON: %s", rec.Body.String())
			}
			if _, err := applyServiceConfigPatch(models.Config{}, []byte(`{"spot_quant_enable":1}`)); err != nil {
				t.Fatalf("GET key not accepted by PUT patch: %v", err)
			}
		})
	}
}
