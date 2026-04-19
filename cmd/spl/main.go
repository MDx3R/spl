package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/MDx3R/spl/internal/app/spl"
	"github.com/MDx3R/spl/internal/scanner"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Specify filepath")
		os.Exit(1)
	}

	filePath := os.Args[1]
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Failed to read file: %v\n", err)
		os.Exit(1)
	}

	var buf bytes.Buffer
	app := spl.NewApp(&buf)

	cleaned, warnings, err := app.Cleaner.Clean(string(data))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("--------------------------------")
	fmt.Println(cleaned)
	fmt.Println("--------------------------------")

	for _, w := range warnings {
		fmt.Printf("%s\n", w)
		return
	}

	buf.WriteString(cleaned)
	app.Scanner.Init()

	var tokens []scanner.Token
	for {
		t := app.Scanner.Next()
		if t.Kind == scanner.EOF {
			break
		}
		tokens = append(tokens, t)
	}

	for _, t := range tokens {
		fmt.Println(t)
	}

	fmt.Println("--------------------------------")
	fmt.Println("No lexical errors")
	fmt.Println("--------------------------------")
}
