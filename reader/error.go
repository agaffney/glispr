package reader

import (
	"fmt"
)

type InvalidCharError struct {
	ch      rune
	line    int
	linePos int
}

func (e InvalidCharError) Error() string {
	return fmt.Sprintf(
		"invalid character at line %d, position %d: %q (%#x)",
		e.line,
		e.linePos,
		e.ch,
		e.ch,
	)
}
