package handler

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"kyle-redis/client"
	"kyle-redis/logger"
	"kyle-redis/realtime"
)

func BasicHandleConnection(conn net.Conn) {
	defer conn.Close()

	// 4. Read client requests.
	reader := bufio.NewReader(conn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			logger.Log.Errorf("Connection closed: %+v", err)
			return
		}
		message = strings.TrimSpace(message) // Remove newline characters
		logger.Log.Infof("Command received: %s\n", message)

		// 5. Process commands based on client input.
		var response string
		switch message {
		case "PING":
			response = "PONG\n"
		case "HELLO":
			response = "Hello, Client!\n"
		case "TIME":
			response = fmt.Sprintf("Current time: %s\n", time.Now().Format("15:04:05"))
		case "EXIT":
			response = "Goodbye!\n"
			conn.Write([]byte(response))
			logger.Log.Infoln("Closing connection with client.")
			return
		default:
			response = "Unknown command. Available commands: PING, HELLO, TIME, EXIT\n"
		}

		// 6. Send the response to the client.
		_, err = conn.Write([]byte(response))
		if err != nil {
			logger.Log.Errorf("Error sending response: %v", err)
			return
		}
	}
}

func RedisHandleConnection(conn net.Conn) {
	defer conn.Close()

	// redis manager
	manager := realtime.NewUserManager2(client.RedisClient, "active_users2", 30)
	// context
	ctx := context.Background()

	// 4. Read client requests.
	reader := bufio.NewReader(conn)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			logger.Log.Errorf("Connection closed: %+v", err)
			return
		}
		command, message, err := parseMessage(msg) // Remove newline characters
		if err != nil {
			logger.Log.Errorf("Connection closed: %+v", err)
			return
		}
		logger.Log.Infof("Command received: %s, %s\n", command, message)

		// 5. Process commands based on client input.
		var response string
		switch command {
		case "PING":
			response = "PONG\n"
		case "HELLO":
			response = "Hello, Client!\n"
		case "TIME":
			response = fmt.Sprintf("Current time: %s\n", time.Now().Format("15:04:05"))
		case "EXIT":
			response = "Goodbye!\n"
		case "ADDUSER":
			err := manager.AddUser(ctx, message)
			if err != nil {
				logger.Log.Errorf("Error sending response: %v", err)
				response = fmt.Sprintf("%v\n", err)
				_, err = conn.Write([]byte(response))
				if err != nil {
					logger.Log.Errorf("Error sending response: %v", err)
					return
				}
			}
			response = fmt.Sprintf("%s\n", conn.RemoteAddr().String())
			logger.Log.Infof("Successfully Update User Aliving: %s\n", conn.RemoteAddr().String())
		case "REAL-TIME":
			err = manager.CleanUpExpiredUsers(ctx)
			if err != nil {
				logger.Log.Errorf("Error sending response: %v", err)
				response = fmt.Sprintf("%v\n", err)
				_, err = conn.Write([]byte(response))
				if err != nil {
					logger.Log.Errorf("Error sending response: %v", err)
					return
				}
			}
			count, err := manager.GetUserCount(ctx)
			if err != nil {
				logger.Log.Errorf("Error sending response: %v", err)
				response = fmt.Sprintf("%v\n", err)
				_, err = conn.Write([]byte(response))
				if err != nil {
					logger.Log.Errorf("Error sending response: %v", err)
					return
				}
			}
			response = fmt.Sprintf("%d\n", count)
			logger.Log.Infof("Current Real-Time User Count: %d\n", count)
		default:
			response = "Unknown command. Available commands: ADDUSER, REAL-TIME, PING, HELLO, TIME, EXIT\n"
		}

		// 6. Send the response to the client.
		_, err = conn.Write([]byte(response))
		if err != nil {
			logger.Log.Errorf("Error sending response: %v", err)
			return
		}
	}
}
