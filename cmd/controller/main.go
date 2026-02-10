package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
)

func main() {
	log.Println("Application started")
	fmt.Println("Press Enter to exit...")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()

	log.Println("Application finished")
}
