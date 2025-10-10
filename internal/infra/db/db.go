package db

import (
	"log"

	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init() {
	env := viper.GetString("env") 
	if e := viper.GetString("APP_ENV"); e != "" {
		env = e                     
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

	if dsn == "" { log.Fatal("empty DSN: check config.database.mysql_local / APP_ENV") }


	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	DB = db
}
