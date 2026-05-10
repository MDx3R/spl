package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/MDx3R/spl/internal/app/spl"
	"github.com/MDx3R/spl/internal/parser"
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
	application := spl.NewApp(&buf)

	cleaned, warnings, err := application.Cleaner.Clean(string(data))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	for _, w := range warnings {
		fmt.Printf("Warning: %s\n", w)
	}

	buf.WriteString(cleaned)

	file := application.Parse()
	if file == nil {
		fmt.Println("Parse returned nil")
		os.Exit(1)
	}

	pr := parser.NewAstPrinter()
	pr.VisitFile(file)
	fmt.Print(pr.String())
}
