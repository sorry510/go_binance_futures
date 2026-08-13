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
	"/ws/notifications",
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

	claims, err := ValidateAuthorization(authHeader)
	if err != nil {
        ctx.Redirect(http.StatusUnauthorized, "/" + webIndex + "/index.html")
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
