package auth

import (
	"never-price-match-server/internal/infra/logger"

	"github.com/gin-gonic/gin"
)

func CookieAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie("sid")
		if err == nil && cookie != "" {
			claims, err := Parse(cookie)
			if err == nil {
				uid := claims.UserID
				c.Set("uid", uid) // Available for subsequent use
			} else {
				logger.L.Warn("parse token failed", logger.Err(err))
			}
		}
		c.Next()
	}
}
