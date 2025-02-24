package tcp

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"

	"kyle-redis/logger"
)

// Send a Redis command
func sendCommand(conn net.Conn, args ...string) error {
	// Serialize the command into the RESP format
	command := serializeCommand(args)

	// Send the command to the Redis server
	_, err := conn.Write([]byte(command))
	if err != nil {
		return fmt.Errorf("failed to write to Redis: %w", err)
	}

	// Read the first line of the response
	reader := bufio.NewReader(conn)
	replyType, _ := reader.ReadByte() // Read the first byte to determine the response type

	switch replyType {
	case '+': // Simple String
		line, _ := reader.ReadString('\n')
		logger.Log.Infof("Response: %s\n", strings.TrimSpace(line))

	case '$': // Bulk String
		// Read the length of the bulk string
		lengthLine, _ := reader.ReadString('\n')
		length, err := strconv.Atoi(strings.TrimSpace(lengthLine))
		if err != nil || length < 0 {
			return fmt.Errorf("invalid bulk length")
		}

		// Read the actual bulk string data
		data := make([]byte, length)
		_, err = reader.Read(data)
		if err != nil {
			return fmt.Errorf("failed to read bulk data: %w", err)
		}

		logger.Log.Infof("Bulk Data: %s\n", string(data))

		// Read the trailing \r\n after the bulk data
		reader.Discard(2)

	case '-': // Error
		line, _ := reader.ReadString('\n')
		logger.Log.Infof("Error: %s\n", strings.TrimSpace(line))

	case ':': // Integer
		line, _ := reader.ReadString('\n')
		logger.Log.Infof("Integer: %s\n", strings.TrimSpace(line))

	case '*': // Array
		line, _ := reader.ReadString('\n')
		logger.Log.Infof("Array: %s\n", strings.TrimSpace(line))

	default:
		logger.Log.Errorln("Unknown response type")
	}

	return nil
}

// Serialize a Redis command into the RESP format
func serializeCommand(args []string) string {
	var sb strings.Builder

	// Write the array length
	sb.WriteString(fmt.Sprintf("*%d\r\n", len(args)))

	// Add the length and content of each string
	for _, arg := range args {
		sb.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(arg), arg))
	}

	return sb.String()
}
