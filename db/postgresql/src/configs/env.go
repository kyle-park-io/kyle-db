package configs

import (
	"os"

	"kyle-postgresql/src/logger"

	"github.com/spf13/viper"
)

func SetDevEnv() {
	// root
	os.Setenv("ROOT_PATH", "/home/kyle/code/kyle-db/db/postgresql")
	viper.Set("ROOT_PATH", "/home/kyle/code/kyle-db/db/postgresql")

	// logger
	os.Setenv("LOG_PATH", viper.GetString("ROOT_PATH")+"/logs")
	viper.Set("LOG_PATH", viper.GetString("ROOT_PATH")+"/logs")
	logger.InitLogger()
	logger.Log.Info("Hi! i'm kyle postgresql.")

	// viper
	os.Setenv("CONFIG_PATH", viper.GetString("ROOT_PATH")+"/configs/config.yaml")
	if err := InitConfig(); err != nil {
		logger.Log.Fatalf("Check Errors, %v", err)
	}
}

func SetProdEnv() {
	// root
	os.Setenv("ROOT_PATH", "/app")
	viper.Set("ROOT_PATH", "/app")

	// logger
	os.Setenv("LOG_PATH", viper.GetString("ROOT_PATH")+"/logs")
	viper.Set("LOG_PATH", viper.GetString("ROOT_PATH")+"/logs")
	logger.InitLogger()
	logger.Log.Info("Hi! i'm kyle postgresql.")

	// viper
	os.Setenv("CONFIG_PATH", viper.GetString("ROOT_PATH")+"/configs/config.yaml")
	if err := InitConfig(); err != nil {
		logger.Log.Fatalf("Check Errors, %v", err)
	}
}
