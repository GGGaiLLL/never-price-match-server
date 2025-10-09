package middleware

import (
	"never-price-match-server/pkg/logger"
	"time"

	"github.com/gin-gonic/gin"
)

func ZapLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)
		logger.L.Info("request",
			logger.Str("method", c.Request.Method),
			logger.Str("path", c.FullPath()),
			logger.Int("status", c.Writer.Status()),
			logger.Str("ip", c.ClientIP()),
			logger.Dur("latency", latency),
			logger.Str("ua", c.Request.UserAgent()),
		)
	}
}


