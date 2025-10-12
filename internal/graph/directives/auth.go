package directives

import (
	"context"
	"errors"
	"never-price-match-server/internal/httpctx"

	"github.com/99designs/gqlgen/graphql"
)

func Auth() func(ctx context.Context, obj interface{}, next graphql.Resolver) (res interface{}, err error) {
	return func(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
		gc := httpctx.Gin(ctx)
		if gc == nil {
			return nil, errors.New("unauthorized")
		}
		uid, ok := gc.Get("uid") // 来自 Cookie/JWT 中间件

		if !ok || uid == "" {
			return nil, errors.New("unauthorized")
		}
		return next(ctx)
	}
}
