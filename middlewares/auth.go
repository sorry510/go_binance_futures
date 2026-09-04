package middlewares

import (
	"fmt"
	"net/http"
	"strings"

	// Loads the global config before the package-level reads below run.
	_ "go_binance_futures/bootstrap"

	"github.com/beego/beego/v2/core/config"
	"github.com/beego/beego/v2/server/web/context"
	"github.com/golang-jwt/jwt/v5"
)

var webIndex, _ = config.String("web::index")
var secretKey, _ = config.String("web::secret_key")

var excludeRoutes = []string{
	"/login",
	"/ws/notifications",
	"/agents/mcp/oauth/client-metadata",
	"/agents/mcp/oauth/callback",
	"/pull",
	"/pm2-log",
	"/pm2-log2",
	"/" + webIndex,
}

// rejectUnauthorized answers a request that failed authentication.
//
// A browser navigating to a page should land back on the login screen, but an
// XHR from the already loaded single page app needs a payload it can recognise.
// Redirecting both meant an expired token made the API answer an HTML body with
// status 401, which the frontend could not tell apart from an empty result, so
// pages rendered as "no data" instead of prompting for login.
func rejectUnauthorized(ctx *context.Context) {
	if isPageRequest(ctx) {
		ctx.Redirect(http.StatusUnauthorized, "/"+webIndex+"/index.html")
		return
	}
	ctx.Output.SetStatus(http.StatusUnauthorized)
	_ = ctx.Output.JSON(map[string]interface{}{
		"code": http.StatusUnauthorized,
		"msg":  "登录已过期或未登录",
		"data": nil,
	}, false, false)
}

// isPageRequest reports whether the caller is a browser asking for a document
// rather than the app asking for data.
func isPageRequest(ctx *context.Context) bool {
	if strings.HasSuffix(ctx.Request.URL.Path, ".html") {
		return true
	}
	if ctx.Request.Header.Get("X-Requested-With") == "XMLHttpRequest" {
		return false
	}
	// Browsers ask for documents with an Accept that prefers text/html, while
	// fetch and axios default to application/json or */*.
	accept := ctx.Request.Header.Get("Accept")
	return strings.Contains(accept, "text/html")
}

func JwtMiddleware(ctx *context.Context) {
	request := ctx.Request
	path := ctx.Request.URL.Path

	// 跳过白名单
	for _, excludeRoute := range excludeRoutes {
		if match, _ := pathMatch(path, excludeRoute); match {
			return
		}
	}

	// 获取请求头中的JWT Token
	authHeader := request.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		rejectUnauthorized(ctx)
		return
	}

	claims, err := ValidateAuthorization(authHeader)
	if err != nil {
		rejectUnauthorized(ctx)
		return
	}
	ctx.Input.SetData("user", claims)
}

func ValidateAuthorization(authorization string) (jwt.MapClaims, error) {
	authorization = strings.TrimSpace(authorization)
	if !strings.HasPrefix(authorization, "Bearer ") {
		return nil, fmt.Errorf("authorization bearer token is required")
	}
	tokenString := strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer "))
	if tokenString == "" || strings.HasPrefix(tokenString, "Bearer ") {
		return nil, fmt.Errorf("invalid bearer token")
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token claims")
}

func pathMatch(actualPath, pattern string) (bool, error) {
	if strings.HasSuffix(pattern, "/*") {
		prefix := pattern[:len(pattern)-2]
		return strings.HasPrefix(actualPath, prefix), nil
	}
	return actualPath == pattern, nil
}
