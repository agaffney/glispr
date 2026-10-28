package reader

import (
	"fmt"
)

// ReaderError describes an error caused by unexpected content in the input.
// Line is the 1-based input line. Offset is the 1-based rune offset within Line.
type ReaderError struct {
	Line    int
	Offset  int
	message string
}

func newReaderError(i *Input, format string, args ...any) ReaderError {
	return ReaderError{
		Line: i.line,
		// linePos is 0 right after a newline, such as when the input ends
		// there, but offsets are 1-based
		Offset:  max(i.linePos, 1),
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
