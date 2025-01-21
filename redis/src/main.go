package main

import (
	"net"

	"kyle-redis/client"
	"kyle-redis/handler"
	"kyle-redis/internal/config"
	"kyle-redis/logger"

	"github.com/spf13/viper"
)

func main() {

	env := "prod"
	// config
	config.SetEnv(env)
	client.Init()

	port := viper.GetString("port.server")
	if port == "" {
		logger.Log.Fatalln("Check env: port.server")
	}

	// 1. Start the server by listening on a specific port.
	listener, err := net.Listen("tcp", "localhost:"+port)
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
		logger.Log.Infoln("Client connected: ", conn.RemoteAddr())

		// 3. Handle each client connection in a separate goroutine.
		// go handler.BasicHandleConnection(conn)
		go handler.RedisHandleConnection(conn)
	}
}
