package main

import (
	"kyle-postgresql/src/configs"

	"github.com/spf13/viper"
)

func main() {

	// env
	env := "dev"
	viper.Set("ENV", env)

	switch env {
	case "dev":
		configs.SetDevEnv()
	case "prod":
		configs.SetProdEnv()
	}
}
