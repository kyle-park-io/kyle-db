package configs

import (
	"errors"
	"os"

	"kyle-postgresql/src/logger"

	"github.com/spf13/viper"
)

func InitConfig() error {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		return errors.New("CONFIG_PATH environment variable is not set")
	}

	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		logger.Log.Errorf("Error reading config file: %v", err)
		return err
	}

	// // check pvc data
	// env := viper.GetString("ENV")
	// if env == "prod" {
	// 	err := utils.CheckPVCData()
	// 	if err != nil {
	// 		logger.Log.Error(err)
	// 		return err
	// 	}
	// }

	return nil
}
