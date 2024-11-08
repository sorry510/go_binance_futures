package utils

import (
	"strconv"
	"time"

	"github.com/beego/beego/v2/core/config"
	"github.com/golang-jwt/jwt/v5"
)

// secretKey 是用于签署JWT的密钥
var secretKey, _ = config.String("web::secret_key")
var key = []byte(secretKey)

func GenerateToken(username string, expires int) (string, error) {
  var t = jwt.NewWithClaims(jwt.SigningMethodHS256, 
    jwt.MapClaims{ 
      "Username": username,
      "jti": strconv.FormatInt(time.Now().UnixNano(), 10),
      "iat": time.Now().Unix(),
      "exp": time.Now().Add(time.Hour * time.Duration(expires)).Unix(), // 设置JWT的有效期为30天
    })
    signedToken, err := t.SignedString(key)
    if err != nil {
      return "", err
    }
    return signedToken, nil
}

