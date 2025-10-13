package graph

import (
	"never-price-match-server/internal/product"
	"never-price-match-server/internal/user"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	UserService    user.Service
	ProductService product.Service
}
