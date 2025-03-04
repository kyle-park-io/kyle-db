package config

import (
	"os"

	"kyle-redis/logger"

	"github.com/spf13/viper"
)

func SetEnv(env string) {
	if env == "prod" {
		os.Setenv("ROOT_PATH", "/app")
		viper.Set("ROOT_PATH", "/app")
	} else {
		os.Setenv("ROOT_PATH", "/home/kyle/code/kyle-db/db/redis/src")
		viper.Set("ROOT_PATH", "/home/kyle/code/kyle-db/db/redis/src")
	}

	os.Setenv("LOG_PATH", viper.GetString("ROOT_PATH")+"/logs")
	viper.Set("LOG_PATH", viper.GetString("ROOT_PATH")+"/logs")

	logger.InitLogger()
	logger.Log.Info("Hi! i'm kyle redis.")

	os.Setenv("CONFIG_PATH", viper.GetString("ROOT_PATH")+"/configs/config.yaml")
	if err := InitConfig(); err != nil {
		logger.Log.Fatalf("Check Errors, %v", err)
	}
}
