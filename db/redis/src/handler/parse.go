package handler

import (
	"fmt"
	"strings"
)

func parseMessage(message string) (command, addr string, err error) {
	parts := strings.Fields(message)
	if len(parts) == 1 {
		return parts[0], "", nil
	} else if len(parts) == 2 {
		return parts[0], parts[1], nil
	} else {
		return "", "", fmt.Errorf("invalid message format")
	}
}
