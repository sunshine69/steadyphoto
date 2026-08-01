package main

import (
	"fmt"
	"os"
	"steadyphoto/internal/processor"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run ./cmd/debug-fn/main.go <filename>")
		fmt.Println("  go run ./cmd/debug-fn/main.go 59e4a97ccf3143c190a05727b121250c.jpg")
		fmt.Println("  go run ./cmd/debug-fn/main.go received_2459394844477772.jpeg")
		return
	}

	filename := os.Args[1]

	t, remaining, err := processor.ExtractDateFromString(filename)
	fmt.Printf("Filename: %s\n", filename)
	fmt.Printf("  Parsed date: %v, IsZero: %v\n", t, t.IsZero())
	fmt.Printf("  Remaining: %s\n", remaining)
	fmt.Printf("  Error: %v\n", err)
}
