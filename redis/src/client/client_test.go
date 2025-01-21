package client

import (
	"testing"

	"kyle-redis/internal/config"

	"github.com/spf13/viper"
)

// go test -v -run TestInitRedisClient
func TestInitRedisClient(t *testing.T) {

	config.SetEnv("dev")

	port := viper.GetString("port.redis")
	if port == "" {
		port = "6379"
	}

	client := InitRedisClient(port)
	t.Logf("client: %+v", client)
}

// go test -v -run TestInitTCPClient
func TestInitTCPClient(t *testing.T) {

	config.SetEnv("dev")

	port := viper.GetString("port.redis")
	if port == "" {
		port = "6379"
	}

	client := InitTCPClient(port)
	t.Logf("client: %+v", client)
}
