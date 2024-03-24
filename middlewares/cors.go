package middlewares

import (
	"github.com/beego/beego/v2/server/web/context"
)


func CorsMiddleware(ctx *context.Context) {
    request := ctx.Request
	
	// 允许 OPTIONS 方法
    if request.Method == "OPTIONS" {
        ctx.Output.Header("Access-Control-Allow-Origin", "*")
        ctx.Output.Header("Access-Control-Allow-Methods", "GET,PATCH,PUT,POST,DELETE,HEAD,OPTIONS")
        ctx.Output.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-HTTP-Method-Override")
        // ctx.Output.Header("Access-Control-Allow-Credentials", "true")
        ctx.Output.SetStatus(204)
        ctx.Output.Body(nil);
        return
    }
}
