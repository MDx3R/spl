package cleaner

import "fmt"

type InvalidCharError struct {
	Position int
	Char     rune
}

func (e *InvalidCharError) Error() string {
	return fmt.Sprintf("Invalid character '%c' at byte position %d", e.Char, e.Position)
}
