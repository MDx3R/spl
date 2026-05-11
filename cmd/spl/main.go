package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/MDx3R/spl/internal/app/spl"
	"github.com/MDx3R/spl/internal/parser"
	"github.com/MDx3R/spl/internal/semantic"
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

	// LR3: parse → AST
	file := application.Parse()
	if file == nil {
		fmt.Println("Parse returned nil")
		os.Exit(1)
	}

	pr := parser.NewAstPrinter()
	pr.VisitFile(file)
	fmt.Print(pr.String())

	// LR4: semantic analysis → symbol table + errors
	analyzer := semantic.NewAnalyzer()
	analyzer.AnalyzeFile(file)
	fmt.Print(analyzer.FormatTable())
	fmt.Print(analyzer.FormatErrors())

	// LR4: triad IR generation (consumes the analysis result)
	emitter := semantic.NewTriadEmitter()
	emitter.EmitFile(file, analyzer.Result())
	fmt.Print(emitter.String())
}
