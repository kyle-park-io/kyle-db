package handler

import (
	"bufio"
	"strings"
	"testing"
)

// go test -v -run TestReadInput
func TestReadInput(t *testing.T) {
	// Mock input
	mockInput := "Hello, Go!\n"
	mockReader := strings.NewReader(mockInput)

	// Replace os.Stdin with the mock reader
	scanner := bufio.NewScanner(mockReader)

	// Read input
	if scanner.Scan() {
		input := scanner.Text()
		expected := "Hello, Go!"
		if input != expected {
			t.Errorf("Expected %q but got %q", expected, input)
		}
	} else {
		t.Errorf("Failed to scan input")
	}
}
