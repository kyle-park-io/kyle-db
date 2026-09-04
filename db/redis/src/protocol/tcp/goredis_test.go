package tcp

import (
	"os"
	"testing"

	"kyle-redis/client"
	"kyle-redis/internal/config"
)

// go test -v -run TestCallRedisClient
func TestCallRedisClient(t *testing.T) {
	skipUnlessManual(t)

	// Init config
	config.SetEnv("dev")

	// Establish a TCP connection to the Redis server
	conn := client.InitRedisClient("6379")
	defer conn.Conn().Close()

	// Send commands to Redis
	err := callRedisClient(conn)
	if err != nil {
		t.Error(err)
	}
}

// Hand-run experiments: they need a Redis on the default port and a config
// file that only exists inside the image, and `config.SetEnv` calls
// log.Fatal when it cannot find one - which killed the whole test binary.
// Set RUN_MANUAL_TESTS=1 to drive them by hand.
func skipUnlessManual(t *testing.T) {
	t.Helper()
	if os.Getenv("RUN_MANUAL_TESTS") == "" {
		t.Skip("hand-run experiment; set RUN_MANUAL_TESTS=1")
	}
}
