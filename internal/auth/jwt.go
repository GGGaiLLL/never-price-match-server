package auth

import (
	"github.com/golang-jwt/jwt/v5"
	"time"
)

var hmacSecret = []byte("change-me") // 可从 viper 读取

func SetSecret(s string) {
	if s != "" {
		hmacSecret = []byte(s)
	}
}

func Sign(userID string, ttl time.Duration) (string, error) {
	claims := jwt.MapClaims{"sub": userID, "exp": time.Now().Add(ttl).Unix()}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(hmacSecret)
}

func Parse(tokenStr string) (string, error) {
	t, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) { return hmacSecret, nil })
	if err != nil || !t.Valid {
		return "", err
	}
	if c, ok := t.Claims.(jwt.MapClaims); ok {
		if sub, _ := c["sub"].(string); sub != "" {
			return sub, nil
		}
	}
	return "", jwt.ErrTokenInvalidClaims
}
