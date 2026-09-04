package handler

import (
	"fmt"
	"strings"
)

// parseMessage splits a command line into its verb and the rest of the line.
//
// The rest is returned whole, not split into fields: ADDUSER's argument is a
// JSON object, and the previous version of this function rejected any line
// with more than two space-separated parts. It also rejected a bare verb
// followed by nothing but spaces, which is how REAL-TIME arrives from some
// clients.
func parseMessage(message string) (command, arg string, err error) {
	line := strings.TrimSpace(message)
	if line == "" {
		return "", "", fmt.Errorf("empty command")
	}

	verb, rest, found := strings.Cut(line, " ")
	if !found {
		return verb, "", nil
	}
	return verb, strings.TrimSpace(rest), nil
}
