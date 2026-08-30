package controllers

import (
	"net/http/httptest"
	"testing"

	beegocontext "github.com/beego/beego/v2/server/web/context"
)

func TestNotificationWebSocketDisablesBeegoRenderingBeforeUpgrade(t *testing.T) {
	request := httptest.NewRequest("GET", "/ws/notifications?token=invalid", nil)
	response := httptest.NewRecorder()
	ctx := beegocontext.NewContext()
	ctx.Reset(response, request)

	ctrl := &NotificationWebSocketController{}
	ctrl.Init(ctx, "NotificationWebSocketController", "Get", nil)
	if !ctrl.EnableRender {
		t.Fatal("expected Beego rendering to start enabled")
	}

	ctrl.Get()

	if ctrl.EnableRender {
		t.Fatal("WebSocket controller must disable Beego rendering before upgrade")
	}
}
