package router

import (
	"never-price-match-server/internal/api"
	"never-price-match-server/internal/middleware"

	"github.com/gin-gonic/gin"
)

func Setup() *gin.Engine {
    r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.ZapLogger())

	r.POST("/users", api.CreateUser)
	r.GET("/users", api.ListUsers)

	return r
}
