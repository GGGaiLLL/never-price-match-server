package httpctx

import (
	"context"
	"github.com/gin-gonic/gin"
)

type ctxKey struct{}

var key ctxKey

func WithGin(ctx context.Context, c *gin.Context) context.Context {
	return context.WithValue(ctx, key, c)
}
func Gin(ctx context.Context) *gin.Context {
	if v := ctx.Value(key); v != nil {
		if c, ok := v.(*gin.Context); ok {
			return c
		}
	}
	return nil
}
