package scanner

import "fmt"

type InvalidCharError struct {
	Line, Col uint
	Char      rune
}

func (e *InvalidCharError) Error() string {
	return fmt.Sprintf("Invalid character '%c' at byte line %d column %d", e.Char, e.Line, e.Col)
}
