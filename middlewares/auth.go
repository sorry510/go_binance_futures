package middlewares

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/beego/beego/v2/core/config"
	"github.com/beego/beego/v2/server/web/context"
	"github.com/golang-jwt/jwt/v5"
)

var webIndex, _ = config.String("web::index")
var secretKey, _ = config.String("web::secret_key")

var excludeRoutes = []string{
	"/login",
	"/pull",
	"/pm2-log",
	"/pm2-log2",
	"/" + webIndex,
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
		ctx.Redirect(http.StatusUnauthorized, "/" + webIndex + "/index.html")
		return
    }

    tokenString := authHeader[len("Bearer "):]

    // 解析并验证Token
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
        }
        return []byte(secretKey), nil
    })
	
	if err != nil {
        ctx.Redirect(http.StatusUnauthorized, "/" + webIndex + "/index.html")
        return
    }

    // 验证通过，将claims放入上下文供后续处理器使用
    if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
        ctx.Input.SetData("user", claims)
    } else {
        ctx.Redirect(http.StatusUnauthorized, "/" + webIndex + "/index.html")
		return
    }
}

func pathMatch(actualPath, pattern string) (bool, error) {
    if strings.HasSuffix(pattern, "/*") {
        prefix := pattern[:len(pattern)-2]
        return strings.HasPrefix(actualPath, prefix), nil
    }
    return actualPath == pattern, nil
}