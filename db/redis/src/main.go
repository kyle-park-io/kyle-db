package main

import (
	"net"
	"os"

	"kyle-redis/client"
	"kyle-redis/handler"
	"kyle-redis/internal/config"
	"kyle-redis/logger"

	"github.com/spf13/viper"
)

func main() {

	// APP_ENV picks the port block in configs/config.yaml. It defaults to
	// prod, which is what the image runs; the override is what makes a local
	// run possible.
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "prod"
	}
	// config
	config.SetEnv(env)
	// InitLogger's own `defer Sync()` fired before anything was logged; the
	// flush belongs to whoever outlives the logger, which is main.
	defer logger.Sync()
	client.Init()

	port := viper.GetString("port.server." + env)
	if port == "" {
		logger.Log.Fatalln("Check env: port.server : ", env)
	}

	// 1. Start the server by listening on a specific port.
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		logger.Log.Fatalf("Error starting server: %+v", err)
	}
	defer listener.Close()
	logger.Log.Infof("Server is listening on port %s..", port)

	for {
		// 2. Wait for client connections.
		conn, err := listener.Accept()
		if err != nil {
			logger.Log.Errorf("Error accepting connection: %+v", err)
			continue
		}
		// One line per accepted connection, and the ticker opens one every
		// couple of seconds forever, so this is Debug rather than Info.
		logger.Log.Debugw("client connected", "peer", conn.RemoteAddr().String())

		// 3. Handle each client connection in a separate goroutine.
		// go handler.BasicHandleConnection(conn)
		go handler.RedisHandleConnection(conn)
	}
}
