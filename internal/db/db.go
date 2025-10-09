package db

import (
	"log"

	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init() {
	env := viper.GetString("env") // 默认读自 config.yaml
	if e := viper.GetString("APP_ENV"); e != "" {
		env = e                     // 优先使用环境变量
	}

	var dsn string
	switch env {
	case "docker":
		dsn = viper.GetString("database.mysql_docker")
	case "dev", "local":
		dsn = viper.GetString("database.mysql_local")
	default:
		dsn = viper.GetString("database.mysql_local")
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	DB = db
}
