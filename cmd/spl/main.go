package main

import (
	"fmt"
	"os"

	"github.com/MDx3R/spl/internal/app/spl"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Specify filepath")
		os.Exit(1)
	}

	app := spl.NewApp()

	filePath := os.Args[1]
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Failed to read file: %v\n", err)
		os.Exit(1)
	}

	cleaned := app.Cleaner.Clean(string(data))

	fmt.Println(cleaned)
}
