package spl

import (
	"fmt"
	"io"
	"os"

	"github.com/MDx3R/spl/internal/cleaner"
	"github.com/MDx3R/spl/internal/parser"
	"github.com/MDx3R/spl/internal/scanner"
)

type App struct {
	Cleaner cleaner.Cleaner
	Scanner *scanner.Scanner
	Parser  *parser.Parser
}

func errorf(line, col uint, msg string) {
	fmt.Fprintf(os.Stderr, "%d:%d: %s\n", line, col, msg)
}

func parseErrorf(tok scanner.Token, msg string) {
	fmt.Fprintf(os.Stderr, "%d:%d: parse error: %s\n", tok.Line, tok.Col, msg)
}

func NewApp(src io.Reader) *App {
	sc := scanner.NewScanner(src, errorf)
	return &App{
		Cleaner: cleaner.NewCleaner(),
		Scanner: sc,
		Parser:  parser.NewParser(sc, parseErrorf),
	}
}

func (a *App) Parse() *parser.File {
	return a.Parser.Parse()
}
