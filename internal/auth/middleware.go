package auth

import (
	"never-price-match-server/internal/infra/logger"

	"github.com/gin-gonic/gin"
)

func CookieAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if t, err := c.Cookie("sid"); err == nil && t != "" {
			if uid, err := Parse(t); err == nil {
				c.Set("uid", uid) // 后续可用
			} else {
				logger.L.Info("invalid token", logger.Err(err))
			}
		}
		c.Next()
	}
}
