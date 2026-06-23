package common

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenFactoryJWTSecret 返回与 TokenFactory 共享的 JWT 签名密钥。
// 优先读取环境变量 TOKENFACTORY_JWT_SECRET，若为空则回退到 SESSION_SECRET（与现有会话一致）。
func TokenFactoryJWTSecret() []byte {
	secret := os.Getenv("TOKENFACTORY_JWT_SECRET")
	if secret == "" {
		secret = SessionSecret
	}
	return []byte(secret)
}

// TokenFactoryJWTExpireHours JWT 有效期（小时），默认 24。
func TokenFactoryJWTExpireHours() int {
	h := os.Getenv("TOKENFACTORY_JWT_HOURS")
	if h == "" {
		return 24
	}
	n := 0
	for _, c := range h {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			break
		}
	}
	if n <= 0 {
		return 24
	}
	return n
}

// IssueTokenFactoryJWT 为用户签发 JWT，供 TokenFactory 侧验证。
// claims 包含 uid (用户ID) 和 role (角色值)，与 TokenFactory 的 JWT 格式一致。
func IssueTokenFactoryJWT(userID int, role int) (string, error) {
	claims := jwt.MapClaims{
		"uid":  userID,
		"role": role,
		"exp":  time.Now().Add(time.Duration(TokenFactoryJWTExpireHours()) * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(TokenFactoryJWTSecret())
}
