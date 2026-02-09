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

	cleaned, warnings, err := app.Cleaner.Clean(string(data))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(cleaned)

	for _, w := range warnings {
		fmt.Printf("%s\n", w)
	}
}
