package reader

import (
	"bufio"
	"errors"
	"io"
)

type Input struct {
	r           *bufio.Reader
	line        int
	linePos     int
	prevLinePos int
}

func NewInput(r io.Reader) (*Input, error) {
	if r == nil {
		return nil, errors.New("nil io.Reader")
	}
	return &Input{
		r: bufio.NewReader(r),
	}, nil
}

func (i *Input) ReadRune() (rune, error) {
	// Reset line number if zero
	i.line = max(i.line, 1)
	c, _, err := i.r.ReadRune()
	if err != nil {
		return 0, err
	}
	// Increment line position
	i.linePos++
	// Check for newline
	if c == '\n' {
		i.line++
		i.prevLinePos = i.linePos
		i.linePos = 0
	}
	return c, nil
}

func (i *Input) UnreadRune() error {
	err := i.r.UnreadRune()
	if err != nil {
		return err
	}
	// Decrement line pos
	i.linePos--
	if i.linePos < 1 {
		i.line--
		i.linePos = i.prevLinePos
	}
	return err
}
