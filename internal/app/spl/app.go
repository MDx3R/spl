package spl

import "github.com/MDx3R/spl/internal/cleaner"

type App struct {
	Cleaner cleaner.Cleaner
}

func NewApp() *App {
	return &App{
		Cleaner: cleaner.NewCleaner(),
	}
}
