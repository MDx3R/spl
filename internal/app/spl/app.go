package spl

import (
	"fmt"
	"io"
	"os"

	"github.com/MDx3R/spl/internal/cleaner"
	"github.com/MDx3R/spl/internal/scanner"
)

type App struct {
	Cleaner cleaner.Cleaner
	Scanner *scanner.Scanner
}

func errorf(line, col uint, msg string) {
	fmt.Fprintf(os.Stderr, "%d:%d: %s\n", line, col, msg)
}

func NewApp(src io.Reader) *App {
	return &App{
		Cleaner: cleaner.NewCleaner(),
		Scanner: scanner.NewScanner(src, errorf),
	}
}
