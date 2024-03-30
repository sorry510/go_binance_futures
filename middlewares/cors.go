package middlewares

import (
	"net/http"

	"github.com/beego/beego/v2/server/web/context"
)


func CorsMiddleware(ctx *context.Context) {
    request := ctx.Request
    ctx.ResponseWriter.Header().Set("Access-Control-Allow-Origin", "*")
    ctx.ResponseWriter.Header().Set("Access-Control-Allow-Methods", "*")
    ctx.ResponseWriter.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-HTTP-Method-Override")
    ctx.ResponseWriter.Header().Set("Access-Control-Max-Age", "3600")
    
	// // 允许 OPTIONS 方法
    if request.Method == http.MethodOptions {
        ctx.ResponseWriter.WriteHeader(http.StatusNoContent)
        return
    }
}
