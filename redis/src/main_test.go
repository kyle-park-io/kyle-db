package main

import (
	"bufio"
	"net"
	"os"

	"kyle-redis/logger"

	"github.com/spf13/viper"
)

func exampleClientConnection() {
	// 1. Connect to the server.
	conn, err := net.Dial("tcp", ":"+viper.GetString("port.server"))
	if err != nil {
		logger.Log.Fatalf("Error connecting to server: %+v", err)
	}
	defer conn.Close()
	logger.Log.Infoln("Connected to the server. Type commands (e.g. PING, HELLO, TIME, EXIT)")

	// 2. Read input from the user and send it to the server.
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		command := scanner.Text()
		_, err := conn.Write([]byte(command + "\n"))
		if err != nil {
			logger.Log.Errorf("Error sending command: %+v", err)
			return
		}

		// 3. Receive the response from the server.
		response, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			logger.Log.Errorf("Error reading response: %+v", err)
			return
		}
		logger.Log.Infof("Response: %s\n", response)

		// Exit the client loop if the EXIT command is sent.
		if command == "EXIT" {
			logger.Log.Infoln("Exiting client.")
			break
		}
	}
}
