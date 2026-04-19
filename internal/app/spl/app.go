package spl

import (
	"io"

	"github.com/MDx3R/spl/internal/cleaner"
	"github.com/MDx3R/spl/internal/scanner"
)

type App struct {
	Cleaner cleaner.Cleaner
	Scanner *scanner.Scanner
}

func NewApp(src io.Reader) *App {
	return &App{
		Cleaner: cleaner.NewCleaner(),
		Scanner: scanner.NewScanner(src),
	}
}
