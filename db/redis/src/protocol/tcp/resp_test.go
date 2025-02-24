package tcp

import (
	"testing"

	"kyle-redis/client"
	"kyle-redis/internal/config"
)

// go test -v -run TestSendCommand
func TestSendCommand(t *testing.T) {

	// Init config
	config.SetEnv("dev")

	// Establish a TCP connection to the Redis server
	conn := client.InitTCPClient("6379")
	defer conn.Close()

	// Send commands to Redis
	sendCommand(conn, "PING")
	sendCommand(conn, "SET", "key", "value")
	sendCommand(conn, "GET", "key")
}
