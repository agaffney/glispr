package reader

import (
	"errors"
	"fmt"
)

var errDivideByZero = errors.New("division by zero")

// ReaderError describes an error caused by unexpected content in the input.
// Line is the 1-based input line. Offset is the 1-based rune offset within Line.
type ReaderError struct {
	Line    int
	Offset  int
	message string
}

func newReaderError(i *Input, format string, args ...any) ReaderError {
	return ReaderError{
		Line:    i.line,
		Offset:  i.linePos,
		message: fmt.Sprintf(format, args...),
	}
}

func (e ReaderError) Error() string {
	return fmt.Sprintf(
		"reader error at line %d, position %d: %s",
		e.Line,
		e.Offset,
		e.message,
	)
}
