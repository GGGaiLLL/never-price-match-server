package app

import (
	"net/http"
	"os"

	"never-price-match-server/internal/graph"
	"never-price-match-server/internal/graph/generated"
	"never-price-match-server/internal/infra/db"
	"never-price-match-server/internal/infra/logger"
	"never-price-match-server/internal/infra/repo"
	"never-price-match-server/internal/user"

	"github.com/gin-contrib/cors"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func initViper() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	_ = viper.ReadInConfig()

	viper.AutomaticEnv()
	_ = viper.BindEnv("APP_ENV")

	env := viper.GetString("APP_ENV")
	if env == "" {
		env = viper.GetString("env")
	}
	if env != "" {
		viper.SetConfigName("config." + env)
		_ = viper.MergeInConfig()
	}
}

func RunFull() error {
	// 1) logger
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "local"
	}
	logger.Init(env)
	defer logger.Sync()

	// 2) viper
	initViper()

	// 3) DB
	db.Init()
	gdb := db.DB

	// 4) AutoMigrate
	if err := gdb.AutoMigrate(&user.User{}); err != nil {
		logger.L.Fatal("auto migrate failed", logger.Err(err))
	}

	// 5) DI
	userRepo := repo.NewUserGormRepo(gdb)
	userSvc := user.NewService(userRepo)
	resolver := &graph.Resolver{UserService: userSvc}
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: resolver}))

	// 6) HTTP
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		AllowMethods:     []string{"POST", "GET", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))
	r.GET("/", func(c *gin.Context) { playground.Handler("GraphQL", "/query").ServeHTTP(c.Writer, c.Request) })
	r.POST("/graphql", func(c *gin.Context) { srv.ServeHTTP(c.Writer, c.Request) })

	addr := viper.GetString("app.addr")
	if addr == "" {
		addr = ":8080"
	}
	logger.L.Info("server started", logger.Str("addr", addr))
	return http.ListenAndServe(addr, r)
}
