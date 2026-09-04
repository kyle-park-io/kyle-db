package config

import (
	"os"

	"kyle-redis/logger"

	"github.com/spf13/viper"
)

// SetEnv resolves where the config and the logs live.
//
// ROOT_PATH from the environment wins. Without that override the only two
// answers were "/app", which exists inside the image and nowhere else, and a
// hardcoded absolute path into one developer's home directory - so the
// server could not be run anywhere but in its container, which is exactly
// where it is hardest to check what it does.
func SetEnv(env string) {
	root := os.Getenv("ROOT_PATH")
	if root == "" {
		if env == "prod" {
			root = "/app"
		} else {
			// Relative to the working directory, so a checkout runs from
			// wherever it happens to be.
			root = "."
		}
	}

	os.Setenv("ROOT_PATH", root)
	viper.Set("ROOT_PATH", root)

	os.Setenv("LOG_PATH", root+"/logs")
	viper.Set("LOG_PATH", root+"/logs")

	logger.InitLogger()

	os.Setenv("CONFIG_PATH", root+"/configs/config.yaml")
	if err := InitConfig(); err != nil {
		logger.Log.Fatalf("Check Errors, %v", err)
	}

	logger.Log.Infow("configuration loaded", "env", env, "root", root)
}
