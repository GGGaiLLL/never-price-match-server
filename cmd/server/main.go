package main

import (
	"fmt"
	"log"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"

	"never-price-match-server/internal/db"
	"never-price-match-server/internal/model"
	"never-price-match-server/internal/router"
	"never-price-match-server/pkg/logger"
)

func loadConfig() {
	viper.SetConfigFile("configs/config.yaml")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("read config error: %v", err)
	}
	env := viper.GetString("env")
	logger.Init(env)
	logger.L.Info("config loaded", logger.Field("env", env))

}

func main() {
	loadConfig()

	// 热加载
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		logger.L.Info("config changed", logger.Field("file", e.Name))
		logger.Sync()
		logger.Init(viper.GetString("env"))
		// 如需动态重连DB或刷新限流参数，在此处理
	})

	db.Init()
	db.DB.AutoMigrate(&model.User{})

	r := router.Setup()
	port := viper.GetInt("server.port")
	logger.L.Info("server starting", logger.Field("port", port))
	if err := r.Run(fmt.Sprintf(":%d", port)); err != nil {
		logger.L.Fatal("server quit", logger.Err(err))
	}
}
