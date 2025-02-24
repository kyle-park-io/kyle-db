package handler

import (
	"bufio"
	"fmt"
	"os"
)

func readInput() string {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Enter text: ")
	scanner.Scan()
	return scanner.Text()
}
