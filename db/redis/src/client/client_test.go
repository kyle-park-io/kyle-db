package client

import (
	"os"
	"testing"

	"kyle-redis/internal/config"

	"github.com/spf13/viper"
)

// go test -v -run TestInitRedisClient
func TestInitRedisClient(t *testing.T) {
	skipUnlessManual(t)

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
	skipUnlessManual(t)

	config.SetEnv("dev")

	port := viper.GetString("port.redis")
	if port == "" {
		port = "6379"
	}

	client := InitTCPClient(port)
	t.Logf("client: %+v", client)
}

// These are hand-run experiments, not checks: they need a Redis on the
// default port and assert nothing. `go test ./...` ran them and failed on a
// config file that only exists inside the image. Set RUN_MANUAL_TESTS=1 to
// drive them by hand.
func skipUnlessManual(t *testing.T) {
	t.Helper()
	if os.Getenv("RUN_MANUAL_TESTS") == "" {
		t.Skip("hand-run experiment; set RUN_MANUAL_TESTS=1")
	}
}
