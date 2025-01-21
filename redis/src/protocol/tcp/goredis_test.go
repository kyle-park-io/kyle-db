package tcp

import (
	"testing"

	"kyle-redis/client"
	"kyle-redis/internal/config"
)

// go test -v -run TestCallRedisClient
func TestCallRedisClient(t *testing.T) {

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
