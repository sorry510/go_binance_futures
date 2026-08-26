package test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	_ "go_binance_futures/routers"

	beego "github.com/beego/beego/v2/server/web"
	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	_, file, _, _ := runtime.Caller(0)
	apppath, _ := filepath.Abs(filepath.Dir(filepath.Join(file, ".."+string(filepath.Separator))))
	beego.TestBeegoInit(apppath)
	// The login controller binds the request body, which beego only retains
	// when this option is on. main.go sets the same flag at startup.
	beego.BConfig.CopyRequestBody = true
}

// TestLoginRejectsWrongCredentials is a smoke test for the router wiring, the
// login controller and the configuration reads it depends on. It deliberately
// avoids the database and the Binance API so it can run anywhere.
//
// It replaces the generated scaffold test that requested "/", a route this
// project never registers, and therefore always failed with 404.
func TestLoginRejectsWrongCredentials(t *testing.T) {
	body := strings.NewReader(`{"username":"nobody","password":"definitely-wrong"}`)
	request, err := http.NewRequest(http.MethodPost, "/login", body)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	beego.BeeApp.Handlers.ServeHTTP(recorder, request)

	Convey("Subject: Login endpoint rejects wrong credentials\n", t, func() {
		Convey("Status Code Should Be 200", func() {
			So(recorder.Code, ShouldEqual, http.StatusOK)
		})
		Convey("The Result Should Not Be Empty", func() {
			So(recorder.Body.Len(), ShouldBeGreaterThan, 0)
		})
		Convey("The Payload Should Carry The Credential Error Code", func() {
			So(recorder.Body.String(), ShouldContainSubstring, `"code":101`)
		})
	})
}

// TestUnknownRouteReturnsNotFound pins down that the router does not expose a
// catch-all handler, so a typo in a path fails loudly instead of being served.
func TestUnknownRouteReturnsNotFound(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "/no-such-route", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	recorder := httptest.NewRecorder()
	beego.BeeApp.Handlers.ServeHTTP(recorder, request)

	Convey("Subject: Unknown route\n", t, func() {
		Convey("Status Code Should Be 404", func() {
			So(recorder.Code, ShouldEqual, http.StatusNotFound)
		})
	})
}
